package system

import (
	"context"
	"fmt"
	"time"

	dv1 "github.com/home-cloud-io/core/api/platform/daemon/v1"
	v1 "github.com/home-cloud-io/core/api/platform/server/v1"
	kvclient "github.com/home-cloud-io/core/services/platform/server/kv-client"
	"github.com/robfig/cron/v3"
	"github.com/steady-bytes/draft/pkg/chassis"
	"golang.org/x/mod/semver"
)

type (
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
)

// CONTAINERS

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
