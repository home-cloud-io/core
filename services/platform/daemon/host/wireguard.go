package host

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	v1 "github.com/home-cloud-io/core/api/platform/daemon/v1"

	"github.com/steady-bytes/draft/pkg/chassis"
)

func AddWireguardInterface(ctx context.Context, logger chassis.Logger, def *v1.AddWireguardInterface) error {
	logger.Info("adding wireguard interface")

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

	// write the private key file
	err = os.MkdirAll(fullWireguardKeyPath(def.Interface.Name), 0700)
	if err != nil {
		return err
	}
	err = os.WriteFile(fullWireguardKeyPath(def.Interface.Name)+"/private", []byte(def.Interface.PrivateKey), 0600)
	if err != nil {
		return err
	}

	// configure NAT
	config.NAT.Enable = true
	config.NAT.ExternalInterface = "eth0"

	// add interface to existing array if necessary
	if config.NAT.InternalInterfaces == nil {
		config.NAT.InternalInterfaces = []string{def.Interface.Name}
	} else {
		config.NAT.InternalInterfaces = append(config.NAT.InternalInterfaces, def.Interface.Name)
	}

	// build out Wireguard peers
	peers := make([]WireguardPeer, len(def.Interface.Peers))
	for _, peer := range def.Interface.Peers {
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
	config.Wireguard.Interfaces[def.Interface.Name] = WireguardInterface{
		IPs:            def.Interface.Ips,
		ListenPort:     def.Interface.ListenPort,
		PrivateKeyFile: fullWireguardKeyPath(def.Interface.Name) + "/private",
		Peers:          peers,
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

	// add to daemon config
	wgConfig := &v1.WireguardConfig{}
	err = chassis.GetConfig().UnmarshalKey(WireguardConfigKey, wgConfig)
	if err != nil {
		return err
	}
	wgConfig.Interfaces = append(wgConfig.Interfaces, &v1.WireguardInterface{
		Id:   def.Interface.Id,
		Name: def.Interface.Name,
	})
	err = chassis.GetConfig().SetAndWrite(WireguardConfigKey, wgConfig)
	if err != nil {
		return err
	}

	return nil
}

func RemoveWireguardInterface(ctx context.Context, logger chassis.Logger, def *v1.RemoveWireguardInterface) error {
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
	err = os.RemoveAll(fullWireguardKeyPath(def.Name))
	if err != nil {
		return err
	}

	// remove interface from Wireguard config
	delete(config.Wireguard.Interfaces, def.Name)

	// remove interface from NAT config
	for i, inf := range config.NAT.InternalInterfaces {
		if inf == def.Name {
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

	// remove from daemon config
	wgConfig := &v1.WireguardConfig{}
	err = chassis.GetConfig().UnmarshalKey(WireguardConfigKey, wgConfig)
	if err != nil {
		return err
	}
	for i, inf := range wgConfig.Interfaces {
		if inf.Name == def.Name {
			wgConfig.Interfaces = append(wgConfig.Interfaces[:i], wgConfig.Interfaces[i+1:]...)
			break
		}
	}
	err = chassis.GetConfig().SetAndWrite(WireguardConfigKey, wgConfig)
	if err != nil {
		return err
	}

	return nil
}

func fullWireguardKeyPath(interfaceName string) string {
	return filepath.Join(WireguardKeyPath(), interfaceName)
}

func AddWireguardPeer(ctx context.Context, logger chassis.Logger, peer *v1.WireguardPeer) error {
	logger.Info("adding wireguard peer")

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

	// Adding peer to all `wg` interfaces. This will need to change when peering to other devices is built.
	// currently the interface name is unknown
	for key, inf := range config.Wireguard.Interfaces {
		inf.Peers = append(inf.Peers, WireguardPeer{
			PublicKey:  peer.PublicKey,
			AllowedIPs: peer.AllowedIps,
		})
		config.Wireguard.Interfaces[key] = inf
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
