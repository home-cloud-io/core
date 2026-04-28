package system

import (
	"context"
	"net/http"

	dv1connect "github.com/home-cloud-io/core/api/platform/daemon/v1/v1connect"
	opv1 "github.com/home-cloud-io/core/services/platform/operator/api/v1"
	"github.com/home-cloud-io/core/services/platform/operator/server/apps"
	k8sclient "github.com/home-cloud-io/core/services/platform/operator/server/k8s-client"
	"github.com/home-cloud-io/core/services/platform/operator/server/utils/strings"
	hstrings "github.com/home-cloud-io/core/services/platform/operator/server/utils/strings"
	"github.com/robfig/cron/v3"

	"github.com/steady-bytes/draft/pkg/chassis"
	"k8s.io/apimachinery/pkg/types"
)

type (
	Controller interface {
		Containers
		Daemon
		Device
		Locators
		OS
		Peering
	}

	controller struct {
		actl         apps.Controller
		k8sclient    k8sclient.System
		daemonClient dv1connect.DaemonServiceClient
		cronID       cron.EntryID
		cr           *cron.Cron
	}
)

func NewController(logger chassis.Logger, kclient k8sclient.System, actl apps.Controller) Controller {
	ctx := context.Background()

	install := &opv1.Install{}
	err := kclient.Get(ctx, types.NamespacedName{
		Namespace: k8sclient.DefaultHomeCloudNamespace,
		Name:      "install",
	}, install)
	if err != nil {
		logger.WithError(err).Panic("failed to get install")
	}

	defaultAddress := DefaultDaemonAddress
	if chassis.GetConfig().Env() == "local" {
		defaultAddress = "http://localhost:9000"
	}

	daemonAddress := defaultAddress
	if install.Spec.Daemon != nil {
		daemonAddress = strings.Default(install.Spec.Daemon.Address, defaultAddress)
	}
	c := &controller{
		actl:         actl,
		k8sclient:    kclient,
		daemonClient: dv1connect.NewDaemonServiceClient(http.DefaultClient, daemonAddress),
	}

	if install.Spec.Settings != nil {
		// run app auto update if configured
		if install.Spec.Settings.AutoUpdateApps {
			go c.actl.AutoUpdate(ctx, logger, hstrings.Default(install.Spec.Settings.AutoUpdateAppsSchedule, apps.DefaultAutoUpdateAppsSchedule))
		}

		// run system auto update if configured
		if install.Spec.Settings.AutoUpdateSystem {
			go c.AutoUpdate(ctx, logger, hstrings.Default(install.Spec.Settings.AutoUpdateSystemSchedule, DefaultAutoUpdateSystemSchedule))
		}
	}

	return c
}

const (
	ErrFailedToBuildSeedGetRequest = "failed to build get request for seed"

	// Currently only a single interface is supported and defaults to this value. In the future we
	// will probably want to support multiple interfaces (e.g. one for trusted mobile clients and another for federated servers)
	DefaultWireguardInterface = "wg0"
	DefaultSTUNServerAddress  = "locator1.home-cloud.io:3478"
	DefaultLocatorAddress     = "https://locator1.home-cloud.io"
	DefaultDaemonAddress      = "http://daemon.home-cloud-system"
)
