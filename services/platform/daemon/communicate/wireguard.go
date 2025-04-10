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
	// TODO: this message doesn't get to or is not processed by the server but only on the actual home cloud server. it works when running locally
	/*
I start to get these messages below on the daemin

5:30PM ERR stream failure error="internal: stream error: stream ID 1; NO_ERROR; received from peer" function=Listen service=home-cloud-daemon
5:30PM ERR failed to send message to server error="no stream" function=Send service=home-cloud-daemon
5:30PM INF listening for messages from server function=listen service=home-cloud-daemon
5:30PM ERR failed to send message to server error="no stream" function=Send service=home-cloud-daemon
5:30PM ERR stream failure error="unavailable: HTTP status 503 Service Unavailable" function=Listen service=home-cloud-daemon

And these on the server

{"level":"info","service":"server","function":"DisableSecureTunnelling","time":"2025-04-04T22:27:46Z","message":"disabling secure tunnelling"}
{"level":"error","service":"server","error":"canceled: client disconnected","function":"Communicate","time":"2025-04-04T22:30:30Z","message":"failed to recieve message"}
{"level":"error","service":"server","error":"canceled: context canceled","function":"DisableSecureTunnelling","time":"2025-04-04T22:30:58Z","message":" failed to disable secure tunnelling"}

And this on blueprint

{"level":"error","service":"blueprint","error":"canceled: client disconnected","function":"Synchronize","time":"2025-04-04T22:30:26Z","message":"connection error"}
	*/
	err = c.SendWithError(resp)
	if err != nil {
		c.logger.WithError(err).Error("failed to send WireguardInterfaceRemoved message")
	}
	c.logger.Error("finished sending WireguardInterfaceRemoved message")
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
