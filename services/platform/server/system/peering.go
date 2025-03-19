package system

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/google/uuid"
	dv1 "github.com/home-cloud-io/core/api/platform/daemon/v1"
	v1 "github.com/home-cloud-io/core/api/platform/server/v1"
	"github.com/home-cloud-io/core/services/platform/server/async"
	kvclient "github.com/home-cloud-io/core/services/platform/server/kv-client"
	"github.com/steady-bytes/draft/pkg/chassis"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

type (
	Peering interface {
		RegisterPeer(ctx context.Context, logger chassis.Logger) (*v1.PeerConfiguration, error)
	}
)

const (
	ErrFailedToGenPubKey     = "failed to generate public key"
	ErrFailedToGenPrivKey    = "failed to generate private key"
	ErrFailedToSetPeerConfig = "failed to save peer config to key/val store"
)

func (c *controller) RegisterPeer(ctx context.Context, logger chassis.Logger) (*v1.PeerConfiguration, error) {
	// create pub/priv key
	pubKey, err := wgtypes.GenerateKey()
	if err != nil {
		logger.WithError(err).Error(ErrFailedToGenPubKey)
		return nil, errors.New(ErrFailedToGenPubKey)
	}

	privKey, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		logger.WithError(err).Error(ErrFailedToGenPrivKey)
		return nil, errors.New(ErrFailedToGenPrivKey)
	}

	clientID := uuid.NewString()

	cfg := &v1.PeerConfiguration{
		Id:         clientID,
		PublicKey:  pubKey.String(),
		PrivateKey: privKey.String(),
	}

	// save to blueprint
	_, err = kvclient.Set(ctx, clientID, cfg)
	if err != nil {
		logger.WithError(err).Error(ErrFailedToSetPeerConfig)
		return nil, errors.New(ErrFailedToSetPeerConfig)
	}

	listener := async.RegisterListener(ctx, c.broadcaster, &async.ListenerOptions[*dv1.WireguardPeerAdded]{
		Callback: func(event *dv1.WireguardPeerAdded) (bool, error) {
			if event.Error != nil {
				return true, fmt.Errorf(event.Error.Error)
			}
			return true, nil
		},
		Timeout: 30 * time.Second,
	})
	err = com.Send(&dv1.ServerMessage{
		Message: &dv1.ServerMessage_AddWireguardPeer{
			AddWireguardPeer: &dv1.AddWireguardPeer{
				Peer: &dv1.WireguardPeer{
					PublicKey: pubKey.String(),
					AllowedIps: []string{"10.100.0.1/32"},
				},
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

	adder, err := getOutboundIP()
	if err != nil {
		logger.WithError(err).Error("failed to get device ip")
		return nil, err
	}

	// get locator address
	settings := &v1.DeviceSettings{}
	err = kvclient.Get(ctx, kvclient.DEFAULT_DEVICE_SETTINGS_KEY, settings)
	if err != nil {
		logger.WithError(err).Warn("failed to get device settings when loading locators")
		return nil, err
	}
	cfg.ClientDetails = &v1.ClientRegistrationDetails{
		ServerAddress:  adder,
		// TODO: get this here?
		// LocatorAddress: settings.LocatorSettings.Locators[0].Address,
		LocatorAddress: "https://locator1.home-cloud.io",
	}

	// get server public key
	wgConfig := &dv1.WireguardConfig{}
	err = kvclient.Get(ctx, kvclient.WIREGUARD_CONFIG_KEY, wgConfig)
	if err != nil {
		logger.WithError(err).Warn("failed to get wireguard config")
		return nil, err
	}

	key, err := wgtypes.ParseKey(wgConfig.Interfaces[0].PrivateKey)
	if err != nil {
		logger.WithError(err).Error("failed to parse server key")
		return nil, err
	}
	cfg.ServerPublicKey = key.PublicKey().String()

	return cfg, nil
}

func getOutboundIP() (string, error) {
	conn, err := net.Dial("udp", "home-cloud.io:80")
	if err != nil {
		return "", err
	}
	defer conn.Close()
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String(), nil
}
