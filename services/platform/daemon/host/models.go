package host

type (
	BootConfig struct {
		Loader BootConfigLoader `json:"loader"`
		BCache BootConfigBCache `json:"bcache"`
	}
	BootConfigLoader struct {
		SystemdBoot BootConfigLoaderSystemdBoot `json:"systemd-boot"`
	}
	BootConfigLoaderSystemdBoot struct {
		Enable bool `json:"enable"`
	}
	BootConfigBCache struct {
		Enable bool `json:"enable"`
	}

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

	SecurityConfig struct {
		Sudo SecurityConfigSudo `json:"sudo"`
	}
	SecurityConfigSudo struct {
		WheelNeedsPassword bool `json:"wheelNeedsPassword"`
	}

	ServicesConfig struct {
		Resolved ServicesConfigResolved `json:"resolved"`
		K3s      ServicesConfigK3s      `json:"k3s"`
		OpenSSH  ServicesConfigOpenSSH  `json:"openssh"`
		Avahi    ServicesConfigAvahi    `json:"avahi"`
	}
	ServicesConfigResolved struct {
		Enable  bool     `json:"enable"`
		Domains []string `json:"domains"`
	}
	ServicesConfigK3s struct {
		Enable     bool   `json:"enable"`
		Role       string `json:"role"`
		ExtraFlags string `json:"extraFlags"`
	}
	ServicesConfigOpenSSH struct {
		Enable bool `json:"enable"`
	}
	ServicesConfigAvahi struct {
		Enable   bool                       `json:"enable"`
		IPv4     bool                       `json:"ipv4"`
		IPv6     bool                       `json:"ipv6"`
		NSSmDNS4 bool                       `json:"nssmdns4"`
		Publish  ServicesConfigAvahiPublish `json:"publish"`
	}
	ServicesConfigAvahiPublish struct {
		Enable       bool `json:"enable"`
		Domain       bool `json:"domain"`
		Addresses    bool `json:"addresses"`
		UserServices bool `json:"userServices"`
	}

	TimeConfig struct {
		TimeZone string `json:"timeZone"`
	}

	UsersConfig struct {
		Users map[string]User `json:"users"`
	}
	User struct {
		IsNormalUser bool        `json:"isNormalUser"`
		ExtraGroups  []string    `json:"extraGroups"`
		OpenSSH      UserOpenSSH `json:"openssh"`
	}
	UserOpenSSH struct {
		AuthorizedKeys UserOpenSSHAuthorizedKeys `json:"authorizedKeys"`
	}
	UserOpenSSHAuthorizedKeys struct {
		Keys []string `json:"keys"`
	}
)
