package main

import (
	"github.com/home-cloud-io/core/services/platform/locator/server"

	ntv1 "github.com/steady-bytes/draft/api/core/control_plane/networking/v1"
	"github.com/steady-bytes/draft/pkg/chassis"
	"github.com/steady-bytes/draft/pkg/loggers/zerolog"
)

func main() {
	var (
		logger = zerolog.New()
		rpc    = server.New(logger)
	)

	defer chassis.New(logger).
		WithRPCHandler(rpc).
		WithRoute(&ntv1.Route{
			Match: &ntv1.RouteMatch{
				Prefix: "/",
			},
			EnableHttp2: true,
		}).
		WithRunner(server.StartStun).
		Start()
}
