package system

import (
	"context"
	"net/http"

	dv1connect "github.com/home-cloud-io/core/api/platform/daemon/v1/v1connect"
	opv1 "github.com/home-cloud-io/core/services/platform/operator/api/v1"
	"github.com/home-cloud-io/core/services/platform/server/apps"
	k8sclient "github.com/home-cloud-io/core/services/platform/server/k8s-client"
	"github.com/home-cloud-io/core/services/platform/server/utils/strings"

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

	daemonAddress := strings.Default(install.Spec.Daemon.Address, DefaultDaemonAddress)
	return &controller{
		actl:         actl,
		k8sclient:    kclient,
		daemonClient: dv1connect.NewDaemonServiceClient(http.DefaultClient, daemonAddress),
	}
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
