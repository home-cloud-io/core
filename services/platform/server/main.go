package main

import (
	"embed"

	"github.com/home-cloud-io/core/services/platform/server/apps"
	"github.com/home-cloud-io/core/services/platform/server/async"
	"github.com/home-cloud-io/core/services/platform/server/internal"
	kvclient "github.com/home-cloud-io/core/services/platform/server/kv-client"
	"github.com/home-cloud-io/core/services/platform/server/system"
	"github.com/home-cloud-io/core/services/platform/server/web"

	ntv1 "github.com/steady-bytes/draft/api/core/control_plane/networking/v1"
	"github.com/steady-bytes/draft/pkg/chassis"
	"github.com/steady-bytes/draft/pkg/loggers/zerolog"
)

//go:embed web-client/dist/*
var files embed.FS

func main() {
	var (
		broadcaster = async.NewBroadcaster()
		logger      = zerolog.New()
		daemonRPC   = system.New(logger, broadcaster)
		actl        = apps.NewController(logger)
		sctl        = system.NewController(logger, broadcaster)
		webRPC      = web.New(logger, actl, sctl)
		// webHTTP     = web.NewHttp(logger, actl, sctl)
		internalRPC = internal.New(logger, sctl)
	)

	runner := func() {
		kvclient.Init()
		system.InitSecretSeed(logger)
		go apps.AppStoreCache(logger)
		go actl.AutoUpdate(logger)
		go sctl.AutoUpdateOS(logger)
		go sctl.AutoUpdateContainers(logger)
	}

	defer chassis.New(logger).
		WithClientApplication(files).
		WithRPCHandler(daemonRPC).
		WithRPCHandler(webRPC).
		// WithRPCHandler(webHTTP).
		WithRPCHandler(internalRPC).
		WithRunner(runner).
		WithRoute(&ntv1.Route{
			Match: &ntv1.RouteMatch{
				Prefix: "/",
			},
			EnableHttp2: true,
		}).
		Start()
}
