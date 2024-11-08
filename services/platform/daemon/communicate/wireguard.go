package communicate

import (
	"context"
	"fmt"

	v1 "github.com/home-cloud-io/core/api/platform/daemon/v1"
	"github.com/home-cloud-io/core/services/platform/daemon/host"
)

type (
	NetworkingConfig struct {
		Hostname       string `json:"hostName"`
		Domain         string
		NetworkManager struct {
			Enable bool
		} `json:"networkmanager"`
		Wireless struct {
			Enable bool
		}
		Firewall struct {
			Enable bool
		}
		NAT struct {
			Enable             bool
			ExternalInterface  string   `json:"externalInterfaces"`
			InternalInterfaces []string `json:"internalInterfaces"`
		} `json:"nat"`
		Wireguard struct {
			Interfaces map[string]WireguardInterface
		}
	}
	WireguardInterface struct {
		IPs            []string `json:"ips"`
		ListenPort     uint32   `json:"listenPort"`
		PrivateKeyFile string   `json:"privateKeyFile"`
		Peers          []WireguardPeer
	}
	WireguardPeer struct {
		PublicKey  string   `json:"publicKey"`
		AllowedIPs []string `json:"allowedIPs"`
	}
)

func (c *client) enableWireguard(ctx context.Context, def *v1.EnableWireguardCommand) {
	var (
		replacers = []host.Replacer{
			func(line string) string {
				if line == "  # networking.nat.enable = true;" {
					return "  networking.nat.enable = true;"
				}
				return line
			},
			func(line string) string {
				if line == "  # networking.nat.externalInterface = \"eth0\";" {
					return "  networking.nat.externalInterface = \"eth0\";"
				}
				return line
			},
			func(line string) string {
				if line == "  # networking.nat.internalInterfaces = [ \"wg0\" ];" {
					return "  networking.nat.internalInterfaces = [ \"wg0\" ];"
				}
				return line
			},
			func(line string) string {
				if line == "  # networking.wireguard.interfaces = {};" {
					lines := "  networking.wireguard.interfaces = {"
					for name, in := range def.Config.Interfaces {
						lines += fmt.Sprintf("\n    %s = {", name)
						lines += "\n      ips = [ "
						for _, ip := range in.Ips {
							lines += fmt.Sprintf("\"%s\" ", ip)
						}
						lines += "];"
						lines += fmt.Sprintf("\n      listenPort = %d;", in.ListenPort)
						lines += fmt.Sprintf("\n      privateKeyFile = \"%s\";", in.PrivateKeyFile)
						// TODO: pull peers from config here or add them later?
						lines += "\n      peers = [];"
						lines += "\n    };"
					}
					lines += "\n  };"
					return lines
				}
				return line
			},
		}
	)

	err := host.LineByLineReplace(host.NixosConfigFile, replacers)
	if err != nil {
		c.logger.WithError(err).Error("failed to write Wireguard config to NixOS configuration")
		err = c.stream.Send(&v1.DaemonMessage{
			Message: &v1.DaemonMessage_WireguardEnabled{
				WireguardEnabled: &v1.WireguardEnabled{
					Error: &v1.DaemonError{
						Error: fmt.Sprintf("failed to write Wireguard config to NixOS configuration: %s", err.Error()),
					},
				},
			},
		})
		if err != nil {
			c.logger.WithError(err).Error("failed to send error message to server")
		}
		return
	}

	err = host.RebuildAndSwitchOS(ctx, c.logger)
	if err != nil {
		c.logger.WithError(err).Error("failed to rebuild and switch to NixOS configuration")
		err = c.stream.Send(&v1.DaemonMessage{
			Message: &v1.DaemonMessage_WireguardEnabled{
				WireguardEnabled: &v1.WireguardEnabled{
					Error: &v1.DaemonError{
						Error: fmt.Sprintf("failed to rebuild and switch to NixOS configuration: %s", err.Error()),
					},
				},
			},
		})
		if err != nil {
			c.logger.WithError(err).Error("failed to send error message to server")
		}
		return
	}

	err = c.stream.Send(&v1.DaemonMessage{
		Message: &v1.DaemonMessage_WireguardEnabled{},
	})
	if err != nil {
		c.logger.WithError(err).Error("failed to send complete message to server")
	}
}
