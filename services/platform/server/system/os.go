package system

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/steady-bytes/draft/pkg/chassis"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	dv1 "github.com/home-cloud-io/core/api/platform/daemon/v1"
	opv1 "github.com/home-cloud-io/core/services/platform/operator/api/v1"
	k8sclient "github.com/home-cloud-io/core/services/platform/server/k8s-client"
)

type (
	OS interface {
		// SystemStats calls the daemon and returns the reported system stats (CPU, RAM, etc.)
		SystemStats(ctx context.Context, loger chassis.Logger) (*dv1.SystemStats, error)
		// EnableWireguard will initialize the Wireguard server
		EnableWireguard(ctx context.Context, logger chassis.Logger) error
		// DisableWireguard will disable the Wireguard server
		DisableWireguard(ctx context.Context, logger chassis.Logger) error
	}
)

func (c *controller) SystemStats(ctx context.Context, loger chassis.Logger) (*dv1.SystemStats, error) {
	resp, err := c.daemonClient.SystemStats(ctx, &connect.Request[dv1.SystemStatsRequest]{})
	if err != nil {
		return nil, err
	}
	return resp.Msg.Stats, nil
}

func (c *controller) EnableWireguard(ctx context.Context, logger chassis.Logger) error {
	var (
		err error
	)

	// TODO: check for existing secret and use that if it exists
	key, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		logger.WithError(err).Error("failed to generate wireguard private key")
		return err
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-private-key", DefaultWireguardInterface),
			Namespace: k8sclient.HomeCloudNamespace,
		},
		StringData: map[string]string{
			"privateKey": key.String(),
		},
	}
	err = c.k8sclient.Create(ctx, secret)
	if err != nil {
		logger.WithError(err).Error("failed to create wireguard private key secret")
		return err
	}

	wgInterface := &opv1.Wireguard{
		ObjectMeta: metav1.ObjectMeta{
			Name:      DefaultWireguardInterface,
			Namespace: k8sclient.HomeCloudNamespace,
		},
		Spec: opv1.WireguardSpec{
			ID:   uuid.New().String(),
			Name: DefaultWireguardInterface,
			PrivateKeySecret: opv1.SecretReference{
				Name:      secret.Name,
				Namespace: &secret.Namespace,
				DataKey:   "privateKey",
			},
			// TODO: determine this using daemon
			NATInterface: "ens18",
			STUNServer:   DefaultSTUNServerAddress,
			Address:      "10.100.0.1/24",
			ListenPort:   51820,
			Locators: []string{
				DefaultLocatorAddress,
			},
			Peers: []opv1.PeerSpec{},
		},
	}
	err = c.k8sclient.Create(ctx, wgInterface)
	if err != nil {
		logger.WithError(err).Error("failed to create wireguard resource")
		return err
	}

	return nil
}

func (c *controller) DisableWireguard(ctx context.Context, logger chassis.Logger) error {
	var (
		err error
	)

	wgInterface := &opv1.Wireguard{
		ObjectMeta: metav1.ObjectMeta{
			Name:      DefaultWireguardInterface,
			Namespace: "home-cloud-system",
		},
	}
	err = c.k8sclient.Delete(ctx, wgInterface)
	if err != nil {
		logger.WithError(err).Error("failed to delete wireguard resource")
		return err
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-private-key", DefaultWireguardInterface),
			Namespace: "home-cloud-system",
		},
	}
	err = c.k8sclient.Delete(ctx, secret)
	if err != nil {
		logger.WithError(err).Error("failed to delete wireguard private key secret")
		return err
	}

	return nil
}
