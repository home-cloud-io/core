package system

import (
	"context"
	"net/http"

	dv1connect "github.com/home-cloud-io/core/api/platform/daemon/v1/v1connect"
	opv1 "github.com/home-cloud-io/core/services/platform/operator/api/v1"
	"github.com/home-cloud-io/core/services/platform/server/apps"
	k8sclient "github.com/home-cloud-io/core/services/platform/server/k8s-client"

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

func NewController(logger chassis.Logger, actl apps.Controller) Controller {
	ctx := context.Background()
	k := k8sclient.NewClient(logger)

	install := &opv1.Install{}
	err := k.Get(ctx, types.NamespacedName{
		Namespace: k8sclient.HomeCloudNamespace,
		Name:      "install",
	}, install)
	if err != nil {
		logger.WithError(err).Panic("failed to get install")
	}

	daemonAddress := install.Spec.Daemon.Address
	if daemonAddress == "" {
		daemonAddress = DefaultDaemonAddress
	}

	return &controller{
		actl:         actl,
		k8sclient:    k,
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
