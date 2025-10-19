package host

// TODO: shift these to protos and r/w them with blueprint?

type (
	NetworkingConfig struct {
		Hostname       string                         `json:"hostName"`
		Domain         string                         `json:"domain"`
		NetworkManager NetworkingConfigNetworkManager `json:"networkmanager"`
		Wireless       NetworkingConfigWireless       `json:"wireless"`
		Firewall       NetworkingConfigFirewall       `json:"firewall"`
		NAT            NetworkingConfigNAT            `json:"nat"`
		Wireguard      NetworkingConfigWireguard      `json:"wireguard"`
	}
	NetworkingConfigNetworkManager struct {
		Enable bool `json:"enable"`
	}
	NetworkingConfigWireless struct {
		Enable bool `json:"enable"`
	}
	NetworkingConfigFirewall struct {
		Enable bool `json:"enable"`
	}
	NetworkingConfigNAT struct {
		Enable             bool     `json:"enable"`
		ExternalInterface  string   `json:"externalInterface,omitempty"`
		InternalInterfaces []string `json:"internalInterfaces,omitempty"`
	}
	NetworkingConfigWireguard struct {
		Interfaces map[string]WireguardInterface `json:"interfaces,omitempty"`
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
