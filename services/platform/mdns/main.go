package main

import (
	"context"
	"fmt"
	"os"

	k8sclient "github.com/home-cloud-io/services/platform/mdns/k8s-client"
	"github.com/home-cloud-io/services/platform/mdns/mdns"
	"github.com/home-cloud-io/services/platform/mdns/services"
	"github.com/steady-bytes/draft/pkg/chassis"
	"github.com/steady-bytes/draft/pkg/loggers/zerolog"

	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/informers"
)

const (
	namespace = "home-cloud"
)

type (
	Runner struct {
		logger chassis.Logger
	}
)

func main() {
	var (
		logger = zerolog.New()
		runner = &Runner{logger: logger}
	)

	defer chassis.New(logger).
		WithRunner(runner.run).
		Start()
}

func (r *Runner) run() {
	k8sClient := k8sclient.NewClient(r.logger)
	ctx := context.Background()

	// channels
	notifyMdns := make(chan services.Resource)
	stopper := make(chan struct{})

	// closers
	defer close(stopper)
	defer runtime.HandleCrash()

	// run listener in the background
	factory := informers.NewSharedInformerFactory(k8sClient, 0)
	serviceController, err := services.NewServicesWatcher(factory, notifyMdns)
	if err != nil {
		r.logger.WithError(err).Panic("failed to initialize services watcher")
	}
	go serviceController.Run(r.logger, stopper)

	// initialize server and add host
	mdnsServer := mdns.New(r.logger)
	// TODO: get config from blueprint
	err = mdnsServer.AddHost(ctx, fmt.Sprintf("%s.local", os.Getenv("HOST_NAME")))
	if err != nil {
		panic(err)
	}

	// listen for resource events
	for {
		select {
		case resource := <-notifyMdns:
			switch resource.Action {
			case services.Added:
				err := mdnsServer.AddHost(ctx, resource.Hostname)
				if err != nil {
					panic(err)
				}
			case services.Deleted:
				err := mdnsServer.RemoveHost(ctx, resource.Hostname)
				if err != nil {
					panic(err)
				}
			}
		case <-stopper:
			r.logger.Info("stopping program")
		}
	}

}
