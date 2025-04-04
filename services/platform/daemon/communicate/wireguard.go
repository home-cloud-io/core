package communicate

import (
	"context"

	v1 "github.com/home-cloud-io/core/api/platform/daemon/v1"
)

func (c *client) addWireguardInterface(ctx context.Context, def *v1.AddWireguardInterface) {
	resp := &v1.DaemonMessage{
		Message: &v1.DaemonMessage_WireguardInterfaceAdded{
			WireguardInterfaceAdded: &v1.WireguardInterfaceAdded{
				WireguardInterface: def.Interface.Name,
			},
		},
	}

	publicKey, err := c.remoteAccess.AddInterface(ctx, def.Interface)
	if err != nil {
		c.logger.WithError(err).Error("failed to add wireguard interface")
		msg := resp.GetWireguardInterfaceAdded()
		msg.Error = err.Error()
	}
	msg := resp.GetWireguardInterfaceAdded()
	msg.PublicKey = publicKey

	c.Send(resp)
}

func (c *client) removeWireguardInterface(ctx context.Context, def *v1.RemoveWireguardInterface) {
	resp := &v1.DaemonMessage{
		Message: &v1.DaemonMessage_WireguardInterfaceRemoved{
			WireguardInterfaceRemoved: &v1.WireguardInterfaceRemoved{
				WireguardInterface: def.Name,
			},
		},
	}

	err := c.remoteAccess.RemoveInterface(ctx, def.Name)
	if err != nil {
		c.logger.WithError(err).Error("failed to remove wireguard interface")
		msg := resp.GetWireguardInterfaceRemoved()
		msg.Error = err.Error()
	}

	c.logger.Info("sending WireguardInterfaceRemoved message")
	c.Send(resp)
}

func (c *client) addWireguardPeer(ctx context.Context, def *v1.AddWireguardPeer) {
	resp := &v1.DaemonMessage{
		Message: &v1.DaemonMessage_WireguardPeerAdded{
			WireguardPeerAdded: &v1.WireguardPeerAdded{
				WireguardInterface: def.WireguardInterface,
				ClientPublicKey:    def.Peer.PublicKey,
			},
		},
	}

	addresses, dnsServers, err := c.remoteAccess.AddPeer(ctx, def.WireguardInterface, def.Peer)
	if err != nil {
		c.logger.WithError(err).Error("failed to add wireguard peer")
		msg := resp.GetWireguardPeerAdded()
		msg.Error = err.Error()
	}
	msg := resp.GetWireguardPeerAdded()
	msg.Addresses = addresses
	msg.DnsServers = dnsServers

	c.Send(resp)
}
