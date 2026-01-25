package system

import (
	"context"
	"fmt"

	dv1 "github.com/home-cloud-io/core/api/platform/daemon/v1"
	v1 "github.com/home-cloud-io/core/api/platform/server/v1"

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

// TODO: change updates to use the operator by setting the Version field on the Install

// CONTAINERS

func (c *controller) CheckForContainerUpdates(ctx context.Context, logger chassis.Logger) ([]*v1.ImageVersion, error) {

	// TODO: remove this?

	return nil, nil
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
