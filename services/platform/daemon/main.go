package main

import (
	"github.com/home-cloud-io/core/services/platform/daemon/communicate"

	"github.com/steady-bytes/draft/pkg/chassis"
	"github.com/steady-bytes/draft/pkg/loggers/zerolog"
)

func main() {
	var (
		logger = zerolog.New()
		server = communicate.New(logger)
	)

	// setup runtime
	chassis.New(logger).
		WithRPCHandler(server).
		Start()
}
