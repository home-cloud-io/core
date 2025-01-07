package main

import (
	"github.com/home-cloud-io/core/services/platform/locator/server"

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
		WithRunner(server.StartStun).
		Start()
}
