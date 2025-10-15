package main

import (
	"github.com/home-cloud-io/core/services/platform/daemon/communicate"
	"github.com/home-cloud-io/core/services/platform/daemon/host"
	kvclient "github.com/home-cloud-io/core/services/platform/daemon/kv-client"

	"github.com/steady-bytes/draft/pkg/chassis"
	"github.com/steady-bytes/draft/pkg/loggers/zerolog"
)

func main() {
	var (
		logger          = zerolog.New()
		secureTunneling = host.NewSecureTunnelingController(logger)
		server          = communicate.New(logger, secureTunneling)
		migrator        = host.NewMigrator(logger)
	)

	runner := func() {
		kvclient.Init()
	}

	// setup runtime
	runtime := chassis.New(logger).
		WithRPCHandler(server).
		// TODO: this doesn't gaurantee the kvclient will be ready for the other runners
		WithRunner(runner).
		WithRunner(migrator.Migrate).
		WithRunner(secureTunneling.Load)

	// start daemon runtime
	runtime.Start()
}
