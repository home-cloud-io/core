package system

import (
	"context"
	"errors"
	"fmt"
	"time"

	dv1 "github.com/home-cloud-io/core/api/platform/daemon/v1"
	v1 "github.com/home-cloud-io/core/api/platform/server/v1"
	"github.com/home-cloud-io/core/services/platform/server/async"
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

	if !settings.RemoteAccessSettings.Enabled {
		return nil, errors.New("secure tunnelling not enabled")
	}

	// TODO: handle multiple interfaces
	wgInterface := settings.RemoteAccessSettings.WireguardInterfaces[0]

	// create pub/priv key
	privKey, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		logger.WithError(err).Error(ErrFailedToGenPrivKey)
		return nil, errors.New(ErrFailedToGenPrivKey)
	}

	peerConfig := &v1.RegisterPeerResponse{
		PrivateKey: privKey.String(),
		PublicKey:  privKey.PublicKey().String(),
		ServerPublicKey: wgInterface.PublicKey,
		ServerId:        wgInterface.Id,
		LocatorServers:  wgInterface.LocatorServers,
	}

	listener := async.RegisterListener(ctx, c.broadcaster, &async.ListenerOptions[*dv1.WireguardPeerAdded]{
		Callback: func(event *dv1.WireguardPeerAdded) (bool, error) {
			if event.ClientPublicKey != peerConfig.PublicKey || event.WireguardInterface != wgInterface.Name {
				return false, nil
			}
			if event.Error != "" {
				return true, fmt.Errorf(event.Error)
			}
			peerConfig.Addresses = event.Addresses
			peerConfig.DnsServers = event.DnsServers
			return true, nil
		},
		Timeout: 30 * time.Second,
	})
	err = com.Send(&dv1.ServerMessage{
		Message: &dv1.ServerMessage_AddWireguardPeer{
			AddWireguardPeer: &dv1.AddWireguardPeer{
				Peer: &dv1.WireguardPeer{
					PublicKey:  peerConfig.PublicKey,
					AllowedIps: peerConfig.Addresses,
				},
				WireguardInterface: wgInterface.Name,
			},
		},
	})
	if err != nil {
		return nil, err
	}
	err = listener.Listen(ctx)
	if err != nil {
		return nil, err
	}

	return peerConfig, nil
}
