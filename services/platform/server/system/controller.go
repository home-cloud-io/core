package system

import (
	"net/http"

	dv1connect "github.com/home-cloud-io/core/api/platform/daemon/v1/v1connect"
	k8sclient "github.com/home-cloud-io/core/services/platform/server/k8s-client"

	"github.com/steady-bytes/draft/pkg/chassis"
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
		k8sclient    k8sclient.System
		daemonClient dv1connect.DaemonServiceClient
	}
)

func NewController(logger chassis.Logger) Controller {
	return &controller{
		k8sclient: k8sclient.NewClient(logger),
		// TODO: make this address configurable
		daemonClient: dv1connect.NewDaemonServiceClient(http.DefaultClient, "http://daemon.home-cloud-system"),
	}
}

const (
	ErrFailedToBuildSeedGetRequest = "failed to build get request for seed"

	// Currently only a single interface is supported and defaults to this value. In the future we
	// will probably want to support multiple interfaces (e.g. one for trusted mobile clients and another for federated servers)
	DefaultWireguardInterface = "wg0"
	DefaultSTUNServerAddress  = "locator1.home-cloud.io:3478"
	DefaultLocatorAddress     = "https://locator1.home-cloud.io"
)
