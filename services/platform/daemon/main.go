package main

import (
	"github.com/home-cloud-io/core/services/platform/daemon/communicate"
	"github.com/home-cloud-io/core/services/platform/daemon/host"
	"github.com/steady-bytes/draft/pkg/chassis"
	"github.com/steady-bytes/draft/pkg/loggers/zerolog"
)

func main() {
	var (
		logger = zerolog.New()
		client = communicate.NewClient(logger)
		server = host.NewServer(logger)
		runner = host.NewRunner(logger)
	)

	defer chassis.New(logger).
		WithRPCHandler(server).
		WithRunner(client.Listen).
		WithRunner(runner.OnStart).
		Start()
}
