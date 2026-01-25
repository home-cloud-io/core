package system

import (
	"context"

	"connectrpc.com/connect"
	"github.com/steady-bytes/draft/pkg/chassis"
	"k8s.io/apimachinery/pkg/types"

	dv1 "github.com/home-cloud-io/core/api/platform/daemon/v1"
	v1 "github.com/home-cloud-io/core/api/platform/server/v1"
	opv1 "github.com/home-cloud-io/core/services/platform/operator/api/v1"
	k8sclient "github.com/home-cloud-io/core/services/platform/server/k8s-client"
)

type (
	Device interface {
		// GetServerSettings returns the current server settings after filtering out the
		// admin username and password.
		GetServerSettings(ctx context.Context) (*v1.DeviceSettings, error)
		// SetServerSettings updates the settings on the server with the given values
		SetServerSettings(ctx context.Context, logger chassis.Logger, settings *v1.DeviceSettings) error
		// GetComponentVersions returns all the versions of system components (server, daemon, etc.)
		GetComponentVersions(ctx context.Context, logger chassis.Logger) (*v1.GetComponentVersionsResponse, error)
	}
)

const (
	ErrDeviceAlreadySetup = "device already setup"

	ErrFailedToCreateSettings = "failed to create device settings"
	ErrFailedToGetSettings    = "failed to get device settings"
	ErrFailedToSetSettings    = "failed to save device settings"
)

// DEVICE

func (c *controller) GetServerSettings(ctx context.Context) (*v1.DeviceSettings, error) {
	settings, err := c.k8sclient.Settings(ctx)
	if err != nil {
		return nil, err
	}
	return &v1.DeviceSettings{
		AutoUpdateApps: settings.AutoUpdateApps,
		AutoUpdateOs:   settings.AutoUpdateSystem,
	}, nil
}

func (c *controller) SetServerSettings(ctx context.Context, logger chassis.Logger, settings *v1.DeviceSettings) error {
	// TODO
	return nil
}

func (c *controller) GetComponentVersions(ctx context.Context, logger chassis.Logger) (*v1.GetComponentVersionsResponse, error) {

	install := &opv1.Install{}
	err := c.k8sclient.Get(ctx, types.NamespacedName{
		Namespace: k8sclient.HomeCloudNamespace,
		Name:      "install",
	}, install)
	if err != nil {
		logger.WithError(err).Error("failed to get install")
		return nil, err
	}

	resp := &v1.GetComponentVersionsResponse{
		Platform: []*dv1.ComponentVersion{
			{
				Name:    "server",
				Domain:  "platform",
				Version: install.Status.Server.Tag,
			},
			{
				Name:    "mdns",
				Domain:  "platform",
				Version: install.Status.MDNS.Tag,
			},
			{
				Name:    "tunnel",
				Domain:  "platform",
				Version: install.Status.Tunnel.Tag,
			},
			{
				Name:    "daemon",
				Domain:  "platform",
				Version: install.Status.Daemon.Tag,
			},
		},
		System: []*dv1.ComponentVersion{
			{
				Name:    "home-cloud",
				Domain:  "system",
				Version: install.Status.Version,
			},
			{
				Name:    "istio",
				Domain:  "system",
				Version: install.Status.Istio.Version,
			},
			{
				Name:    "gateway-api",
				Domain:  "system",
				Version: install.Status.GatewayAPI.Version,
			},
		},
	}

	k8sVersion, err := c.k8sclient.GetServerVersion(ctx)
	if err != nil {
		resp.System = append(resp.System, &dv1.ComponentVersion{
			Name:    "kubernetes",
			Domain:  "system",
			Version: err.Error(),
		})
	} else {
		resp.System = append(resp.System, &dv1.ComponentVersion{
			Name:    "kubernetes",
			Domain:  "system",
			Version: k8sVersion,
		})
	}

	osVersion, err := c.daemonClient.Version(ctx, connect.NewRequest(&dv1.VersionRequest{}))
	if err != nil {
		resp.System = append(resp.System, &dv1.ComponentVersion{
			Name:    "unknown",
			Domain:  "system",
			Version: err.Error(),
		})
	} else {
		resp.System = append(resp.System, &dv1.ComponentVersion{
			Name:    osVersion.Msg.Name,
			Domain:  "system",
			Version: osVersion.Msg.Version,
		})
	}

	return resp, nil
}
