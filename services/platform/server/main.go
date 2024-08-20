package main

import (
	"embed"

	dv1 "github.com/home-cloud-io/core/api/platform/daemon/v1"
	"github.com/home-cloud-io/core/services/platform/server/daemon"
	"github.com/home-cloud-io/core/services/platform/server/web"

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
		messages  = make(chan *dv1.DaemonMessage)
		daemonRPC = daemon.New(logger, messages)
		webRPC    = web.New(logger, messages)
	)

	defer chassis.New(logger).
		WithClientApplication(files).
		WithRPCHandler(daemonRPC).
		WithRPCHandler(webRPC).
		WithRoute(&ntv1.Route{
			Match: &ntv1.RouteMatch{
				Prefix: "/",
			},
			EnableHttp2: true,
		}).
		Start()
}
