package system

import (
	"context"

	dv1 "github.com/home-cloud-io/core/api/platform/daemon/v1"

	"github.com/steady-bytes/draft/pkg/chassis"
)

type (
	Containers interface {
		// GetContainerLogs will return all logs for all containers in the cluster.
		GetContainerLogs(ctx context.Context, logger chassis.Logger, sinceSeconds int64) ([]*dv1.Log, error)
	}
)

func (c *controller) GetContainerLogs(ctx context.Context, logger chassis.Logger, sinceSeconds int64) ([]*dv1.Log, error) {
	return c.k8sclient.GetLogs(ctx, logger, sinceSeconds)
}
