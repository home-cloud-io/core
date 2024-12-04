package system

import (
	"context"
	"errors"

	"github.com/google/uuid"
	v1 "github.com/home-cloud-io/core/api/platform/server/v1"
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

	// send to daemon

	// grab remaining server config

	return nil, errors.New("implement me")
}
