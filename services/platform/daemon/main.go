package main

import (
	"github.com/home-cloud-io/core/services/platform/daemon/communicate"
	"github.com/home-cloud-io/core/services/platform/daemon/host"

	"github.com/steady-bytes/draft/pkg/chassis"
	"github.com/steady-bytes/draft/pkg/loggers/zerolog"
)

func main() {
	var (
		logger   = zerolog.New()
		mdns     = host.NewDNSPublisher(logger)
		remoteAccess = host.NewRemoteAccessController(logger)
		client   = communicate.NewClient(logger, mdns, remoteAccess)
		migrator = host.NewMigrator(logger)
	)

	// setup runtime
	runtime := chassis.New(logger).
		WithRunner(client.Listen).
		WithRunner(mdns.Start).
		WithRunner(migrator.Migrate).
		WithRunner(remoteAccess.Load)

	// start daemon runtime
	runtime.Start()
}
