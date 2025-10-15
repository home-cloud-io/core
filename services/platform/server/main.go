package main

import (
	"embed"

	"github.com/home-cloud-io/core/services/platform/server/apps"
	"github.com/home-cloud-io/core/services/platform/server/async"
	kvclient "github.com/home-cloud-io/core/services/platform/server/kv-client"
	"github.com/home-cloud-io/core/services/platform/server/system"
	"github.com/home-cloud-io/core/services/platform/server/web"

	"github.com/steady-bytes/draft/pkg/chassis"
	"github.com/steady-bytes/draft/pkg/loggers/zerolog"
)

//go:embed web-client/dist/*
var files embed.FS

func main() {
	var (
		broadcaster = async.NewBroadcaster()
		logger      = zerolog.New()
		actl        = apps.NewController(logger)
		sctl        = system.NewController(logger, broadcaster)
		webRPC      = web.New(logger, actl, sctl)
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
		WithRPCHandler(webRPC).
		WithRunner(runner).
		Start()
}
