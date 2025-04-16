package communicate

import (
	"context"

	v1 "github.com/home-cloud-io/core/api/platform/daemon/v1"
)

func (c *client) addWireguardInterface(ctx context.Context, msg *v1.ServerMessage) {
	def := msg.GetAddWireguardInterface()
	resp := &v1.DaemonMessage{
		Message: &v1.DaemonMessage_WireguardInterfaceAdded{
			WireguardInterfaceAdded: &v1.WireguardInterfaceAdded{
				WireguardInterface: def.Interface.Name,
			},
		},
	}

	publicKey, err := c.secureTunneling.AddInterface(ctx, def.Interface)
	if err != nil {
		c.logger.WithError(err).Error("failed to add wireguard interface")
		msg := resp.GetWireguardInterfaceAdded()
		msg.Error = err.Error()
	}
	message := resp.GetWireguardInterfaceAdded()
	message.PublicKey = publicKey

	c.Send(resp, msg)
}

func (c *client) removeWireguardInterface(ctx context.Context, msg *v1.ServerMessage) {
	def := msg.GetRemoveWireguardInterface()
	resp := &v1.DaemonMessage{
		Message: &v1.DaemonMessage_WireguardInterfaceRemoved{
			WireguardInterfaceRemoved: &v1.WireguardInterfaceRemoved{
				WireguardInterface: def.Name,
			},
		},
	}

	err := c.secureTunneling.RemoveInterface(ctx, def.Name)
	if err != nil {
		c.logger.WithError(err).Error("failed to remove wireguard interface")
		msg := resp.GetWireguardInterfaceRemoved()
		msg.Error = err.Error()
	}

	c.Send(resp, msg)
}

func (c *client) addWireguardPeer(ctx context.Context, msg *v1.ServerMessage) {
	def := msg.GetAddWireguardPeer()
	resp := &v1.DaemonMessage{
		Message: &v1.DaemonMessage_WireguardPeerAdded{
			WireguardPeerAdded: &v1.WireguardPeerAdded{
				WireguardInterface: def.WireguardInterface,
				ClientPublicKey:    def.Peer.PublicKey,
			},
		},
	}

	addresses, dnsServers, err := c.secureTunneling.AddPeer(ctx, def.WireguardInterface, def.Peer)
	if err != nil {
		c.logger.WithError(err).Error("failed to add wireguard peer")
		msg := resp.GetWireguardPeerAdded()
		msg.Error = err.Error()
	}
	message := resp.GetWireguardPeerAdded()
	message.Addresses = addresses
	message.DnsServers = dnsServers

	c.Send(resp, msg)
}
