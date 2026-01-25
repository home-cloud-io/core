package main

import (
	"embed"

	"github.com/home-cloud-io/core/services/platform/locator/server"

	"github.com/steady-bytes/draft/pkg/chassis"
	"github.com/steady-bytes/draft/pkg/loggers/zerolog"
)

//go:embed web-client/index.html
var files embed.FS
func main() {
	var (
		logger = zerolog.New()
		rpc    = server.New(logger)
	)

	defer chassis.New(logger).
		WithRPCHandler(rpc).
		WithClientApplication(files, "web-client").
		WithRunner(server.StartStun).
		Start()
}
