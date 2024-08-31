package main

import (
	"embed"

	dv1 "github.com/home-cloud-io/core/api/platform/daemon/v1"
	"github.com/home-cloud-io/core/services/platform/server/daemon"
	"github.com/home-cloud-io/core/services/platform/server/system"
	"github.com/home-cloud-io/core/services/platform/server/versioning"
	"github.com/home-cloud-io/core/services/platform/server/web"

	ntv1 "github.com/steady-bytes/draft/api/core/control_plane/networking/v1"
	"github.com/steady-bytes/draft/pkg/chassis"
	"github.com/steady-bytes/draft/pkg/loggers/zerolog"
)

//go:embed web-client/dist/*
var files embed.FS

func main() {
	var (
		logger     = zerolog.New()
		messages   = make(chan *dv1.DaemonMessage)
		daemonRPC  = daemon.New(logger, messages)
		controller = system.NewController(logger, messages)
		updater    = versioning.NewUpdater(logger, messages, controller)
		webRPC     = web.New(logger, controller, updater)
		storeCache = web.NewStoreCache(logger)
	)

	if err := web.NewSecretSeed(logger); err != nil {
		logger.Error("failed to create secret seed")
	}

	defer chassis.New(logger).
		WithClientApplication(files).
		WithRPCHandler(daemonRPC).
		WithRPCHandler(webRPC).
		WithRunner(storeCache.Refresh).
		WithRunner(updater.AutoUpdateSystem).
		WithRunner(updater.AutoUpdateSystemContainers).
		WithRunner(updater.AutoUpdateApps).
		WithRoute(&ntv1.Route{
			Match: &ntv1.RouteMatch{
				Prefix: "/",
			},
			EnableHttp2: true,
		}).
		Start()
}
