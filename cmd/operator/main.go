package main

import (
	"github.com/steady-bytes/draft/pkg/chassis"
	"github.com/steady-bytes/draft/pkg/loggers/zerolog"

	"github.com/home-cloud-io/core/cmd/operator/controller"
	"github.com/home-cloud-io/core/cmd/operator/server/apps"
	k8sclient "github.com/home-cloud-io/core/cmd/operator/server/k8s-client"
	"github.com/home-cloud-io/core/cmd/operator/server/system"
	"github.com/home-cloud-io/core/cmd/operator/server/web"
	"github.com/home-cloud-io/core/web/client"
)

func main() {
	c := chassis.New(zerolog.New())
	defer c.Start()

	var (
		kclient = k8sclient.NewClient(c.Logger())
		actl    = apps.NewController(kclient)
		sctl    = system.NewController(c.Logger(), kclient, actl)
	)

	c = c.WithClientApplication(client.Files, client.Root).
		WithRPCHandler(web.New(c.Logger(), actl, sctl)).
		WithRunner(func() {
			controller.Start(c.Logger())
		})
}
