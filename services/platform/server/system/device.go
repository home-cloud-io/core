package system

import (
	"context"
	"net/http"

	"connectrpc.com/connect"
	"github.com/robfig/cron/v3"
	"github.com/steady-bytes/draft/pkg/chassis"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	dv1 "github.com/home-cloud-io/core/api/platform/daemon/v1"
	v1 "github.com/home-cloud-io/core/api/platform/server/v1"
	opv1 "github.com/home-cloud-io/core/services/platform/operator/api/v1"
	"github.com/home-cloud-io/core/services/platform/server/apps"
	k8sclient "github.com/home-cloud-io/core/services/platform/server/k8s-client"
	hstrings "github.com/home-cloud-io/core/services/platform/server/utils/strings"
)

type (
	Device interface {
		// GetServerSettings returns the current server settings after filtering out the
		// admin username and password.
		GetServerSettings(ctx context.Context, logger chassis.Logger) (*v1.DeviceSettings, error)
		// SetServerSettings updates the settings on the server with the given values
		SetServerSettings(ctx context.Context, logger chassis.Logger, settings *v1.DeviceSettings) error
		// AutoUpdate will check for and install the latest Home Cloud version on a schedule
		AutoUpdate(ctx context.Context, logger chassis.Logger, schedule string)
		// Update will check for and install the latest Home Cloud version once
		Update(ctx context.Context, logger chassis.Logger) error
		// GetComponentVersions returns all the versions of system components (server, daemon, etc.)
		GetComponentVersions(ctx context.Context, logger chassis.Logger) (*v1.GetComponentVersionsResponse, error)
	}
)

const (
	DefaultAutoUpdateSystemSchedule = "0 1 * * *"
	LatestReleaseManifestURL        = "https://github.com/home-cloud-io/core/releases/latest/download/manifest.yaml"

	ErrDeviceAlreadySetup     = "device already setup"
	ErrFailedToCreateSettings = "failed to create device settings"
	ErrFailedToGetSettings    = "failed to get device settings"
	ErrFailedToSetSettings    = "failed to save device settings"
)

// DEVICE

func (c *controller) GetServerSettings(ctx context.Context, logger chassis.Logger) (*v1.DeviceSettings, error) {
	settings, err := c.k8sclient.Settings(ctx)
	if err != nil {
		return nil, err
	}
	s := &v1.DeviceSettings{
		Hostname:                 settings.Hostname,
		AutoUpdateApps:           settings.AutoUpdateApps,
		AutoUpdateSystem:         settings.AutoUpdateSystem,
		AutoUpdateAppsSchedule:   settings.AutoUpdateAppsSchedule,
		AutoUpdateSystemSchedule: settings.AutoUpdateSystemSchedule,
		AppStores:                []*v1.AppStore{},
		SecureTunnelingSettings:  &v1.SecureTunnelingSettings{},
	}

	// set app stores
	for _, store := range settings.AppStores {
		s.AppStores = append(s.AppStores, &v1.AppStore{
			Url:         store.URL,
			RawChartUrl: store.RawChartURL,
		})
	}

	if len(s.AppStores) == 0 {
		s.AppStores = []*v1.AppStore{
			{
				Url:         apps.DefaultAppStoreURL,
				RawChartUrl: apps.DefaultAppStoreRawChartURL,
			},
		}
	}

	// get wireguard server config
	wireguardServer := &opv1.Wireguard{}
	err = c.k8sclient.Get(ctx, types.NamespacedName{
		Name:      DefaultWireguardInterface,
		Namespace: k8sclient.DefaultHomeCloudNamespace,
	}, wireguardServer)
	if err != nil {
		// not found means not enabled
		if kerrors.IsNotFound(err) {
			return s, nil
		}
		logger.WithError(err).Error("failed to get wireguard server config")
		return nil, err
	}

	// get wireguard server private key (so we can derive the public key)
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

	// save to settings object
	s.SecureTunnelingSettings = &v1.SecureTunnelingSettings{
		Enabled: true,
		WireguardInterfaces: []*v1.WireguardInterface{
			{
				Id:             wireguardServer.Spec.ID,
				Name:           wireguardServer.Name,
				Port:           int32(wireguardServer.Spec.ListenPort),
				PublicKey:      wireguardServerPrivateKey.PublicKey().String(),
				StunServer:     wireguardServer.Spec.STUNServer,
				LocatorServers: wireguardServer.Spec.Locators,
			},
		},
	}

	return s, nil
}

func (c *controller) SetServerSettings(ctx context.Context, logger chassis.Logger, settings *v1.DeviceSettings) error {

	install := &opv1.Install{}
	err := c.k8sclient.Get(ctx, types.NamespacedName{
		Namespace: k8sclient.DefaultHomeCloudNamespace,
		Name:      "install",
	}, install)
	if err != nil {
		logger.WithError(err).Error("failed to get install")
		return err
	}
	if install.Spec.Settings == nil {
		install.Spec.Settings = &opv1.SettingsSpec{}
	}

	if settings.AutoUpdateApps {
		c.actl.AutoUpdate(ctx, logger, hstrings.Default(settings.AutoUpdateAppsSchedule, apps.DefaultAutoUpdateAppsSchedule))
	}

	if settings.AutoUpdateSystem {
		c.AutoUpdate(ctx, logger, hstrings.Default(settings.AutoUpdateSystemSchedule, DefaultAutoUpdateSystemSchedule))
	}

	install.Spec.Settings.Hostname = settings.Hostname
	install.Spec.Settings.AutoUpdateApps = settings.AutoUpdateApps
	install.Spec.Settings.AutoUpdateSystem = settings.AutoUpdateSystem
	install.Spec.Settings.AutoUpdateAppsSchedule = settings.AutoUpdateAppsSchedule
	install.Spec.Settings.AutoUpdateSystemSchedule = settings.AutoUpdateSystemSchedule
	install.Spec.Settings.AppStores = []opv1.AppStore{}
	for _, store := range settings.AppStores {
		install.Spec.Settings.AppStores = append(install.Spec.Settings.AppStores, opv1.AppStore{
			URL:         store.Url,
			RawChartURL: store.RawChartUrl,
		})
	}

	return c.k8sclient.Update(ctx, install)
}

func (c *controller) AutoUpdate(ctx context.Context, logger chassis.Logger, schedule string) {
	f := func() {
		err := c.Update(context.Background(), logger)
		if err != nil {
			logger.WithError(err).Error("failed to run auto system update job")
		}
	}

	// create new if no current entry, otherwise remove old entry
	if c.cronID == 0 {
		c.cr = cron.New()
	} else {
		c.cr.Remove(c.cronID)
	}

	// add new entry
	logger.WithField("cron", schedule).Info("setting system auto-update interval")
	id, err := c.cr.AddFunc(schedule, f)
	if err != nil {
		logger.WithError(err).Panic("failed to initialize auto-update for system")
	}
	c.cronID = id

	// no-op if already started
	c.cr.Start()
}

func (c *controller) Update(ctx context.Context, logger chassis.Logger) error {
	logger.Info("running update check for system")

	install := &opv1.Install{}
	err := c.k8sclient.Get(ctx, types.NamespacedName{
		Namespace: k8sclient.DefaultHomeCloudNamespace,
		Name:      "install",
	}, install)
	if err != nil {
		logger.WithError(err).Error("failed to get install")
		return err
	}

	// get version manifest from repo
	resp, err := http.Get(LatestReleaseManifestURL)
	if err != nil {
		logger.WithError(err).Error("failed to download latest release manifest")
		return err
	}

	// decode body into spec
	dec := yaml.NewDecoder(resp.Body)
	latest := opv1.InstallSpec{}
	err = dec.Decode(&latest)
	if err != nil {
		logger.WithError(err).Error("failed to decode latest release manifest")
		return err
	}

	// TODO: should probably have a semver check to avoid downgrading?
	install.Spec.Version = latest.Version

	return c.k8sclient.Update(ctx, install)
}

func (c *controller) GetComponentVersions(ctx context.Context, logger chassis.Logger) (*v1.GetComponentVersionsResponse, error) {

	install := &opv1.Install{}
	err := c.k8sclient.Get(ctx, types.NamespacedName{
		Namespace: k8sclient.DefaultHomeCloudNamespace,
		Name:      "install",
	}, install)
	if err != nil {
		logger.WithError(err).Error("failed to get install")
		return nil, err
	}

	resp := &v1.GetComponentVersionsResponse{
		Platform: []*dv1.ComponentVersion{
			{
				Name:    "server",
				Domain:  "platform",
				Version: install.Status.Server.Tag,
			},
			{
				Name:    "mdns",
				Domain:  "platform",
				Version: install.Status.MDNS.Tag,
			},
			{
				Name:    "tunnel",
				Domain:  "platform",
				Version: install.Status.Tunnel.Tag,
			},
			{
				Name:    "daemon",
				Domain:  "platform",
				Version: install.Status.Daemon.Tag,
			},
			{
				Name:    "operator",
				Domain:  "platform",
				Version: install.Status.Operator.Tag,
			},
		},
		System: []*dv1.ComponentVersion{
			{
				Name:    "home-cloud",
				Domain:  "system",
				Version: install.Status.Version,
			},
			{
				Name:    "istio",
				Domain:  "system",
				Version: install.Status.Istio.Version,
			},
			{
				Name:    "gateway-api",
				Domain:  "system",
				Version: install.Status.GatewayAPI.Version,
			},
		},
	}

	k8sVersion, err := c.k8sclient.GetServerVersion(ctx)
	if err != nil {
		resp.System = append(resp.System, &dv1.ComponentVersion{
			Name:    "kubernetes",
			Domain:  "system",
			Version: err.Error(),
		})
	} else {
		resp.System = append(resp.System, &dv1.ComponentVersion{
			Name:    "kubernetes",
			Domain:  "system",
			Version: k8sVersion,
		})
	}

	osVersion, err := c.daemonClient.Version(ctx, connect.NewRequest(&dv1.VersionRequest{}))
	if err != nil {
		resp.System = append(resp.System, &dv1.ComponentVersion{
			Name:    "unknown",
			Domain:  "system",
			Version: err.Error(),
		})
	} else {
		resp.System = append(resp.System, &dv1.ComponentVersion{
			Name:    osVersion.Msg.Name,
			Domain:  "system",
			Version: osVersion.Msg.Version,
		})
	}

	return resp, nil
}
