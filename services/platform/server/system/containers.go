package system

import (
	"context"
	"fmt"
	"sort"

	dv1 "github.com/home-cloud-io/core/api/platform/daemon/v1"
	v1 "github.com/home-cloud-io/core/api/platform/server/v1"
	kvclient "github.com/home-cloud-io/core/services/platform/server/kv-client"
	"github.com/robfig/cron/v3"
	"github.com/steady-bytes/draft/pkg/chassis"
)

type (
	Containers interface {
		// CheckForContainerUpdates will compare current system container images against the latest ones
		// available and return the result.
		CheckForContainerUpdates(ctx context.Context, logger chassis.Logger) ([]*v1.ImageVersion, error)
		// AutoUpdateContainers will check for and install any container updates on a schedule. It is
		// designed to be called at bootup.
		AutoUpdateContainers(logger chassis.Logger)
		// UpdateContainers will check for and install any container updates one time.
		UpdateContainers(ctx context.Context, logger chassis.Logger) error
		// GetContainerLogs will return all logs for all containers in the cluster.
		GetContainerLogs(ctx context.Context, logger chassis.Logger, sinceSeconds int64) ([]*dv1.Log, error)
	}
)

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

	// add shorthand name to image structs
	for _, image := range images {
		image.Name = componentFromImage(image.Image)
	}

	// sort images
	sort.Slice(images, func(i, j int) bool {
		return images[i].Name < images[j].Name
	})

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

	// TODO: Write to Install CRD and let Operator perform upgrade
	fmt.Println(images)

	return nil
}

func (c *controller) GetContainerLogs(ctx context.Context, logger chassis.Logger, sinceSeconds int64) ([]*dv1.Log, error) {
	return c.k8sclient.GetLogs(ctx, logger, sinceSeconds)
}
