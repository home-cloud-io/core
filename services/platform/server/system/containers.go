package system

import (
	"context"
	"fmt"
	"sort"

	dv1 "github.com/home-cloud-io/core/api/platform/daemon/v1"
	v1 "github.com/home-cloud-io/core/api/platform/server/v1"
	opv1 "github.com/home-cloud-io/core/services/platform/operator/api/v1"

	"github.com/robfig/cron/v3"
	"github.com/steady-bytes/draft/pkg/chassis"
	"k8s.io/apimachinery/pkg/types"
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

// TODO: istio version management

// TODO: completely rethink updates
// 		 what if we had an rss feed on the website with updates published there so
// 		 we could control releases better?

// CONTAINERS

func (c *controller) CheckForContainerUpdates(ctx context.Context, logger chassis.Logger) ([]*v1.ImageVersion, error) {
	var (
		images []*v1.ImageVersion
	)

	install := &opv1.Install{}
	err := c.k8sclient.Get(ctx, types.NamespacedName{
		Name:      "install",
		Namespace: "home-cloud-system",
	}, install)
	if err != nil {
		logger.WithError(err).Error("failed to get installation opject")
		return nil, err
	}

	images = append(images, &v1.ImageVersion{
		Image:   install.Status.Server.Image,
		Current: install.Status.Server.Tag,
	})
	images = append(images, &v1.ImageVersion{
		Image:   install.Status.Daemon.Image,
		Current: install.Status.Daemon.Tag,
	})

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
