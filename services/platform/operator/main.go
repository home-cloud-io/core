package main

import (
	"embed"

	"github.com/steady-bytes/draft/pkg/chassis"
	"github.com/steady-bytes/draft/pkg/loggers/zerolog"

	"github.com/home-cloud-io/core/services/platform/operator/controller"
	"github.com/home-cloud-io/core/services/platform/operator/server/apps"
	k8sclient "github.com/home-cloud-io/core/services/platform/operator/server/k8s-client"
	"github.com/home-cloud-io/core/services/platform/operator/server/system"
	"github.com/home-cloud-io/core/services/platform/operator/server/web"
)

//go:embed web-client/dist/*
var files embed.FS

func main() {
	c := chassis.New(zerolog.New())
	defer c.Start()

	var (
		kclient = k8sclient.NewClient(c.Logger())
		actl    = apps.NewController(kclient)
		sctl    = system.NewController(c.Logger(), kclient, actl)
	)

	c = c.WithClientApplication(files, "web-client/dist").
		WithRPCHandler(web.New(c.Logger(), actl, sctl)).
		WithRunner(func() {
			controller.Start(c.Logger())
		})
}
