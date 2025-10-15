package host

import (
	"context"
	"errors"
	"net"
	"slices"

	v1 "github.com/home-cloud-io/core/api/platform/daemon/v1"
	sv1 "github.com/home-cloud-io/core/api/platform/server/v1"
	kvclient "github.com/home-cloud-io/core/services/platform/daemon/kv-client"

	"github.com/steady-bytes/draft/pkg/chassis"
)

type (
	SecureTunnelingController interface {
		// Load is intended to be called at start up and will read secure tunneling configured resources and start
		// them (e.g. Wireguard interfaces and their respective STUN servers and Locator connections).
		Load()

		// AddInterface will add a Wireguard interface to the host.
		AddInterface(ctx context.Context, wireguardInterface *v1.WireguardInterface) (publicKey string, err error)
		// RemoveInterface will remove a Wireguard interface from the host and also remove any dependent
		// resources (STUN bindings and Locator connections).
		RemoveInterface(ctx context.Context, wgInterfaceName string) error
		// AddPeer will add a Wireguard peer to the given interface.
		AddPeer(ctx context.Context, wgInterfaceName string, peer *v1.WireguardPeer) (addresses []string, dnsServers []string, err error)
		// TODO: add RemovePeer()

		// AddLocator will add a Locator conneciton to the given interface.
		AddLocator(ctx context.Context, wgInterfaceName string, locatorAddress string) error
		// RemoveLocator will remove a Locator connection from the given interface.
		RemoveLocator(ctx context.Context, wgInterfaceName string, locatorAddress string) error

		// BindSTUNServer will add (or replace) a STUN bunding to the given Wireguard interface.
		BindSTUNServer(ctx context.Context, wgInterfaceName string, stunServer string) error
	}
	secureTunnelingController struct {
		logger              chassis.Logger
		locatorController   LocatorController
		stunController      STUNController
		wireguardController WireguardController
	}
)

const (
	SecureTunnelingNotEnabledError = "secure tunneling not enabled"
)

func NewSecureTunnelingController(logger chassis.Logger) SecureTunnelingController {
	stunController := NewSTUNController(logger)
	return secureTunnelingController{
		logger:              logger,
		stunController:      stunController,
		locatorController:   NewLocatorController(logger, stunController),
		wireguardController: NewWireguardController(),
	}
}

func (c secureTunnelingController) Load() {
	ctx := context.Background()
	settings, err := kvclient.Settings(ctx)
	if err != nil {
		return
	}

	// iterate through all Wireguard interfaces and do two things for each interface:
	// 1. create a STUN binding (with port multiplexing) to configured server
	// 2. connect to all configured Locator servers
	for _, wgInterface := range settings.SecureTunnelingSettings.WireguardInterfaces {
		log := c.logger.WithFields(chassis.Fields{
			"interface_id":   wgInterface.Id,
			"interface_name": wgInterface.Name,
		})
		log.Info("loading wireguard interface STUN and Locator servers")

		// create STUN binding for interface
		err := c.stunController.Bind(int(wgInterface.Port), wgInterface.StunServer)
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

func (c secureTunnelingController) AddInterface(ctx context.Context, wgInterface *v1.WireguardInterface) (publicKey string, err error) {
	settings, err := kvclient.Settings(ctx)
	if err != nil {
		return
	}

	// initializing secur tunneling so create empty settings
	if settings.SecureTunnelingSettings == nil {
		settings.SecureTunnelingSettings = &sv1.SecureTunnelingSettings{
			Enabled:             true,
			WireguardInterfaces: make([]*sv1.WireguardInterface, 0),
		}
	}

	// make sure the interface doesn't already exist in settings
	for _, existingInterface := range settings.SecureTunnelingSettings.WireguardInterfaces {
		if existingInterface.Name == wgInterface.Name {
			return "", errors.New("wireguard interface with same name already exists in settings")
		}
	}

	publicKey, err = c.wireguardController.AddInterface(ctx, c.logger, wgInterface)
	if err != nil {
		return "", err
	}

	// update settings config
	settings.SecureTunnelingSettings.WireguardInterfaces = append(settings.SecureTunnelingSettings.WireguardInterfaces, &sv1.WireguardInterface{
		Id:        wgInterface.Id,
		Name:      wgInterface.Name,
		Port:      int32(wgInterface.ListenPort),
		PublicKey: publicKey,
	})
	_, err = kvclient.Set(ctx, kvclient.DEFAULT_DEVICE_SETTINGS_KEY, settings)
	if err != nil {
		c.logger.WithError(err).Error("failed to update settings when adding wireguard interface")
		return "", err
	}

	return publicKey, nil
}

func (c secureTunnelingController) RemoveInterface(ctx context.Context, wgInterfaceName string) error {
	settings, err := kvclient.Settings(ctx)
	if err != nil {
		return err
	}

	var wgInterface *sv1.WireguardInterface
	for i, inf := range settings.SecureTunnelingSettings.WireguardInterfaces {
		if inf.Name == wgInterfaceName {
			wgInterface = inf
			// remove interface from settings
			settings.SecureTunnelingSettings.WireguardInterfaces = slices.Delete(settings.SecureTunnelingSettings.WireguardInterfaces, i, i+1)
			break
		}
	}
	if wgInterface == nil {
		return errors.New("given wireguard interface not found in settings")
	}

	// close all locator connections
	for _, l := range wgInterface.LocatorServers {
		c.locatorController.Close(wgInterface, l)
	}

	// cancel STUN binding
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
	_, err = kvclient.Set(ctx, kvclient.DEFAULT_DEVICE_SETTINGS_KEY, settings)
	if err != nil {
		c.logger.WithError(err).Error("failed to update settings when removing wireguard interface")
		return err
	}

	c.logger.Info("finished removing wireguard interface")

	return nil
}

func (c secureTunnelingController) AddPeer(ctx context.Context, wgInterfaceName string, peer *v1.WireguardPeer) (addresses []string, dnsServers []string, err error) {
	settings, err := kvclient.Settings(ctx)
	if err != nil {
		return nil, nil, err
	}

	var wgInterface *sv1.WireguardInterface
	for _, inf := range settings.SecureTunnelingSettings.WireguardInterfaces {
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

func (c secureTunnelingController) AddLocator(ctx context.Context, wgInterfaceName string, locatorAddress string) error {
	settings, err := kvclient.Settings(ctx)
	if err != nil {
		return err
	}

	var wgInterface *sv1.WireguardInterface
	for _, inf := range settings.SecureTunnelingSettings.WireguardInterfaces {
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
	_, err = kvclient.Set(ctx, kvclient.DEFAULT_DEVICE_SETTINGS_KEY, settings)
	if err != nil {
		c.logger.WithError(err).Error("failed to update settings when adding locator")
		return err
	}

	// open connection to new locator
	c.locatorController.Connect(ctx, wgInterface, locatorAddress)

	return nil
}

func (c secureTunnelingController) RemoveLocator(ctx context.Context, wgInterfaceName string, locatorAddress string) error {
	settings, err := kvclient.Settings(ctx)
	if err != nil {
		return err
	}

	var wgInterface *sv1.WireguardInterface
	for _, inf := range settings.SecureTunnelingSettings.WireguardInterfaces {
		if inf.Name == wgInterfaceName {
			wgInterface = inf
			break
		}
	}
	if wgInterface == nil {
		return errors.New("given wireguard interface not found in settings")
	}

	// find and remove the locator from the wireguard interface
	c.logger.WithField("locators", wgInterface.LocatorServers).Info("locators")
	for i, l := range wgInterface.LocatorServers {
		if l == locatorAddress {
			c.logger.WithField("i", i).Info("found locator")
			wgInterface.LocatorServers = slices.Delete(wgInterface.LocatorServers, i, i+1)
			break
		}
	}
	c.logger.WithField("locators", wgInterface.LocatorServers).Info("locators")

	// update settings config
	_, err = kvclient.Set(ctx, kvclient.DEFAULT_DEVICE_SETTINGS_KEY, settings)
	if err != nil {
		c.logger.WithError(err).Error("failed to update settings after removing locator")
		return err
	}

	// close connection to locator
	c.locatorController.Close(wgInterface, locatorAddress)

	return nil
}

func (c secureTunnelingController) BindSTUNServer(ctx context.Context, wgInterfaceName string, stunServerAddress string) error {
	settings, err := kvclient.Settings(ctx)
	if err != nil {
		return err
	}

	var wgInterface *sv1.WireguardInterface
	for _, inf := range settings.SecureTunnelingSettings.WireguardInterfaces {
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
	_, err = kvclient.Set(ctx, kvclient.DEFAULT_DEVICE_SETTINGS_KEY, settings)
	if err != nil {
		c.logger.WithError(err).Error("failed to update settings after after binding STUN server")
		return err
	}

	return nil
}

// Get preferred outbound ip of this machine
func getOutboundIP() (string, error) {
	conn, err := net.Dial("udp4", "home-cloud.io:80")
	if err != nil {
		return "", err
	}
	defer conn.Close()
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String(), nil
}
