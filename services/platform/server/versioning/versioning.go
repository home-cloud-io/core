package versioning

// import (
// 	"context"
// 	"fmt"
// 	"strings"
// 	"sync"

// 	dv1 "github.com/home-cloud-io/core/api/platform/daemon/v1"
// 	v1 "github.com/home-cloud-io/core/api/platform/server/v1"
// 	"github.com/home-cloud-io/core/services/platform/server/daemon"
// 	k8sclient "github.com/home-cloud-io/core/services/platform/server/k8s-client"
// 	kvclient "github.com/home-cloud-io/core/services/platform/server/kv-client"
// 	"github.com/home-cloud-io/core/services/platform/server/system"

// 	"github.com/containers/image/v5/docker"
// 	"github.com/go-git/go-git/v5"
// 	"github.com/go-git/go-git/v5/plumbing"
// 	"github.com/go-git/go-git/v5/storage/memory"
// 	"github.com/robfig/cron/v3"
// 	"github.com/steady-bytes/draft/pkg/chassis"
// 	"golang.org/x/mod/semver"
// )

// type (
// 	Updater interface {
// 		AutoUpdateSystem()
// 		AutoUpdateSystemContainers()
// 		AutoUpdateApps()

// 		CheckForSystemUpdates(ctx context.Context, logger chassis.Logger) (*v1.CheckForSystemUpdatesResponse, error)
// 		CheckForContainerUpdates(ctx context.Context, logger chassis.Logger) ([]*v1.ImageVersion, error)
// 	}
// 	updater struct {
// 		logger           chassis.Logger
// 		k8sclient        k8sclient.Client
// 		messages         chan *dv1.DaemonMessage
// 		systemUpdateLock sync.Mutex
// 		controller       system.Controller
// 	}
// )

// const (
// 	ErrFailedToGetSettings = "failed to get settings"

// 	homeCloudCoreRepo  = "https://github.com/home-cloud-io/core"
// 	homeCloudCoreTrunk = "main"
// 	daemonTagPath      = "refs/tags/services/platform/daemon/"
// )

// func NewUpdater(logger chassis.Logger, messages chan *dv1.DaemonMessage, controller system.Controller) Updater {
// 	return &updater{
// 		logger:           logger,
// 		k8sclient:        k8sclient.NewClient(logger),
// 		messages:         messages,
// 		systemUpdateLock: sync.Mutex{},
// 		controller:       controller,
// 	}
// }

// // UpdateSystem (if enabled) will check for and install system updates every day at 1am
// func (u *updater) AutoUpdateSystem() {
// 	c := cron.New()
// 	c.AddFunc("0 1 * * *", u.updateSystem)
// 	go c.Start()
// }

// func (u *updater) updateSystem() {
// 	ctx := context.Background()
// 	u.logger.Info("")
// 	settings := &v1.DeviceSettings{}
// 	err := kvclient.Get(ctx, kvclient.DEFAULT_DEVICE_SETTINGS_KEY, settings)
// 	if err != nil {
// 		u.logger.WithError(err).Error("failed to get device settings")
// 		return
// 	}

// 	if !settings.AutoUpdateOs {
// 		u.logger.Info("auto update sytem not enabled")
// 		return
// 	}

// 	updates, err := u.CheckForSystemUpdates(ctx, u.logger)
// 	if err != nil {
// 		u.logger.WithError(err).Error("failed to check for system updates")
// 		return
// 	}

// 	// if the daemon needs updating, install it along with the os updates
// 	// otherwise just install the os update
// 	if updates.DaemonVersions.Current != updates.DaemonVersions.Latest {
// 		err = daemon.GetCommander().ChangeDaemonVersion(&dv1.ChangeDaemonVersionCommand{
// 			Version: updates.DaemonVersions.Latest,
// 			// TODO: need to get hashes from somewhere
// 		})
// 	} else {
// 		err = daemon.GetCommander().InstallOSUpdate()
// 	}
// 	if err != nil {
// 		u.logger.WithError(err).Error("failed to install system update")
// 		return
// 	}
// }

// // UpdateSystemContainers (if enabled) will check for and install system container updates every day at 2am
// func (u *updater) AutoUpdateSystemContainers() {
// 	c := cron.New()
// 	c.AddFunc("0 2 * * *", u.updateSystemContainers)
// 	go c.Start()
// }

// func (u *updater) updateSystemContainers() {
// 	ctx := context.Background()
// 	u.logger.Info("")
// 	settings := &v1.DeviceSettings{}
// 	err := kvclient.Get(ctx, kvclient.DEFAULT_DEVICE_SETTINGS_KEY, settings)
// 	if err != nil {
// 		u.logger.WithError(err).Error("failed to get device settings")
// 		return
// 	}

// 	// TODO: should this be a different setting?
// 	if !settings.AutoUpdateOs {
// 		u.logger.Info("auto update sytem not enabled")
// 		return
// 	}

// 	images, err := u.CheckForContainerUpdates(ctx, u.logger)
// 	if err != nil {
// 		u.logger.WithError(err).Error("failed to check for system container updates")
// 		return
// 	}

// 	commander := daemon.GetCommander()
// 	for _, image := range images {
// 		if semver.Compare(image.Latest, image.Current) == 1 {
// 			err := commander.SetSystemImage(&dv1.SetSystemImageCommand{
// 				CurrentImage:   fmt.Sprintf("%s:%s", image.Image, image.Current),
// 				RequestedImage: fmt.Sprintf("%s:%s", image.Image, image.Latest),
// 			})
// 			if err != nil {
// 				u.logger.WithFields(chassis.Fields{
// 					"image": image.Image,
// 					"current_version": image.Current,
// 					"latest_version": image.Latest,
// 				}).WithError(err).Error("failed to update system container image")
// 				// don't return, try to update other containers
// 			}
// 		}
// 	}
// }

// func (u *updater) CheckForSystemUpdates(ctx context.Context, logger chassis.Logger) (*v1.CheckForSystemUpdatesResponse, error) {
// 	if !u.systemUpdateLock.TryLock() {
// 		logger.Warn("call to check for system updates while another check is already in progress")
// 		return nil, fmt.Errorf("system update check already in progress")
// 	}
// 	defer u.systemUpdateLock.Unlock()

// 	var (
// 		response = &v1.CheckForSystemUpdatesResponse{}
// 	)

// 	// get the os update diff from the daemon
// 	commander := daemon.GetCommander()
// 	err := commander.RequestOSUpdateDiff()
// 	if err != nil {
// 		return nil, err
// 	}
// 	for {
// 		msg := <-u.messages
// 		switch msg.Message.(type) {
// 		case *dv1.DaemonMessage_OsUpdateDiff:
// 			m := msg.Message.(*dv1.DaemonMessage_OsUpdateDiff)
// 			response.OsDiff = m.OsUpdateDiff.Description
// 		default:
// 			logger.WithField("message", msg).Warn("unrequested message type received")
// 		}
// 		if response.OsDiff != "" {
// 			break
// 		}
// 	}

// 	// get the current daemon version from the daemon
// 	err = commander.RequestCurrentDaemonVersion()
// 	if err != nil {
// 		return nil, err
// 	}
// 	for {
// 		msg := <-u.messages
// 		switch msg.Message.(type) {
// 		case *dv1.DaemonMessage_CurrentDaemonVersion:
// 			m := msg.Message.(*dv1.DaemonMessage_CurrentDaemonVersion)
// 			response.DaemonVersions = &v1.DaemonVersions{
// 				Current: m.CurrentDaemonVersion.Version,
// 			}
// 		default:
// 			logger.WithField("message", msg).Warn("unrequested message type received")
// 		}
// 		if response.DaemonVersions != nil {
// 			break
// 		}
// 	}

// 	// get latest available daemon version
// 	latest, err := GetLatestDaemonVersion()
// 	if err != nil {
// 		return nil, err
// 	}
// 	response.DaemonVersions.Latest = latest

// 	return response, nil
// }

// func (u *updater) CheckForContainerUpdates(ctx context.Context, logger chassis.Logger) ([]*v1.ImageVersion, error) {
// 	var (
// 		images []*v1.ImageVersion
// 	)

// 	// populate current versions (from k8s)
// 	images, err := u.k8sclient.CurrentImages(ctx)
// 	if err != nil {
// 		logger.WithError(err).Error("failed to get current container versions")
// 		return nil, err
// 	}

// 	// populate latest versions (from registry)
// 	images, err = GetLatestImageTags(ctx, images)
// 	if err != nil {
// 		logger.WithError(err).Error("failed to get latest image versions")
// 		return nil, err
// 	}

// 	return images, err
// }

// func GetLatestDaemonVersion() (string, error) {
// 	// clone repo
// 	repo, err := git.Clone(memory.NewStorage(), nil, &git.CloneOptions{
// 		URL:           homeCloudCoreRepo,
// 		ReferenceName: homeCloudCoreTrunk,
// 		SingleBranch:  true,
// 		Depth:         1,
// 		Tags:          git.AllTags,
// 	})
// 	if err != nil {
// 		return "", err
// 	}

// 	// pull out daemon versions from tags
// 	iter, err := repo.Tags()
// 	if err != nil {
// 		return "", err
// 	}
// 	versions := []string{}
// 	err = iter.ForEach(func(tag *plumbing.Reference) error {
// 		name := tag.Name().String()
// 		if strings.HasPrefix(name, daemonTagPath) {
// 			versions = append(versions, strings.TrimPrefix(name, daemonTagPath))
// 		}
// 		return nil
// 	})
// 	if err != nil {
// 		return "", err
// 	}

// 	if len(versions) == 0 {
// 		return "", fmt.Errorf("no versions found")
// 	}

// 	// sort versions by semver
// 	semver.Sort(versions)

// 	// grab latest
// 	return versions[len(versions)-1], nil
// }

// func GetLatestImageTags(ctx context.Context, images []*v1.ImageVersion) ([]*v1.ImageVersion, error) {
// 	for _, image := range images {
// 		latest, err := getLatestImageTag(ctx, image.Image)
// 		if err != nil {
// 			return nil, err
// 		}
// 		image.Latest = latest
// 	}
// 	return images, nil
// }

// func getLatestImageTag(ctx context.Context, image string) (string, error) {
// 	ref, err := docker.ParseReference(fmt.Sprintf("//%s", image))
// 	if err != nil {
// 		return "", err
// 	}

// 	tags, err := docker.GetRepositoryTags(ctx, nil, ref)
// 	if err != nil {
// 		return "", err
// 	}

// 	semverTags := []string{}
// 	for _, t := range tags {
// 		if !semver.IsValid(t) {
// 			continue
// 		}
// 		semverTags = append(semverTags, t)
// 	}

// 	var latestVersion string
// 	if len(semverTags) > 0 {
// 		semver.Sort(semverTags)
// 		latestVersion = semverTags[len(semverTags)-1]
// 	}

// 	return latestVersion, nil
// }
