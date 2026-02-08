package system

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/google/uuid"
	"github.com/steady-bytes/draft/pkg/chassis"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "github.com/home-cloud-io/core/api/platform/server/v1"
	opv1 "github.com/home-cloud-io/core/services/platform/operator/api/v1"
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

	// get wireguard server config
	wireguardServer := &opv1.Wireguard{}
	err := c.k8sclient.Get(ctx, types.NamespacedName{
		Name:      DefaultWireguardInterface,
		Namespace: "home-cloud-system",
	}, wireguardServer)
	if err != nil {
		logger.WithError(err).Error("failed to get wireguard server config")
		if kerrors.IsNotFound(err) {
			return nil, errors.New("secure tunneling not enabled")
		}
		return nil, err
	}

	// get wireguard server private key
	wireguardServerSecret := &corev1.Secret{}
	err = c.k8sclient.Get(ctx, types.NamespacedName{
		Name:      wireguardServer.Spec.PrivateKeySecret.Name,
		Namespace: *wireguardServer.Spec.PrivateKeySecret.Namespace,
	}, wireguardServerSecret)
	if err != nil {
		logger.WithError(err).Error("failed to get wireguard server secret")
		return nil, err
	}
	wireguardServerPrivateKey, err := wgtypes.ParseKey(string(wireguardServerSecret.Data["privateKey"]))
	if err != nil {
		logger.WithError(err).Error("failed to parse wireguard server private key")
		return nil, err
	}

	// create private key for peer
	peerPrivateKey, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		logger.WithError(err).Error(ErrFailedToGenPrivKey)
		return nil, errors.New(ErrFailedToGenPrivKey)
	}

	// find the first unused ip in the address space
	existingAddresses := []string{}
	for _, existingPeer := range wireguardServer.Spec.Peers {
		if existingPeer.PublicKey == peerPrivateKey.PublicKey().String() {
			return nil, errors.New("peer with requested public key already registered with given interface")
		}
		existingAddresses = append(existingAddresses, existingPeer.AllowedIPs...)
	}
	var peerAddress string
	for i := 2; i < 255; i++ {
		peerAddress = fmt.Sprintf("10.100.0.%d/32", i)
		if !slices.Contains(existingAddresses, peerAddress) {
			break
		}
	}
	if peerAddress == "" {
		return nil, errors.New("no available ip address for peer found")
	}

	// create peer secret
	peerSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-private-key", uuid.NewString()),
			Namespace: "home-cloud-system",
		},
		StringData: map[string]string{
			"privateKey": peerPrivateKey.String(),
		},
	}
	err = c.k8sclient.Create(ctx, peerSecret)
	if err != nil {
		logger.WithError(err).Error("failed to create wireguard peer private key secret")
		return nil, err
	}

	// add peer to server
	wireguardServer.Spec.Peers = append(wireguardServer.Spec.Peers, opv1.PeerSpec{
		PublicKey: peerPrivateKey.PublicKey().String(),
		PrivateKeySecret: &opv1.SecretReference{
			Name:      peerSecret.Name,
			Namespace: &peerSecret.Namespace,
			DataKey:   "privateKey",
		},
		AllowedIPs: []string{peerAddress},
	})
	err = c.k8sclient.Update(ctx, wireguardServer)
	if err != nil {
		logger.WithError(err).Error("failed to update wireguard server config")
		return nil, err
	}

	return &v1.RegisterPeerResponse{
		PrivateKey:      peerPrivateKey.String(),
		PublicKey:       peerPrivateKey.PublicKey().String(),
		ServerPublicKey: wireguardServerPrivateKey.PublicKey().String(),
		ServerId:        wireguardServer.Spec.ID,
		Addresses:       []string{peerAddress},
		LocatorServers:  wireguardServer.Spec.Locators,
		// TODO: configure home cloud managed DNS server
		// DnsServers: []string{},
	}, nil
}
