package main

import (
	"embed"

	"github.com/home-cloud-io/core/services/platform/server/daemon"
	"github.com/home-cloud-io/core/services/platform/server/mdns"

	ntv1 "github.com/steady-bytes/draft/api/core/control_plane/networking/v1"
	"github.com/steady-bytes/draft/pkg/chassis"
	"github.com/steady-bytes/draft/pkg/loggers/zerolog"
)

//go:embed web-client/dist/index.html
//go:embed web-client/dist/main.js
var files embed.FS

func main() {
	var (
		logger    = zerolog.New()
		daemonRPC = daemon.New(logger)
	)

	defer chassis.New(logger).
		WithClientApplication(files).
		WithRPCHandler(daemonRPC).
		WithRunner(daemonRPC.Run).
		WithRoute(&ntv1.Route{
			Match: &ntv1.RouteMatch{
				Prefix: "/",
			},
		}).
		Start()

	go mdns.ServeMDNS(logger)
}
