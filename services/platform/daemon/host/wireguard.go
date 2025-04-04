package host

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	v1 "github.com/home-cloud-io/core/api/platform/daemon/v1"

	"github.com/steady-bytes/draft/pkg/chassis"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

type (
	WireguardController interface {
		AddInterface(ctx context.Context, logger chassis.Logger, wgInterface *v1.WireguardInterface) (publicKey string, err error)
		RemoveInterface(ctx context.Context, logger chassis.Logger, wgInterfaceName string) error
		AddPeer(ctx context.Context, logger chassis.Logger, wgInterfaceName string, peer *v1.WireguardPeer) (addresses []string, err error)
	}
	wireguardController struct{}
)

func NewWireguardController() WireguardController {
	return wireguardController{}
}

func (c wireguardController) AddInterface(ctx context.Context, logger chassis.Logger, wgInterface *v1.WireguardInterface) (publicKey string, err error) {
	logger.Info("adding wireguard interface")

	// read config
	config := NetworkingConfig{}
	f, err := os.ReadFile(NetworkingConfigFile())
	if err != nil {
		return "", err
	}
	err = json.Unmarshal(f, &config)
	if err != nil {
		return "", err
	}

	// check to see if the interface already exists
	_, ok := config.Wireguard.Interfaces[wgInterface.Name]
	if ok {
		return "", errors.New("wireguard interface already exists")
	}

	// generate a private key and write to file system
	privateKey, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		return "", err
	}
	err = os.MkdirAll(fullWireguardKeyPath(wgInterface.Name), 0700)
	if err != nil {
		return "", err
	}
	err = os.WriteFile(fullWireguardKeyPath(wgInterface.Name)+"/private", []byte(privateKey.String()), 0600)
	if err != nil {
		return "", err
	}

	// configure NAT
	config.NAT.Enable = true
	config.NAT.ExternalInterface = "eth0"

	// add interface to existing array if necessary
	if config.NAT.InternalInterfaces == nil {
		config.NAT.InternalInterfaces = []string{wgInterface.Name}
	} else {
		config.NAT.InternalInterfaces = append(config.NAT.InternalInterfaces, wgInterface.Name)
	}

	// build out Wireguard peers
	peers := make([]WireguardPeer, len(wgInterface.Peers))
	for _, peer := range wgInterface.Peers {
		peers = append(peers, WireguardPeer{
			PublicKey:  peer.PublicKey,
			AllowedIPs: peer.AllowedIps,
		})
	}

	// make sure map isn't nil
	if config.Wireguard.Interfaces == nil {
		config.Wireguard.Interfaces = make(map[string]WireguardInterface)
	}

	// add interface to existing map
	config.Wireguard.Interfaces[wgInterface.Name] = WireguardInterface{
		IPs:            wgInterface.Ips,
		ListenPort:     wgInterface.ListenPort,
		PrivateKeyFile: fullWireguardKeyPath(wgInterface.Name) + "/private",
		Peers:          peers,
	}

	// write config
	b, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return "", err
	}

	err = os.WriteFile(NetworkingConfigFile(), b, 0777)
	if err != nil {
		return "", err
	}

	err = RebuildAndSwitchOS(ctx, logger)
	if err != nil {
		return "", err
	}

	return privateKey.PublicKey().String(), nil
}

func (c wireguardController) RemoveInterface(ctx context.Context, logger chassis.Logger, wgInterfaceName string) error {
	logger.Info("removing wireguard interface")

	// read config
	config := NetworkingConfig{}
	f, err := os.ReadFile(NetworkingConfigFile())
	if err != nil {
		return err
	}
	err = json.Unmarshal(f, &config)
	if err != nil {
		return err
	}

	// remove private key file
	err = os.RemoveAll(fullWireguardKeyPath(wgInterfaceName))
	if err != nil {
		return err
	}

	// remove interface from Wireguard config
	delete(config.Wireguard.Interfaces, wgInterfaceName)

	// remove interface from NAT config
	for i, inf := range config.NAT.InternalInterfaces {
		if inf == wgInterfaceName {
			config.NAT.InternalInterfaces = append(config.NAT.InternalInterfaces[:i], config.NAT.InternalInterfaces[i+1:]...)
			break
		}
	}

	// if we just removed the last Wireguard interface we can disable NAT
	if len(config.Wireguard.Interfaces) == 0 {
		config.NAT.Enable = false
		config.NAT.ExternalInterface = ""
	}

	// write config
	b, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	err = os.WriteFile(NetworkingConfigFile(), b, 0777)
	if err != nil {
		return err
	}

	err = RebuildAndSwitchOS(ctx, logger)
	if err != nil {
		return err
	}

	return nil
}

func (c wireguardController) AddPeer(ctx context.Context, logger chassis.Logger, wgInterfaceName string, wgPeer *v1.WireguardPeer) (addresses []string, err error) {
	// read config
	config := NetworkingConfig{}
	f, err := os.ReadFile(NetworkingConfigFile())
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(f, &config)
	if err != nil {
		return nil, err
	}

	wgInterface, ok := config.Wireguard.Interfaces[wgInterfaceName]
	if !ok {
		return nil, errors.New("wireguard interface does not exist")
	}

	existingAddresses := []string{}
	for _, existingPeer := range wgInterface.Peers {
		if existingPeer.PublicKey == wgPeer.PublicKey {
			return nil, errors.New("peer with requested public key already registered with given interface")
		}
		existingAddresses = append(existingAddresses, existingPeer.AllowedIPs...)
	}
	wgInterface.Peers = append(wgInterface.Peers, WireguardPeer{
		PublicKey:  wgPeer.PublicKey,
		AllowedIPs: wgPeer.AllowedIps,
	})
	config.Wireguard.Interfaces[wgInterfaceName] = wgInterface

	// find the first unused ip in the address space
	var address string
	for i := 2; i < 255; i++ {
		address := fmt.Sprintf("10.100.0.%d/32", i)
		if !slices.Contains(existingAddresses, address) {
			break
		}
	}
	if address == "" {
		return nil, errors.New("no available ip address for peer found")
	}

	// write config
	b, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return nil, err
	}

	err = os.WriteFile(NetworkingConfigFile(), b, 0777)
	if err != nil {
		return nil, err
	}

	err = RebuildAndSwitchOS(ctx, logger)
	if err != nil {
		return nil, err
	}

	return []string{address}, nil
}

func fullWireguardKeyPath(interfaceName string) string {
	return filepath.Join(WireguardKeyPath(), interfaceName)
}
