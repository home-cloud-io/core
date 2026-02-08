package main

import (
	"context"
	"embed"

	"github.com/home-cloud-io/core/services/platform/server/apps"
	"github.com/home-cloud-io/core/services/platform/server/system"
	"github.com/home-cloud-io/core/services/platform/server/web"

	"github.com/steady-bytes/draft/pkg/chassis"
	"github.com/steady-bytes/draft/pkg/loggers/zerolog"
)

//go:embed web-client/dist/*
var files embed.FS

func main() {
	var (
		logger = zerolog.New()
		actl   = apps.NewController(logger)
		sctl   = system.NewController(logger, actl)
		webRPC = web.New(logger, actl, sctl)
	)

	runner := func() {
		go actl.AutoUpdate(context.Background(), logger)
	}

	defer chassis.New(logger).
		WithClientApplication(files, "web-client/dist").
		WithRPCHandler(webRPC).
		WithRunner(runner).
		Start()
}
