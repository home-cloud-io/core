package main

import (
	"context"
	"fmt"

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
	serviceController, err := services.NewServicesWatcher(r.logger, factory, namespace, notifyMdns)
	if err != nil {
		r.logger.WithError(err).Panic("failed to initialize services watcher")
	}
	go serviceController.Run(stopper)


	// initialize server
	mdnsServer := mdns.New(r.logger)
	for {
		select {
		case advertiseResource := <-notifyMdns:
			hostname := fmt.Sprintf("%s-home-cloud.local", advertiseResource.Name)
			r.logger.Infof("advertising: %s", hostname)
			switch advertiseResource.Action {
			case services.Added:
				err := mdnsServer.AddHost(ctx, hostname)
				if err != nil {
					panic(err)
				}
			case services.Deleted:
				err := mdnsServer.RemoveHost(ctx, hostname)
				if err != nil {
					panic(err)
				}
			}
		case <-stopper:
			r.logger.Info("stopping program")
		}
	}

}