package controller

import (
	"net/http"

	dv1connect "github.com/home-cloud-io/core/api/platform/daemon/v1/v1connect"
)

// DaemonClient creates a DaemonServiceClient and defaults the address if not set
func DaemonClient(address string) dv1connect.DaemonServiceClient {
	if address == "" {
		address = DefaultDaemonAddress
	}
	return dv1connect.NewDaemonServiceClient(http.DefaultClient, address)
}
