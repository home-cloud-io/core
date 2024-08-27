package main

import (
	"embed"

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
		daemonRPC = daemon.New(logger)
		webRPC    = web.New(logger)
	)

	// Create the app store cache
	// TODO: Put this in a goroutine and do it on a timer every 24 hours
	if err := web.NewStoreCache(logger); err != nil {
		logger.WithError(err).Error("failed to create app store cache")
	}

	if err := web.NewSecretSeed(logger); err != nil {
		logger.WithError(err).Error("failed to create secret seed")
	}

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
