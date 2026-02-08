package main

import (
	"github.com/home-cloud-io/core/services/platform/daemon/server"

	"github.com/steady-bytes/draft/pkg/chassis"
	"github.com/steady-bytes/draft/pkg/loggers/zerolog"
)

func main() {
	var (
		logger = zerolog.New()
		s      = server.New(logger)
	)

	// setup runtime
	chassis.New(logger).
		WithRPCHandler(s).
		Start()
}
