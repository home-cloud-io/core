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
		stun     = host.NewSTUNClient(logger)
		locator  = host.NewLocatorController(logger, stun)
		client   = communicate.NewClient(logger, mdns, stun, locator)
		migrator = host.NewMigrator(logger)
	)

	// setup runtime
	runtime := chassis.New(logger).
		WithRunner(client.Listen).
		WithRunner(mdns.Start).
		WithRunner(migrator.Migrate).
		WithRunner(locator.Load)

	// start daemon runtime
	runtime.Start()
}
