package system

import (
	"context"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"

	dv1 "github.com/home-cloud-io/core/api/platform/daemon/v1"
	v1 "github.com/home-cloud-io/core/api/platform/server/v1"
	"github.com/home-cloud-io/core/services/platform/server/daemon"
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
)

type (
	Controller interface {
		GetServerSettings(ctx context.Context) (*v1.DeviceSettings, error)
		IsDeviceSetup(ctx context.Context) (bool, error)
		InitializeDevice(ctx context.Context, settings *v1.DeviceSettings) (string, error)
		Login(ctx context.Context, username, password string) (string, error)
		CheckForOSUpdates(ctx context.Context, logger chassis.Logger) (*v1.CheckForSystemUpdatesResponse, error)
		CheckForContainerUpdates(ctx context.Context, logger chassis.Logger) ([]*v1.ImageVersion, error)
		AutoUpdateOS(logger chassis.Logger)
		AutoUpdateContainers(logger chassis.Logger)
		UpdateOS(ctx context.Context, logger chassis.Logger) error
		UpdateContainers(ctx context.Context, logger chassis.Logger) error
	}

	controller struct {
		k8sclient        k8sclient.Client
		messages         chan *dv1.DaemonMessage
		systemUpdateLock sync.Mutex
	}
)

func NewController(logger chassis.Logger, messages chan *dv1.DaemonMessage) Controller {
	return &controller{
		k8sclient:        k8sclient.NewClient(logger),
		messages:         messages,
		systemUpdateLock: sync.Mutex{},
	}
}

const (
	ErrDeviceAlreadySetup          = "device already setup"
	ErrFailedToCreateSettings      = "failed to create settings"
	ErrFailedToSaveSettings        = "failed to save settings"
	ErrFailedToGetSettings         = "failed to get settings"
	ErrFailedToBuildSeedGetRequest = "failed to build get request for seed"
	ErrFailedToGetSeedValue        = "failed to get seed value"
	ErrFailedToUnmarshalSeedValue  = "failed to unmarshal seed value"

	homeCloudCoreRepo       = "https://github.com/home-cloud-io/core"
	homeCloudCoreTrunk      = "main"
	daemonTagPath           = "refs/tags/services/platform/daemon/"
	osAutoUpdateCron        = "0 1 * * *"
	containerAutoUpdateCron = "0 2 * * *"
)

func (c *controller) GetServerSettings(ctx context.Context) (*v1.DeviceSettings, error) {
	settings := &v1.DeviceSettings{}
	err := kvclient.Get(ctx, kvclient.DEFAULT_DEVICE_SETTINGS_KEY, settings)
	if err != nil {
		return nil, errors.New(ErrFailedToGetSettings)
	}

	settings.AdminUser.Password = "" // don't return the password
	settings.AdminUser.Username = "" // don't return the username

	return settings, nil
}

// IsDeviceSetup checks if the device is already setup by checking if the DEFAULT_DEVICE_SETTINGS_KEY key exists in the key-value store
// with the default settings model
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

// InitializeDevice initializes the device with the given settings. It first checks if the device is already setup
// Uses the user-provided password to set the password for the "admin" user on the device
// and save the remaining settings in the key-value store
func (c *controller) InitializeDevice(ctx context.Context, settings *v1.DeviceSettings) (string, error) {
	yes, err := c.IsDeviceSetup(ctx)
	if err != nil {
		return "", errors.New(ErrFailedToGetSettings)
	} else if yes {
		return "", errors.New(ErrDeviceAlreadySetup)
	}

	// TODO: set the password for the "admin" user on the device (call to daemon)

	// TODO: Get seed salt value from `blue_print`
	seed, err := getSaltValue(ctx)
	if err != nil {
		return "", err
	} else {
		// salt & hash the meat before you put in on the grill
		settings.AdminUser.Password = hashPassword(settings.AdminUser.Password, []byte(seed))
	}

	key, err := kvclient.Set(ctx, kvclient.DEFAULT_DEVICE_SETTINGS_KEY, settings)
	if err != nil {
		return "", errors.New(ErrFailedToCreateSettings)
	}

	return key, nil
}

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
	commander := daemon.GetCommander()
	err := commander.RequestOSUpdateDiff()
	if err != nil {
		return nil, err
	}
	for {
		msg := <-c.messages
		switch msg.Message.(type) {
		case *dv1.DaemonMessage_OsUpdateDiff:
			m := msg.Message.(*dv1.DaemonMessage_OsUpdateDiff)
			response.OsDiff = m.OsUpdateDiff.Description
		default:
			logger.WithField("message", msg).Warn("unrequested message type received")
		}
		if response.OsDiff != "" {
			break
		}
	}

	// get the current daemon version from the daemon
	err = commander.RequestCurrentDaemonVersion()
	if err != nil {
		return nil, err
	}
	for {
		msg := <-c.messages
		switch msg.Message.(type) {
		case *dv1.DaemonMessage_CurrentDaemonVersion:
			m := msg.Message.(*dv1.DaemonMessage_CurrentDaemonVersion)
			response.DaemonVersions = &v1.DaemonVersions{
				Current: m.CurrentDaemonVersion.Version,
			}
		default:
			logger.WithField("message", msg).Warn("unrequested message type received")
		}
		if response.DaemonVersions != nil {
			break
		}
	}

	// get latest available daemon version
	latest, err := GetLatestDaemonVersion()
	if err != nil {
		return nil, err
	}
	response.DaemonVersions.Latest = latest

	return response, nil
}

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
	images, err = GetLatestImageTags(ctx, images)
	if err != nil {
		logger.WithError(err).Error("failed to get latest image versions")
		return nil, err
	}

	return images, err
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
	cr.AddFunc(osAutoUpdateCron, f)
	go cr.Start()
}

func (u *controller) UpdateOS(ctx context.Context, logger chassis.Logger) error {
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
		err = daemon.GetCommander().ChangeDaemonVersion(&dv1.ChangeDaemonVersionCommand{
			Version: updates.DaemonVersions.Latest,
			// TODO: need to get hashes from somewhere
		})
	} else {
		err = daemon.GetCommander().InstallOSUpdate()
	}
	if err != nil {
		logger.WithError(err).Error("failed to install system update")
		return err
	}

	return nil
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
	cr.AddFunc(osAutoUpdateCron, f)
	go cr.Start()
}

func (c *controller) UpdateContainers(ctx context.Context, logger chassis.Logger) error {
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

	commander := daemon.GetCommander()
	for _, image := range images {
		if semver.Compare(image.Latest, image.Current) == 1 {
			err := commander.SetSystemImage(&dv1.SetSystemImageCommand{
				CurrentImage:   fmt.Sprintf("%s:%s", image.Image, image.Current),
				RequestedImage: fmt.Sprintf("%s:%s", image.Image, image.Latest),
			})
			if err != nil {
				logger.WithFields(chassis.Fields{
					"image":           image.Image,
					"current_version": image.Current,
					"latest_version":  image.Latest,
				}).WithError(err).Error("failed to update system container image")
				// don't return, try to update other containers
			}
		}
	}

	return nil
}

func GetLatestDaemonVersion() (string, error) {
	// clone repo
	repo, err := git.Clone(memory.NewStorage(), nil, &git.CloneOptions{
		URL:           homeCloudCoreRepo,
		ReferenceName: homeCloudCoreTrunk,
		SingleBranch:  true,
		Depth:         1,
		Tags:          git.AllTags,
	})
	if err != nil {
		return "", err
	}

	// pull out daemon versions from tags
	iter, err := repo.Tags()
	if err != nil {
		return "", err
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
		return "", err
	}

	if len(versions) == 0 {
		return "", fmt.Errorf("no versions found")
	}

	// sort versions by semver
	semver.Sort(versions)

	// grab latest
	return versions[len(versions)-1], nil
}

func GetLatestImageTags(ctx context.Context, images []*v1.ImageVersion) ([]*v1.ImageVersion, error) {
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
