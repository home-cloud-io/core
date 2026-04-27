package main

import (
	"context"
	"embed"
	"os"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/steady-bytes/draft/pkg/chassis"
	"github.com/steady-bytes/draft/pkg/loggers/zerolog"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/discovery"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	v1 "github.com/home-cloud-io/core/services/platform/operator/api/v1"
	"github.com/home-cloud-io/core/services/platform/operator/controller"
	"github.com/home-cloud-io/core/services/platform/operator/controller/talos"
	"github.com/home-cloud-io/core/services/platform/operator/logger"
	"github.com/home-cloud-io/core/services/platform/operator/server/apps"
	k8sclient "github.com/home-cloud-io/core/services/platform/operator/server/k8s-client"
	"github.com/home-cloud-io/core/services/platform/operator/server/system"
	"github.com/home-cloud-io/core/services/platform/operator/server/web"
)

var (
	//go:embed web-client/dist/*
	files  embed.FS
	scheme = runtime.NewScheme()
)

func init() {
	// initialize scheme
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	// add home-cloud.io crds
	utilruntime.Must(v1.AddToScheme(scheme))
	// add gateway api crds
	utilruntime.Must(gwv1.Install(scheme))
	// add talos crds
	utilruntime.Must(talos.AddToScheme(scheme))
}

func main() {
	// configure chassis
	c := chassis.New(zerolog.New())
	defer c.Start()

	var (
		kclient = k8sclient.NewClient(c.Logger())
		actl    = apps.NewController(kclient)
		sctl    = system.NewController(c.Logger(), kclient, actl)
	)
	c = c.WithClientApplication(files, "web-client/dist").
		WithRPCHandler(web.New(c.Logger(), actl, sctl))

	// clean copy of logger for controllers
	logr := logger.NewLogger(c.Logger().WithFields(nil))
	ctrl.SetLogger(logr)

	globalCtx, globalCancel := context.WithCancel(ctrl.SetupSignalHandler())

	// configure manager
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Logger:                        logr,
		Scheme:                        scheme,
		HealthProbeBindAddress:        "",
		LeaderElection:                true,
		LeaderElectionID:              "operator.home-cloud.io",
		LeaderElectionReleaseOnCancel: true,
		LeaderElectionNamespace:       "home-cloud-system",
	})
	if err != nil {
		c.Logger().WithError(err).Error("failed to create manager")
		os.Exit(1)
	}

	discoveryClient, err := discovery.NewDiscoveryClientForConfig(mgr.GetConfig())
	if err != nil {
		c.Logger().WithError(err).Error("failed to create discovery client")
		os.Exit(1)
	}

	// create app controller
	if err = (&controller.AppReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		c.Logger().WithField("controller", "app").WithError(err).Error("failed to create controller")
		os.Exit(1)
	}

	// create install controller
	if err = (&controller.InstallReconciler{
		Client:          mgr.GetClient(),
		DiscoveryClient: discoveryClient,
		Scheme:          mgr.GetScheme(),
		Config:          mgr.GetConfig(),
		Cancel:          globalCancel,
	}).SetupWithManager(mgr); err != nil {
		c.Logger().WithField("controller", "Install").WithError(err).Error("failed to create controller")
		os.Exit(1)
	}

	// add health/ready checks
	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		c.Logger().WithError(err).Error("failed to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		c.Logger().WithError(err).Error("failed to set up ready check")
		os.Exit(1)
	}

	// start manager
	c.Logger().Info("starting manager")
	go func() {
		err := mgr.Start(globalCtx)
		if err != nil {
			c.Logger().WithError(err).Error("problem running manager")
		}
		// send interupt to the chassis
		chassis.Closer() <- os.Interrupt
	}()
}
