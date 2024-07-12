package main

import (
	"embed"

	"github.com/home-cloud-io/core/services/platform/server/daemon"
	"github.com/home-cloud-io/core/services/platform/server/mdns"
	"github.com/steady-bytes/draft/pkg/chassis"
	"github.com/steady-bytes/draft/pkg/loggers/zerolog"
)

//go:embed web-client/dist/index.html
//go:embed web-client/dist/main.js
var files embed.FS

func main() {
	var (
		logger    = zerolog.New()
		daemonRPC = daemon.New()
	)

	defer chassis.New(logger).
		WithClientApplication(files).
		WithRPCHandler(daemonRPC).
		Start()

	go mdns.ServeMDNS(logger)
}
