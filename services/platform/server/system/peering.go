package system

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	dv1 "github.com/home-cloud-io/core/api/platform/daemon/v1"
	v1 "github.com/home-cloud-io/core/api/platform/server/v1"
	kvclient "github.com/home-cloud-io/core/services/platform/server/kv-client"
	"github.com/steady-bytes/draft/pkg/chassis"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

type (
	Peering interface {
		RegisterPeer(ctx context.Context, logger chassis.Logger) (*v1.RegisterPeerResponse, error)
	}
)

const (
	ErrFailedToGenPubKey     = "failed to generate public key"
	ErrFailedToGenPrivKey    = "failed to generate private key"
	ErrFailedToSetPeerConfig = "failed to save peer config to key/val store"
)

func (c *controller) RegisterPeer(ctx context.Context, logger chassis.Logger) (*v1.RegisterPeerResponse, error) {

	settings := &v1.DeviceSettings{}
	err := kvclient.Get(ctx, kvclient.DEFAULT_DEVICE_SETTINGS_KEY, settings)
	if err != nil {
		logger.WithError(err).Warn("failed to get device settings when loading locators")
		return nil, err
	}

	if !settings.SecureTunnelingSettings.Enabled {
		return nil, errors.New("secure tunneling not enabled")
	}

	// TODO: handle multiple interfaces
	wgInterface := settings.SecureTunnelingSettings.WireguardInterfaces[0]

	// create pub/priv key
	privKey, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		logger.WithError(err).Error(ErrFailedToGenPrivKey)
		return nil, errors.New(ErrFailedToGenPrivKey)
	}

	peerConfig := &v1.RegisterPeerResponse{
		PrivateKey:      privKey.String(),
		PublicKey:       privKey.PublicKey().String(),
		ServerPublicKey: wgInterface.PublicKey,
		ServerId:        wgInterface.Id,
		LocatorServers:  wgInterface.LocatorServers,
	}

	resp, err := c.daemonClient.AddWireguardPeer(ctx, &connect.Request[dv1.AddWireguardPeerRequest]{
		Msg: &dv1.AddWireguardPeerRequest{
			Peer: &dv1.WireguardPeer{
				PublicKey:  peerConfig.PublicKey,
				AllowedIps: []string{"0.0.0.0/0"},
			},
			WireguardInterface: wgInterface.Name,
		},
	})
	if err != nil {
		return nil, err
	}
	peerConfig.Addresses = resp.Msg.Addresses
	peerConfig.DnsServers = resp.Msg.DnsServers

	return peerConfig, nil
}
