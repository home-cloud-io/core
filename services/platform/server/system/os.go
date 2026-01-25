package system

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"github.com/steady-bytes/draft/pkg/chassis"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	dv1 "github.com/home-cloud-io/core/api/platform/daemon/v1"
	v1 "github.com/home-cloud-io/core/api/platform/server/v1"
	opv1 "github.com/home-cloud-io/core/services/platform/operator/api/v1"
	k8sclient "github.com/home-cloud-io/core/services/platform/server/k8s-client"
)

type (
	OS interface {
		// CheckForOSUpdates...
		CheckForOSUpdates(ctx context.Context, logger chassis.Logger) (*v1.CheckForSystemUpdatesResponse, error)
		// AutoUpdateOS will check for and install any OS (including Daemon) updates on a schedule. It is
		// designed to be called at bootup.
		AutoUpdateOS(logger chassis.Logger)
		// UpdateOS will check for and install any OS (including Daemon) updates one time.
		UpdateOS(ctx context.Context, logger chassis.Logger) error
		// SystemStats...
		SystemStats(ctx context.Context, loger chassis.Logger) (*dv1.SystemStats, error)
		// EnableWireguard will initialize the Wireguard server
		EnableWireguard(ctx context.Context, logger chassis.Logger) error
		// DisableWireguard will disable the Wireguard server
		DisableWireguard(ctx context.Context, logger chassis.Logger) error
	}
)

var CurrentStats *dv1.SystemStats

// OS

func (c *controller) CheckForOSUpdates(ctx context.Context, logger chassis.Logger) (*v1.CheckForSystemUpdatesResponse, error) {
	logger.Info("checking for os updates")

	// TODO: check by calling system service (talos)
	// c.daemonClient.

	return nil, nil
}

func (c *controller) AutoUpdateOS(logger chassis.Logger) {
	cr := cron.New()
	f := func() {
		ctx := context.Background()
		err := c.UpdateOS(ctx, logger)
		if err != nil {
			logger.WithError(err).Error("failed to run auto os update job")
		}
	}
	cron := chassis.GetConfig().GetString(osAutoUpdateCronConfigKey)
	logger.WithField("cron", cron).Info("setting os auto-update interval")
	_, err := cr.AddFunc(cron, f)
	if err != nil {
		logger.WithError(err).Panic("failed to initialize auto-update for os")
	}
	cr.Start()
}

func (u *controller) UpdateOS(ctx context.Context, logger chassis.Logger) error {
	logger.Info("updating os")

	// TODO: update by calling system service (talos)

	return nil
}

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
