package controller

import (
	"net/http"

	dv1connect "github.com/home-cloud-io/core/api/platform/daemon/v1/v1connect"
	"github.com/steady-bytes/draft/pkg/chassis"
)

// DaemonClient creates a DaemonServiceClient and defaults the address if not set
func DaemonClient(address string) dv1connect.DaemonServiceClient {
	if address == "" {
		address = DefaultDaemonAddress
		if chassis.GetConfig().Env() == "local" {
			address = "http://localhost:9000"
		}
	}
	return dv1connect.NewDaemonServiceClient(http.DefaultClient, address)
}
