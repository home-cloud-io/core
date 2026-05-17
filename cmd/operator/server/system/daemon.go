package system

import (
	"context"

	dv1 "github.com/home-cloud-io/core/api/platform/daemon/v1"

	"connectrpc.com/connect"
)

type (
	Daemon interface {
		// ShutdownHost will shutdown the host machine running Home Cloud.
		ShutdownHost(ctx context.Context) error
		// RestartHost will restart the host machine running Home Cloud
		RestartHost(ctx context.Context) error
	}
)

// DAEMON

func (c *controller) ShutdownHost(ctx context.Context) error {
	_, err := c.daemonClient.ShutdownHost(ctx, &connect.Request[dv1.ShutdownHostRequest]{})
	if err != nil {
		return err
	}
	return nil
}

func (c *controller) RestartHost(ctx context.Context) error {
	_, err := c.daemonClient.RebootHost(ctx, &connect.Request[dv1.RebootHostRequest]{})
	if err != nil {
		return err
	}
	return nil
}
