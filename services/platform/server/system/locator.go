package system

import (
	"context"
	"fmt"
	"slices"

	"connectrpc.com/connect"
	dv1 "github.com/home-cloud-io/core/api/platform/daemon/v1"
	v1 "github.com/home-cloud-io/core/api/platform/server/v1"
	kvclient "github.com/home-cloud-io/core/services/platform/server/kv-client"
)

type (
	Locators interface {
		AddLocator(ctx context.Context, wgInterfaceName string, locatorAddress string) (err error)
		RemoveLocator(ctx context.Context, wgInterfaceName string, locatorAddress string) error
	}
)

func (c *controller) AddLocator(ctx context.Context, wgInterfaceName string, locatorAddress string) (err error) {
	_, err = c.daemonClient.AddLocatorServer(ctx, &connect.Request[dv1.AddLocatorServerRequest]{
		Msg: &dv1.AddLocatorServerRequest{
			LocatorAddress:     locatorAddress,
			WireguardInterface: wgInterfaceName,
		},
	})
	if err != nil {
		return err
	}

	settings := &v1.DeviceSettings{}
	err = kvclient.Get(ctx, kvclient.DEFAULT_DEVICE_SETTINGS_KEY, settings)
	if err != nil {
		return err
	}
	settings.SecureTunnelingSettings.WireguardInterfaces[0].LocatorServers = append(settings.SecureTunnelingSettings.WireguardInterfaces[0].LocatorServers, locatorAddress)
	_, err = kvclient.Set(ctx, kvclient.DEFAULT_DEVICE_SETTINGS_KEY, settings)
	if err != nil {
		return fmt.Errorf("failed to save settings")
	}

	return nil
}

func (c *controller) RemoveLocator(ctx context.Context, wgInterfaceName string, locatorAddress string) (err error) {
	_, err = c.daemonClient.RemoveLocatorServer(ctx, &connect.Request[dv1.RemoveLocatorServerRequest]{
		Msg: &dv1.RemoveLocatorServerRequest{
			LocatorAddress:     locatorAddress,
			WireguardInterface: wgInterfaceName,
		},
	})
	if err != nil {
		return err
	}

	settings := &v1.DeviceSettings{}
	err = kvclient.Get(ctx, kvclient.DEFAULT_DEVICE_SETTINGS_KEY, settings)
	if err != nil {
		return err
	}
	for i, l := range settings.SecureTunnelingSettings.WireguardInterfaces[0].LocatorServers {
		if l == locatorAddress {
			settings.SecureTunnelingSettings.WireguardInterfaces[0].LocatorServers = slices.Delete(settings.SecureTunnelingSettings.WireguardInterfaces[0].LocatorServers, i, i+1)
		}
	}
	_, err = kvclient.Set(ctx, kvclient.DEFAULT_DEVICE_SETTINGS_KEY, settings)
	if err != nil {
		return fmt.Errorf("failed to save settings")
	}

	return nil
}
