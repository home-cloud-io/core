package system

import (
	"context"
	"errors"
	"fmt"
	"slices"

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
	response, err := com.Request(ctx, &dv1.ServerMessage{
		Message: &dv1.ServerMessage_AddLocatorServerCommand{
			AddLocatorServerCommand: &dv1.AddLocatorServerCommand{
				LocatorAddress:     locatorAddress,
				WireguardInterface: wgInterfaceName,
			},
		},
	}, nil)
	if err != nil {
		return err
	}
	e := response.GetLocatorServerAdded()
	if e.Error != "" {
		return errors.New(e.Error)
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
	response, err := com.Request(ctx, &dv1.ServerMessage{
		Message: &dv1.ServerMessage_RemoveLocatorServerCommand{
			RemoveLocatorServerCommand: &dv1.RemoveLocatorServerCommand{
				LocatorAddress:     locatorAddress,
				WireguardInterface: wgInterfaceName,
			},
		},
	}, nil)
	if err != nil {
		return err
	}
	e := response.GetLocatorServerRemoved()
	if e.Error != "" {
		return errors.New(e.Error)
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
