package host

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"slices"

	v1 "github.com/home-cloud-io/core/api/platform/daemon/v1"
	sv1 "github.com/home-cloud-io/core/api/platform/server/v1"

	"github.com/steady-bytes/draft/pkg/chassis"
)

type (
	RemoteAccessController interface {
		// Enable()
		// Disable()
		Load()

		AddInterface(ctx context.Context, wireguardInterface *v1.WireguardInterface) (publicKey string, err error)
		RemoveInterface(ctx context.Context, wgInterfaceName string) error
		AddPeer(ctx context.Context, wgInterfaceName string, peer *v1.WireguardPeer) (addresses []string, dnsServers []string, err error)

		AddLocator(ctx context.Context, wgInterfaceName string, locatorAddress string) error
		RemoveLocator(ctx context.Context, wgInterfaceName string, locatorAddress string) error
		RemoveAllLocators(ctx context.Context, wgInterfaceName string) error

		BindSTUNServer(ctx context.Context, wgInterfaceName string, stunServer string) error
	}
	remoteAccessController struct {
		logger              chassis.Logger
		locatorController   LocatorController
		stunController      STUNController
		wireguardController WireguardController
	}
)

const (
	SecureTunnelingNotEnabledError = "remote access not enabled"
)

func NewRemoteAccessController(logger chassis.Logger) RemoteAccessController {
	stunController := NewSTUNController(logger)
	return remoteAccessController{
		logger:              logger,
		stunController:      stunController,
		locatorController:   NewLocatorController(logger, stunController),
		wireguardController: NewWireguardController(),
	}
}

func (c remoteAccessController) Load() {
	ctx := context.Background()
	settings, err := secureTunnelingSettings()
	if err != nil {
		c.logger.WithError(err).Error("failed to read remote access settings")
		return
	}
	// get networking configuration from NixOS config file
	config := NetworkingConfig{}
	f, err := os.ReadFile(NetworkingConfigFile())
	if err != nil {
		c.logger.WithError(err).Error("failed to read networking config")
		return
	}
	err = json.Unmarshal(f, &config)
	if err != nil {
		c.logger.WithError(err).Error("failed to unmarshal networking config")
	}

	// iterate through all Wireguard interfaces and do two things for each interface:
	// 1. create a STUN binding (with port multiplexing) to configured server
	// 2. connect to all configured Locator servers
	for _, wgInterface := range settings.WireguardInterfaces {
		log := c.logger.WithFields(chassis.Fields{
			"interface_id":   wgInterface.Id,
			"interface_name": wgInterface.Name,
		})
		log.Info("loading wireguard interface STUN and Locator servers")

		infConfig, ok := config.Wireguard.Interfaces[wgInterface.Name]
		if !ok {
			log.Error("no configured wireguard interface with given name")
			continue
		}

		// create STUN binding for interface
		err := c.stunController.Bind(int(infConfig.ListenPort), wgInterface.StunServer)
		if err != nil {
			log.WithError(err).Error("failed to get public address using STUN client")
			continue
		}

		// open connections to all locator servers configured for interface
		for _, locatorAddress := range wgInterface.LocatorServers {
			c.locatorController.Connect(ctx, wgInterface, locatorAddress)
		}
	}

}

func (c remoteAccessController) AddInterface(ctx context.Context, wgInterface *v1.WireguardInterface) (publicKey string, err error) {
	settings, err := secureTunnelingSettings()
	if err != nil {
		if err.Error() != SecureTunnelingNotEnabledError {
			return "", err
		}
		settings = &sv1.SecureTunnelingSettings{}
	}

	// TODO: add a command to enable/disable secure tunneling without modifying any other config (will need to figure out wireguard nixos config)
	settings.Enabled = true

	// make sure the interface doesn't already exist in settings
	for _, existingInterface := range settings.WireguardInterfaces {
		if existingInterface.Name == wgInterface.Name {
			return "", errors.New("wireguard interface with same name already exists in settings")
		}
	}

	publicKey, err = c.wireguardController.AddInterface(ctx, c.logger, wgInterface)
	if err != nil {
		return "", err
	}

	// update settings config
	settings.WireguardInterfaces = append(settings.WireguardInterfaces, &sv1.WireguardInterface{
		Id:        wgInterface.Id,
		Name:      wgInterface.Name,
		Port:      int32(wgInterface.ListenPort),
		PublicKey: publicKey,
	})
	chassis.GetConfig().SetAndWrite(SecureTunnelingSettingsKey, settings)

	return publicKey, nil
}

func (c remoteAccessController) RemoveInterface(ctx context.Context, wgInterfaceName string) error {
	settings, err := secureTunnelingSettings()
	if err != nil {
		return err
	}

	var wgInterface *sv1.WireguardInterface
	for i, inf := range settings.WireguardInterfaces {
		if inf.Name == wgInterfaceName {
			wgInterface = inf
			// remove interface from settings
			settings.WireguardInterfaces = slices.Delete(settings.WireguardInterfaces, i, i+1)
			break
		}
	}
	if wgInterface == nil {
		return errors.New("given wireguard interface not found in settings")
	}

	// close all locator connections
	err = c.RemoveAllLocators(ctx, wgInterfaceName)
	if err != nil {
		c.logger.WithError(err).Error("failed to remove all locators")
		return err
	}

	// close STUN binding
	err = c.stunController.Cancel(int(wgInterface.Port))
	if err != nil {
		c.logger.WithError(err).Error("failed to cancel STUN binding")
		return err
	}

	// remove interface
	err = c.wireguardController.RemoveInterface(ctx, c.logger, wgInterfaceName)
	if err != nil {
		c.logger.WithError(err).Error("failed to remove wireguard interface")
		return err
	}

	// update settings config
	chassis.GetConfig().SetAndWrite(SecureTunnelingSettingsKey, settings)

	return nil
}

func (c remoteAccessController) AddPeer(ctx context.Context, wgInterfaceName string, peer *v1.WireguardPeer) (addresses []string, dnsServers []string, err error) {
	settings, err := secureTunnelingSettings()
	if err != nil {
		return nil, nil, err
	}

	var wgInterface *sv1.WireguardInterface
	for _, inf := range settings.WireguardInterfaces {
		if inf.Name == wgInterfaceName {
			wgInterface = inf
			break
		}
	}
	if wgInterface == nil {
		return nil, nil, errors.New("given wireguard interface not found in settings")
	}

	addresses, err = c.wireguardController.AddPeer(ctx, c.logger, wgInterfaceName, peer)
	if err != nil {
		c.logger.WithError(err).Error("failed to add wireguard peer")
		return nil, nil, err
	}

	dnsAddress, err := getOutboundIP()
	if err != nil {
		c.logger.WithError(err).Error("failed to get outbound ip address for wireguard peer dns")
		return nil, nil, err
	}

	return addresses, []string{dnsAddress}, nil
}

func (c remoteAccessController) AddLocator(ctx context.Context, wgInterfaceName string, locatorAddress string) error {
	settings, err := secureTunnelingSettings()
	if err != nil {
		return err
	}

	var wgInterface *sv1.WireguardInterface
	for _, inf := range settings.WireguardInterfaces {
		if inf.Name == wgInterfaceName {
			wgInterface = inf
			break
		}
	}
	if wgInterface == nil {
		return errors.New("given wireguard interface not found in settings")
	}

	// make sure the locator server isn't already configured for the given interface
	for _, l := range wgInterface.LocatorServers {
		if l == locatorAddress {
			return errors.New("locator server is already configured for the given wireguard interface")
		}
	}

	// save new locator address to settings
	wgInterface.LocatorServers = append(wgInterface.LocatorServers, locatorAddress)

	// update settings config
	chassis.GetConfig().SetAndWrite(SecureTunnelingSettingsKey, settings)

	// open connection to new locator
	c.locatorController.Connect(ctx, wgInterface, locatorAddress)

	return nil
}

func (c remoteAccessController) RemoveLocator(ctx context.Context, wgInterfaceName string, locatorAddress string) error {
	settings, err := secureTunnelingSettings()
	if err != nil {
		return err
	}

	var wgInterface *sv1.WireguardInterface
	for _, inf := range settings.WireguardInterfaces {
		if inf.Name == wgInterfaceName {
			wgInterface = inf
			break
		}
	}
	if wgInterface == nil {
		return errors.New("given wireguard interface not found in settings")
	}

	// find and remove the locator from the wireguard interface
	for i, l := range wgInterface.LocatorServers {
		if l == locatorAddress {
			wgInterface.LocatorServers = slices.Delete(wgInterface.LocatorServers, i, i)
			break
		}
	}

	// update settings config
	chassis.GetConfig().SetAndWrite(SecureTunnelingSettingsKey, settings)

	// close connection to locator
	c.locatorController.Close(wgInterface, locatorAddress)

	return nil
}

func (c remoteAccessController) RemoveAllLocators(ctx context.Context, wgInterfaceName string) error {
	settings, err := secureTunnelingSettings()
	if err != nil {
		return err
	}

	var wgInterface *sv1.WireguardInterface
	for _, inf := range settings.WireguardInterfaces {
		if inf.Name == wgInterfaceName {
			wgInterface = inf
			break
		}
	}
	if wgInterface == nil {
		return errors.New("given wireguard interface not found in settings")
	}

	// close connection to all locators
	for _, l := range wgInterface.LocatorServers {
		c.locatorController.Close(wgInterface, l)
	}

	// update settings config
	wgInterface.LocatorServers = []string{}
	chassis.GetConfig().SetAndWrite(SecureTunnelingSettingsKey, settings)

	return nil
}

func (c remoteAccessController) BindSTUNServer(ctx context.Context, wgInterfaceName string, stunServerAddress string) error {
	settings, err := secureTunnelingSettings()
	if err != nil {
		return err
	}

	var wgInterface *sv1.WireguardInterface
	for _, inf := range settings.WireguardInterfaces {
		if inf.Name == wgInterfaceName {
			wgInterface = inf
			break
		}
	}
	if wgInterface == nil {
		return errors.New("given wireguard interface not found in settings")
	}

	// first, attempt to cancel any current binding on the given interface port
	err = c.stunController.Cancel(int(wgInterface.Port))
	if err != nil {
		c.logger.WithError(err).Error("failed to cancel STUN binding")
		return err
	}

	// now, bind the new server on the given interface port
	err = c.stunController.Bind(int(wgInterface.Port), stunServerAddress)
	if err != nil {
		c.logger.WithError(err).Error("failed to bind to STUN server")
		return err
	}

	// update settings config
	wgInterface.StunServer = stunServerAddress
	chassis.GetConfig().SetAndWrite(SecureTunnelingSettingsKey, settings)

	return nil
}

func secureTunnelingSettings() (*sv1.SecureTunnelingSettings, error) {
	var (
		settings = &sv1.SecureTunnelingSettings{}
		err      error
	)

	// check if locator already is configured for the given interface
	err = chassis.GetConfig().UnmarshalKey(SecureTunnelingSettingsKey, settings)
	if err != nil {
		return nil, err
	}

	// make sure settings are enabled
	if !settings.Enabled {
		return nil, errors.New(SecureTunnelingNotEnabledError)
	}

	return settings, err
}
