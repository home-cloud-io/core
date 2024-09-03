package main

import (
	"github.com/home-cloud-io/core/services/platform/daemon/communicate"
	"github.com/steady-bytes/draft/pkg/chassis"
	"github.com/steady-bytes/draft/pkg/loggers/zerolog"
)

func main() {
	var (
		logger = zerolog.New()
		client = communicate.NewClient(logger)
	)

	defer chassis.New(logger).
		WithRunner(client.Listen).
		Start()
}
