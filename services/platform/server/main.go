package main

import (
	"embed"

	dv1 "github.com/home-cloud-io/core/api/platform/daemon/v1"
	"github.com/home-cloud-io/core/services/platform/server/apps"
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
		logger      = zerolog.New()
		messages    = make(chan *dv1.DaemonMessage)
		daemonRPC   = system.New(logger, messages)
		actl        = apps.NewController(logger)
		sctl        = system.NewController(logger, messages)
		webRPC      = web.New(logger, actl, sctl)
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
