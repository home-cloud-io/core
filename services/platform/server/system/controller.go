package system

import (
	"context"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	dv1 "github.com/home-cloud-io/core/api/platform/daemon/v1"
	v1 "github.com/home-cloud-io/core/api/platform/server/v1"
	"github.com/home-cloud-io/core/services/platform/server/async"
	k8sclient "github.com/home-cloud-io/core/services/platform/server/k8s-client"
	kvclient "github.com/home-cloud-io/core/services/platform/server/kv-client"

	"github.com/containers/image/v5/docker"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/robfig/cron/v3"
	kvv1 "github.com/steady-bytes/draft/api/core/registry/key_value/v1"
	"github.com/steady-bytes/draft/pkg/chassis"
	"golang.org/x/mod/semver"
	"golang.org/x/sync/errgroup"
)

type (
	Controller interface {
		Daemon
		OS
		Containers
		Device
	}
	Daemon interface {
		// ShutdownHost will shutdown the host machine running Home Cloud.
		ShutdownHost() error
		// RestartHost will restart the host machine running Home Cloud
		RestartHost() error
		// ChangeDaemonVersion will update the NixOS config with a new Daemon version
		// and switch to it.
		ChangeDaemonVersion(cmd *dv1.ChangeDaemonVersionCommand) error
		// AddMdnsHost adds a host to the avahi mDNS server managed by the daemon
		AddMdnsHost(hostname string) error
		// RemoveMdnsHost removes a host to the avahi mDNS server managed by the daemon
		RemoveMdnsHost(hostname string) error
		// UploadFileStream will stream a file in chunks as an upload to the daemon
		UploadFileStream(ctx context.Context, logger chassis.Logger, buf io.Reader, fileId, fileName string) (string, error)
	}
	OS interface {
		// CheckForOSUpdates will run the Nix commands to check for any NixOS updates to install.
		CheckForOSUpdates(ctx context.Context, logger chassis.Logger) (*v1.CheckForSystemUpdatesResponse, error)
		// InstallOSUpdate will install the update (if any) generated by CheckForOSUpdates.
		InstallOSUpdate() error
		// AutoUpdateOS will check for and install any OS (including Daemon) updates on a schedule. It is
		// designed to be called at bootup.
		AutoUpdateOS(logger chassis.Logger)
		// UpdateOS will check for and install any OS (including Daemon) updates one time.
		UpdateOS(ctx context.Context, logger chassis.Logger) error
	}
	Containers interface {
		// SetSystemImage will update the image for a system container.
		SetSystemImage(cmd *dv1.SetSystemImageCommand) error
		// CheckForContainerUpdates will compare current system container images against the latest ones
		// available and return the result.
		CheckForContainerUpdates(ctx context.Context, logger chassis.Logger) ([]*v1.ImageVersion, error)
		// AutoUpdateContainers will check for and install any container updates on a schedule. It is
		// designed to be called at bootup.
		AutoUpdateContainers(logger chassis.Logger)
		// UpdateContainers will check for and install any container updates one time.
		UpdateContainers(ctx context.Context, logger chassis.Logger) error
	}
	Device interface {
		// GetServerSettings returns the current server settings after filtering out the
		// admin username and password.
		GetServerSettings(ctx context.Context) (*v1.DeviceSettings, error)
		// SetServerSettings updates the settings on the server with the given values
		SetServerSettings(ctx context.Context, logger chassis.Logger, settings *v1.DeviceSettings) error
		// IsDeviceSetup checks if the device is already setup by checking if the DEFAULT_DEVICE_SETTINGS_KEY key exists in the key-value store
		// with the default settings model
		IsDeviceSetup(ctx context.Context) (bool, error)
		// InitializeDevice initializes the device with the given settings. It first checks if the device is already setup
		// Uses the user-provided password to set the password for the "admin" user on the device
		// and save the remaining settings in the key-value store
		InitializeDevice(ctx context.Context, logger chassis.Logger, settings *v1.DeviceSettings) error
		// Login receives a username and password and returns a token.
		//
		// NOTE: the token is unimplemented so this only checks the validity of the password for now
		Login(ctx context.Context, username, password string) (string, error)
	}

	controller struct {
		k8sclient        k8sclient.System
		systemUpdateLock sync.Mutex
		broadcaster      async.Broadcaster
	}
	fileChunk struct {
		index int
		data  []byte
	}
)

func NewController(logger chassis.Logger, broadcaster async.Broadcaster) Controller {
	config := chassis.GetConfig()
	config.SetDefault(osAutoUpdateCronConfigKey, "0 1 * * *")
	config.SetDefault(containerAutoUpdateCronConfigKey, "0 2 * * *")
	return &controller{
		k8sclient:        k8sclient.NewClient(logger),
		systemUpdateLock: sync.Mutex{},
		broadcaster:      broadcaster,
	}
}

const (
	ErrDeviceAlreadySetup          = "device already setup"
	ErrFailedToCreateSettings      = "failed to create settings"
	ErrFailedToSaveSettings        = "failed to save settings"
	ErrFailedToGetSettings         = "failed to get settings"
	ErrFailedToSetSettings         = "failed to save settings"
	ErrFailedToBuildSeedGetRequest = "failed to build get request for seed"
	ErrFailedToGetSeedValue        = "failed to get seed value"
	ErrFailedToUnmarshalSeedValue  = "failed to unmarshal seed value"

	homeCloudCoreRepo                = "https://github.com/home-cloud-io/core"
	homeCloudCoreTrunk               = "main"
	daemonTagPath                    = "refs/tags/services/platform/daemon/"
	osAutoUpdateCronConfigKey        = "server.updates.os_auto_update_cron"
	containerAutoUpdateCronConfigKey = "server.updates.containers_auto_update_cron"
)

// DAEMON

func (c *controller) ShutdownHost() error {
	err := com.Send(&dv1.ServerMessage{
		Message: &dv1.ServerMessage_Shutdown{},
	})
	if err != nil {
		return err
	}
	return nil
}

func (c *controller) RestartHost() error {
	err := com.Send(&dv1.ServerMessage{
		Message: &dv1.ServerMessage_Restart{},
	})
	if err != nil {
		return err
	}
	return nil
}

func (c *controller) ChangeDaemonVersion(cmd *dv1.ChangeDaemonVersionCommand) error {
	err := com.Send(&dv1.ServerMessage{
		Message: &dv1.ServerMessage_ChangeDaemonVersionCommand{
			ChangeDaemonVersionCommand: cmd,
		},
	})
	if err != nil {
		return err
	}
	return nil
}

func (c *controller) InstallOSUpdate() error {
	err := com.Send(&dv1.ServerMessage{
		Message: &dv1.ServerMessage_InstallOsUpdateCommand{},
	})
	if err != nil {
		return err
	}
	return nil
}

func (c *controller) SetSystemImage(cmd *dv1.SetSystemImageCommand) error {
	err := com.Send(&dv1.ServerMessage{
		Message: &dv1.ServerMessage_SetSystemImageCommand{
			SetSystemImageCommand: cmd,
		},
	})
	if err != nil {
		return err
	}
	return nil
}

func (c *controller) AddMdnsHost(hostname string) error {
	err := com.Send(&dv1.ServerMessage{
		Message: &dv1.ServerMessage_AddMdnsHostCommand{
			AddMdnsHostCommand: &dv1.AddMdnsHostCommand{
				Hostname: hostname,
			},
		},
	})
	if err != nil {
		return err
	}
	return nil
}

func (c *controller) RemoveMdnsHost(hostname string) error {
	err := com.Send(&dv1.ServerMessage{
		Message: &dv1.ServerMessage_RemoveMdnsHostCommand{
			RemoveMdnsHostCommand: &dv1.RemoveMdnsHostCommand{
				Hostname: hostname,
			},
		},
	})
	if err != nil {
		return err
	}
	return nil
}

func (c *controller) UploadFileStream(ctx context.Context, logger chassis.Logger, buf io.Reader, fileId, fileName string) (string, error) {
	logger.Info("uploading file")
	var listenerErr error
	done := make(chan bool)
	go func(){
		listenerErr = async.RegisterListener(ctx, c.broadcaster, &async.ListenerOptions[*dv1.UploadFileReady]{
			Callback: func(event *dv1.UploadFileReady) (bool, error) {
				if event.Id == fileId {
					done <- true
					return true, nil
				}
				return false, nil
			},
			Timeout: 5 * time.Second,
		}).Listen(ctx)
		if listenerErr != nil {
			done <- true
		}
	}()

	// prepare upload to daemon
	err := com.Send(&dv1.ServerMessage{
		Message: &dv1.ServerMessage_UploadFileRequest{
			UploadFileRequest: &dv1.UploadFileRequest{
				Data: &dv1.UploadFileRequest_Info{
					Info: &dv1.FileInfo{
						FileId:   fileId,
						FilePath: fileName,
					},
				},
			},
		},
	})
	if err != nil {
		logger.WithError(err).Error("failed to ready daemon for file upload")
		return fileId, err
	}
	logger.Info("waiting for done signal")
	<-done
	if listenerErr != nil {
		logger.WithError(listenerErr).Error("failed to ready daemon for file upload")
		return fileId, listenerErr
	}
	logger.Info("daemon ready for file upload")

	// chunk file and upload
	err = c.streamFile(ctx, logger, buf, fileId)
	if err != nil {
		logger.WithError(err).Error("failed to upload chunked file")
		return fileId, err
	}

	return fileId, nil
}

// OS

func (c *controller) CheckForOSUpdates(ctx context.Context, logger chassis.Logger) (*v1.CheckForSystemUpdatesResponse, error) {
	if !c.systemUpdateLock.TryLock() {
		logger.Warn("call to check for system updates while another check is already in progress")
		return nil, fmt.Errorf("system update check already in progress")
	}
	defer c.systemUpdateLock.Unlock()

	var (
		response = &v1.CheckForSystemUpdatesResponse{}
	)

	// get the os update diff from the daemon
	done := make(chan bool)
	go async.RegisterListener(ctx, c.broadcaster, &async.ListenerOptions[*dv1.OSUpdateDiff]{
		Callback: func(event *dv1.OSUpdateDiff) (bool, error) {
			response.OsDiff = event.Description
			done <- true
			return true, nil
		},
	}).Listen(ctx)
	err := com.Send(&dv1.ServerMessage{
		Message: &dv1.ServerMessage_RequestOsUpdateDiff{},
	})
	if err != nil {
		return nil, err
	}
	<-done

	// get the current daemon version from the daemon
	done = make(chan bool)
	go async.RegisterListener(ctx, c.broadcaster, &async.ListenerOptions[*dv1.CurrentDaemonVersion]{
		Callback: func(event *dv1.CurrentDaemonVersion) (bool, error) {
			response.DaemonVersions = &v1.DaemonVersions{
				Current: &v1.DaemonVersion{
					Version:    event.Version,
					VendorHash: event.VendorHash,
					SrcHash:    event.SrcHash,
				},
			}
			done <- true
			return true, nil
		},
	}).Listen(ctx)
	err = com.Send(&dv1.ServerMessage{
		Message: &dv1.ServerMessage_RequestCurrentDaemonVersion{},
	})
	if err != nil {
		return nil, err
	}

	// get latest available daemon version
	latest, err := getLatestDaemonVersion()
	if err != nil {
		return nil, err
	}
	response.DaemonVersions.Latest = latest

	return response, nil
}

func (c *controller) AutoUpdateOS(logger chassis.Logger) {
	cr := cron.New()
	f := func() {
		ctx := context.Background()
		err := c.UpdateOS(ctx, logger)
		if err != nil {
			logger.WithError(err).Error("failed to run auto os update job")
		}
	}
	cron := chassis.GetConfig().GetString(osAutoUpdateCronConfigKey)
	logger.WithField("cron", cron).Info("setting os auto-update interval")
	_, err := cr.AddFunc(cron, f)
	if err != nil {
		logger.WithError(err).Panic("failed to initialize auto-update for os")
	}
	cr.Start()
}

func (u *controller) UpdateOS(ctx context.Context, logger chassis.Logger) error {
	logger.Info("updating os")
	settings := &v1.DeviceSettings{}
	err := kvclient.Get(ctx, kvclient.DEFAULT_DEVICE_SETTINGS_KEY, settings)
	if err != nil {
		logger.WithError(err).Error("failed to get device settings")
		return err
	}

	if !settings.AutoUpdateOs {
		logger.Info("auto update sytem not enabled")
		return nil
	}

	updates, err := u.CheckForOSUpdates(ctx, logger)
	if err != nil {
		logger.WithError(err).Error("failed to check for system updates")
		return err
	}

	// if the daemon needs updating, install it along with the os updates
	// otherwise just install the os update
	if updates.DaemonVersions.Current != updates.DaemonVersions.Latest {
		err = com.Send(&dv1.ServerMessage{
			Message: &dv1.ServerMessage_ChangeDaemonVersionCommand{
				ChangeDaemonVersionCommand: &dv1.ChangeDaemonVersionCommand{
					Version:    updates.DaemonVersions.Latest.Version,
					VendorHash: updates.DaemonVersions.Latest.VendorHash,
					SrcHash:    updates.DaemonVersions.Latest.SrcHash,
				},
			},
		})
	} else {
		err = com.Send(&dv1.ServerMessage{
			Message: &dv1.ServerMessage_InstallOsUpdateCommand{},
		})
	}
	if err != nil {
		logger.WithError(err).Error("failed to install system update")
		return err
	}

	return nil
}

// CONTAINERS

func (c *controller) CheckForContainerUpdates(ctx context.Context, logger chassis.Logger) ([]*v1.ImageVersion, error) {
	var (
		images []*v1.ImageVersion
	)

	// populate current versions (from k8s)
	images, err := c.k8sclient.CurrentImages(ctx)
	if err != nil {
		logger.WithError(err).Error("failed to get current container versions")
		return nil, err
	}

	// populate latest versions (from registry)
	images, err = getLatestImageTags(ctx, images)
	if err != nil {
		logger.WithError(err).Error("failed to get latest image versions")
		return nil, err
	}

	return images, err
}

func (c *controller) AutoUpdateContainers(logger chassis.Logger) {
	cr := cron.New()
	f := func() {
		ctx := context.Background()
		err := c.UpdateContainers(ctx, logger)
		if err != nil {
			logger.WithError(err).Error("failed to run auto container update job")
		}
	}
	cron := chassis.GetConfig().GetString(containerAutoUpdateCronConfigKey)
	logger.WithField("cron", cron).Info("setting container auto-update interval")
	_, err := cr.AddFunc(cron, f)
	if err != nil {
		logger.WithError(err).Panic("failed to initialize auto-update for system containers")
	}
	cr.Start()
}

func (c *controller) UpdateContainers(ctx context.Context, logger chassis.Logger) error {
	logger.Info("updating containers")
	settings := &v1.DeviceSettings{}
	err := kvclient.Get(ctx, kvclient.DEFAULT_DEVICE_SETTINGS_KEY, settings)
	if err != nil {
		logger.WithError(err).Error("failed to get device settings")
		return err
	}

	// TODO: should this be a different setting?
	if !settings.AutoUpdateOs {
		logger.Info("auto update sytem not enabled")
		return nil
	}

	images, err := c.CheckForContainerUpdates(ctx, logger)
	if err != nil {
		logger.WithError(err).Error("failed to check for system container updates")
		return err
	}

	for _, image := range images {
		log := logger.WithFields(chassis.Fields{
			"image":           image.Image,
			"current_version": image.Current,
			"latest_version":  image.Latest,
		})
		if semver.Compare(image.Latest, image.Current) == 1 {
			log.Info("updating image")
			err := com.Send(&dv1.ServerMessage{
				Message: &dv1.ServerMessage_SetSystemImageCommand{
					SetSystemImageCommand: &dv1.SetSystemImageCommand{
						CurrentImage:   fmt.Sprintf("%s:%s", image.Image, image.Current),
						RequestedImage: fmt.Sprintf("%s:%s", image.Image, image.Latest),
					},
				},
			})
			if err != nil {
				log.WithError(err).Error("failed to update system container image")
				// don't return, try to update other containers
			}
			// TODO: this is a hack, should really be event-driven
			time.Sleep(3 * time.Second)
		} else {
			log.Info("no update needed")
		}
	}

	return nil
}

// DEVICE

func (c *controller) GetServerSettings(ctx context.Context) (*v1.DeviceSettings, error) {
	settings := &v1.DeviceSettings{}
	err := kvclient.Get(ctx, kvclient.DEFAULT_DEVICE_SETTINGS_KEY, settings)
	if err != nil {
		return nil, errors.New(ErrFailedToGetSettings)
	}

	settings.AdminUser.Password = "" // don't return the password

	return settings, nil
}

func (c *controller) SetServerSettings(ctx context.Context, logger chassis.Logger, settings *v1.DeviceSettings) error {
	// set the device settings on the host (via the daemon)
	done := make(chan bool)
	var listenerErr error
	go func() {
		listenerErr = async.RegisterListener(ctx, c.broadcaster, &async.ListenerOptions[*dv1.DeviceInitialized]{
			Callback: func(event *dv1.DeviceInitialized) (bool, error) {
				done<-true
				if event.Error != nil {
					return true, fmt.Errorf(event.Error.Error)
				}
				return true, nil
			},
		}).Listen(ctx)
	}()
	err := com.Send(&dv1.ServerMessage{
		Message: &dv1.ServerMessage_InitializeDeviceCommand{
			InitializeDeviceCommand: &dv1.InitializeDeviceCommand{
				User: &dv1.SetUserPasswordCommand{
					// TODO: Support multiple users? Right now the username "admin" is hardcoded into NixOS.
					Username: "admin",
					Password: settings.AdminUser.Password,
				},
				TimeZone: &dv1.SetTimeZoneCommand{
					TimeZone: settings.Timezone,
				},
			},
		},
	})
	if err != nil {
		return err
	}
	<-done
	if listenerErr != nil {
		return listenerErr
	}

	// salt and hash given password if set
	// otherwise, get existing value from cache
	if settings.AdminUser.Password != "" {
		logger.Info("updating admin user password")
		// get seed salt value from blueprint
		seed, err := getSaltValue(ctx)
		if err != nil {
			return err
		} else {
			// salt & hash the meat before you put in on the grill
			settings.AdminUser.Password = hashPassword(settings.AdminUser.Password, []byte(seed))
		}
	} else {
		existingSettings := &v1.DeviceSettings{}
		err := kvclient.Get(ctx, kvclient.DEFAULT_DEVICE_SETTINGS_KEY, existingSettings)
		if err != nil {
			return fmt.Errorf(ErrFailedToGetSettings)
		}
		settings.AdminUser.Password = existingSettings.AdminUser.Password
	}

	_, err = kvclient.Set(ctx, kvclient.DEFAULT_DEVICE_SETTINGS_KEY, settings)
	if err != nil {
		return errors.New(ErrFailedToCreateSettings)
	}

	return nil
}

func (c *controller) IsDeviceSetup(ctx context.Context) (bool, error) {
	// list is used to get all the `DeviceSettings` objects in the key-value store
	// it will not fail if the key does not exist like `Get` would
	settings := &v1.DeviceSettings{}
	list, err := kvclient.List(ctx, settings)
	if err != nil {
		return false, errors.New(ErrFailedToGetSettings)
	}

	if len(list) == 0 {
		return false, nil
	}
	return true, nil
}

func (c *controller) InitializeDevice(ctx context.Context, logger chassis.Logger, settings *v1.DeviceSettings) error {
	yes, err := c.IsDeviceSetup(ctx)
	if err != nil {
		return errors.New(ErrFailedToGetSettings)
	} else if yes {
		logger.Warn("device is already set up")
		return errors.New(ErrDeviceAlreadySetup)
	}

	// set the device settings on the host (via the daemon)
	done := make(chan bool)
	var listenerErr error
	go func() {
		listenerErr = async.RegisterListener(ctx, c.broadcaster, &async.ListenerOptions[*dv1.DeviceInitialized]{
			Callback: func(event *dv1.DeviceInitialized) (bool, error) {
				done<-true
				if event.Error != nil {
					return true, fmt.Errorf(event.Error.Error)
				}
				return true, nil
			},
		}).Listen(ctx)
	}()
	err = com.Send(&dv1.ServerMessage{
		Message: &dv1.ServerMessage_InitializeDeviceCommand{
			InitializeDeviceCommand: &dv1.InitializeDeviceCommand{
				User: &dv1.SetUserPasswordCommand{
					// TODO: Support multiple users? Right now the username "admin" is hardcoded into NixOS.
					Username: "admin",
					Password: settings.AdminUser.Password,
				},
				TimeZone: &dv1.SetTimeZoneCommand{
					TimeZone: settings.Timezone,
				},
			},
		},
	})
	if err != nil {
		return err
	}
	<-done
	if listenerErr != nil {
		return listenerErr
	}

	// get seed salt value from blueprint
	seed, err := getSaltValue(ctx)
	if err != nil {
		return err
	} else {
		// salt & hash the meat before you put in on the grill
		settings.AdminUser.Password = hashPassword(settings.AdminUser.Password, []byte(seed))
	}

	_, err = kvclient.Set(ctx, kvclient.DEFAULT_DEVICE_SETTINGS_KEY, settings)
	if err != nil {
		return errors.New(ErrFailedToCreateSettings)
	}

	return nil
}

func (c *controller) Login(ctx context.Context, username, password string) (string, error) {
	settings := &v1.DeviceSettings{}
	err := kvclient.Get(ctx, kvclient.DEFAULT_DEVICE_SETTINGS_KEY, settings)
	if err != nil {
		return "", errors.New(ErrFailedToGetSettings)
	}

	salt, err := getSaltValue(ctx)
	if err != nil {
		return "", errors.New(ErrFailedToGetSettings)
	}

	// Check if the password is correct. If not, return an error
	if hashPassword(password, []byte(salt)) != settings.AdminUser.Password {
		return "", errors.New("invalid username or password")
	}

	// TODO: forge token

	return "JWT_TOKEN", nil
}

// helper functions

func getSaltValue(ctx context.Context) (string, error) {
	seedVal := &kvv1.Value{}
	err := kvclient.Get(ctx, kvclient.SEED_KEY, seedVal)
	if err != nil {
		return "", errors.New(ErrFailedToBuildSeedGetRequest)
	}

	return seedVal.GetData(), nil
}

func hashPassword(password string, salt []byte) string {
	// a little salt & hash before saving the password
	var (
		pwBytes        = []byte(password)
		sha512Hasher   = sha512.New()
		hashedPassword = sha512Hasher.Sum(nil)
	)

	pwBytes = append(pwBytes, []byte(salt)...)
	sha512Hasher.Write(pwBytes)

	return hex.EncodeToString(hashedPassword)
}

func getLatestDaemonVersion() (*v1.DaemonVersion, error) {
	var (
		latest = &v1.DaemonVersion{}
	)

	// clone repo
	repo, err := git.Clone(memory.NewStorage(), nil, &git.CloneOptions{
		URL:           homeCloudCoreRepo,
		ReferenceName: homeCloudCoreTrunk,
		SingleBranch:  true,
		Depth:         1,
		Tags:          git.AllTags,
	})
	if err != nil {
		return nil, err
	}

	// pull out daemon versions from tags
	iter, err := repo.Tags()
	if err != nil {
		return nil, err
	}
	versions := []string{}
	err = iter.ForEach(func(tag *plumbing.Reference) error {
		name := tag.Name().String()
		if strings.HasPrefix(name, daemonTagPath) {
			versions = append(versions, strings.TrimPrefix(name, daemonTagPath))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	if len(versions) == 0 {
		return nil, fmt.Errorf("no versions found")
	}

	// sort versions by semver
	semver.Sort(versions)
	latest.Version = versions[len(versions)-1]

	// find hashes from tags
	iter, err = repo.Tags()
	if err != nil {
		return nil, err
	}
	err = iter.ForEach(func(tag *plumbing.Reference) error {
		name := tag.Name().String()
		prefix := fmt.Sprintf("refs/tags/daemon_%s", latest.Version)

		// ignore tag if it doesn't match the hash tag format for the latest version
		if !strings.HasPrefix(name, prefix) {
			return nil
		}

		// check which type of tag it is and save it
		parts := strings.Split(name, "_")
		t := parts[2]
		hash := parts[3]
		switch t {
		case "src":
			latest.SrcHash = hash
		case "vendor":
			latest.VendorHash = hash
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	// grab latest
	return latest, nil
}

func getLatestImageTags(ctx context.Context, images []*v1.ImageVersion) ([]*v1.ImageVersion, error) {
	for _, image := range images {
		latest, err := getLatestImageTag(ctx, image.Image)
		if err != nil {
			return nil, err
		}
		image.Latest = latest
	}
	return images, nil
}

func getLatestImageTag(ctx context.Context, image string) (string, error) {
	ref, err := docker.ParseReference(fmt.Sprintf("//%s", image))
	if err != nil {
		return "", err
	}

	tags, err := docker.GetRepositoryTags(ctx, nil, ref)
	if err != nil {
		return "", err
	}

	semverTags := []string{}
	for _, t := range tags {
		if !semver.IsValid(t) {
			continue
		}
		semverTags = append(semverTags, t)
	}

	var latestVersion string
	if len(semverTags) > 0 {
		semver.Sort(semverTags)
		latestVersion = semverTags[len(semverTags)-1]
	}

	return latestVersion, nil
}

func (c *controller) streamFile(ctx context.Context, logger chassis.Logger, buf io.Reader, fileId string) error {
	var (
		g         errgroup.Group
		log       = logger.WithField("file_id", fileId)
		chunkSize = 10 << 20
	)

	// create channels to communicate between goroutines
	chunks := make(chan fileChunk, 1)

	// start multiple goroutines to process chunks in parallel
	for i := 0; i < 4; i++ {
		log := log.WithField("worker", i)
		g.Go(func() error {
			log.Debug("waiting for work")
			for chunk := range chunks {
				log := log.WithField("chunk_index", chunk.index)
				log.Info("uploading chunk")
				// upload chunk
				done := make(chan bool)
				go async.RegisterListener(ctx, c.broadcaster, &async.ListenerOptions[*dv1.UploadFileChunkCompleted]{
					Callback: func(event *dv1.UploadFileChunkCompleted) (bool, error) {
						if event.FileId == fileId && event.Index == uint32(chunk.index) {
							done <- true
							return true, nil
						}
						return false, nil
					},
				}).Listen(ctx)
				err := com.Send(&dv1.ServerMessage{
					Message: &dv1.ServerMessage_UploadFileRequest{
						UploadFileRequest: &dv1.UploadFileRequest{
							Data: &dv1.UploadFileRequest_Chunk{
								Chunk: &dv1.FileChunk{
									FileId: fileId,
									Index:  uint32(chunk.index),
									Data:   chunk.data,
								},
							},
						},
					},
				})
				if err != nil {
					return err
				}

				// wait for done signal before uploading next chunk
				log.Debug("wait for done signal")
				<-done
				log.Debug("chunk upload complete")
			}
			log.Debug("done with work")
			return nil
		})
	}

	// send chunks to workers
	currentChunk := 0
	for {
		chunk := make([]byte, chunkSize)
		count, err := io.ReadFull(buf, chunk)
		if err != nil {
			// EOF means we're done
			if err == io.EOF {
				break
			}
			// error out on non-EOF error
			if err != io.ErrUnexpectedEOF {
				return err
			}
			// ErrUnexpectedEOF means we hit the end of the file before reaching chunkSize
			// so we need to trim excess bytes
			chunk = chunk[:count]
		}
		// send chunk to workers
		chunks <- fileChunk{
			index: currentChunk,
			data:  chunk,
		}
		// exit if we hit ErrUnexpectedEOF and trimmed the chunk
		if len(chunk) < chunkSize {
			break
		}
		currentChunk++
	}
	close(chunks)
	currentChunk++

	// wait for all goroutines to finish
	err := g.Wait()
	if err != nil {
		return err
	}
	log.Info("finished uploading chunks")

	// send done signal to daemon
	err = com.Send(&dv1.ServerMessage{
		Message: &dv1.ServerMessage_UploadFileRequest{
			UploadFileRequest: &dv1.UploadFileRequest{
				Data: &dv1.UploadFileRequest_Done{
					Done: &dv1.FileDone{
						FileId:     fileId,
						ChunkCount: uint32(currentChunk),
					},
				},
			},
		},
	})
	if err != nil {
		return err
	}

	return nil
}
