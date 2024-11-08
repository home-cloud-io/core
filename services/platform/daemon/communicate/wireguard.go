package communicate

import (
	"context"
	"fmt"

	v1 "github.com/home-cloud-io/core/api/platform/daemon/v1"
	"github.com/home-cloud-io/core/services/platform/daemon/host"
)

func (c *client) addWireguardInterface(ctx context.Context, def *v1.AddWireguardInterface) {
	err := host.AddWireguardInterface(ctx, c.logger, def)
	if err != nil {
		c.logger.WithError(err).Error("failed to rebuild and switch to NixOS configuration")
		err = c.stream.Send(&v1.DaemonMessage{
			Message: &v1.DaemonMessage_WireguardInterfaceAdded{
				WireguardInterfaceAdded: &v1.WireguardInterfaceAdded{
					Error: &v1.DaemonError{
						Error: fmt.Sprintf("failed to rebuild and switch to NixOS configuration: %s", err.Error()),
					},
				},
			},
		})
		if err != nil {
			c.logger.WithError(err).Error("failed to send error message to server")
		}
		return
	}

	err = c.stream.Send(&v1.DaemonMessage{
		Message: &v1.DaemonMessage_WireguardInterfaceAdded{},
	})
	if err != nil {
		c.logger.WithError(err).Error("failed to send complete message to server")
	}
}

func (c *client) removeWireguardInterface(ctx context.Context, def *v1.RemoveWireguardInterface) {
	err := host.RemoveWireguardInterface(ctx, c.logger, def)
	if err != nil {
		c.logger.WithError(err).Error("failed to rebuild and switch to NixOS configuration")
		err = c.stream.Send(&v1.DaemonMessage{
			Message: &v1.DaemonMessage_WireguardInterfaceRemoved{
				WireguardInterfaceRemoved: &v1.WireguardInterfaceRemoved{
					Error: &v1.DaemonError{
						Error: fmt.Sprintf("failed to rebuild and switch to NixOS configuration: %s", err.Error()),
					},
				},
			},
		})
		if err != nil {
			c.logger.WithError(err).Error("failed to send error message to server")
		}
		return
	}

	err = c.stream.Send(&v1.DaemonMessage{
		Message: &v1.DaemonMessage_WireguardInterfaceRemoved{},
	})
	if err != nil {
		c.logger.WithError(err).Error("failed to send complete message to server")
	}

}
