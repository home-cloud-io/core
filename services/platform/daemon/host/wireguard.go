package host

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	v1 "github.com/home-cloud-io/core/api/platform/daemon/v1"

	"github.com/steady-bytes/draft/pkg/chassis"
)

type (
	NetworkingConfig struct {
		Hostname       string `json:"hostName"`
		Domain         string `json:"domain"`
		NetworkManager struct {
			Enable bool `json:"enable"`
		} `json:"networkmanager"`
		Wireless struct {
			Enable bool `json:"enable"`
		} `json:"wireless"`
		Firewall struct {
			Enable bool `json:"enable"`
		} `json:"firewall"`
		NAT struct {
			Enable             bool     `json:"enable"`
			ExternalInterface  string   `json:"externalInterfaces,omitempty"`
			InternalInterfaces []string `json:"internalInterfaces,omitempty"`
		} `json:"nat"`
		Wireguard struct {
			Interfaces map[string]WireguardInterface `json:"interfaces"`
		} `json:"wireguard"`
	}
	WireguardInterface struct {
		IPs            []string        `json:"ips"`
		ListenPort     uint32          `json:"listenPort"`
		PrivateKeyFile string          `json:"privateKeyFile"`
		Peers          []WireguardPeer `json:"peers"`
	}
	WireguardPeer struct {
		PublicKey  string   `json:"publicKey"`
		AllowedIPs []string `json:"allowedIPs"`
	}
)

func AddWireguardInterface(ctx context.Context, logger chassis.Logger, def *v1.AddWireguardInterface) error {

	// read config
	config := NetworkingConfig{}
	f, err := os.ReadFile(NetworkingConfigFile)
	if err != nil {
		return err
	}
	err = json.Unmarshal(f, &config)
	if err != nil {
		return err
	}

	// write the private key file
	err = os.MkdirAll(wireguardKeyPath(def.Interface.Name), 0700)
	if err != nil {
		return err
	}
	err = os.WriteFile(wireguardKeyPath(def.Interface.Name)+"/private", []byte(def.Interface.PrivateKey), 0600)
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
	// add interface to existing map
	config.Wireguard.Interfaces[def.Interface.Name] = WireguardInterface{
		IPs:            def.Interface.Ips,
		ListenPort:     def.Interface.ListenPort,
		PrivateKeyFile: def.Interface.PrivateKeyFile,
		Peers:          peers,
	}

	// write config
	b, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	err = os.WriteFile(NetworkingConfigFile, b, 0777)
	if err != nil {
		return err
	}

	err = RebuildAndSwitchOS(ctx, logger)
	if err != nil {
		return err
	}

	return nil
}

func RemoveWireguardInterface(ctx context.Context, logger chassis.Logger, def *v1.RemoveWireguardInterface) error {

	// read config
	config := NetworkingConfig{}
	f, err := os.ReadFile(NetworkingConfigFile)
	if err != nil {
		return err
	}
	err = json.Unmarshal(f, &config)
	if err != nil {
		return err
	}

	// remove private key file
	err = os.RemoveAll(wireguardKeyPath(def.Name))
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
	err = os.WriteFile(NetworkingConfigFile, b, 0777)
	if err != nil {
		return err
	}

	err = RebuildAndSwitchOS(ctx, logger)
	if err != nil {
		return err
	}

	return nil
}

func wireguardKeyPath(interfaceName string) string {
	return filepath.Join(WireguardKeyPath, interfaceName)
}
