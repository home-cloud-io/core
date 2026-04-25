package main

import (
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
	"github.com/home-cloud-io/core/services/platform/operator/internal/controller"
	"github.com/home-cloud-io/core/services/platform/operator/internal/controller/talos"
	"github.com/home-cloud-io/core/services/platform/operator/logger"
)

var (
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
	// configure logger
	log := zerolog.New()
	_ = chassis.New(log).DisableMux()
	log.SetLevel(chassis.GetConfig().LogLevel())

	// clean copy of logger for controllers
	logr := logger.NewLogger(log.WithFields(nil))
	ctrl.SetLogger(logr)

	// configure manager
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Logger:                 logr,
		Scheme:                 scheme,
		HealthProbeBindAddress: ":" + chassis.GetConfig().GetString("service.network.bind_port"),
	})
	if err != nil {
		log.WithError(err).Error("failed to create manager")
		os.Exit(1)
	}

	discoveryClient, err := discovery.NewDiscoveryClientForConfig(mgr.GetConfig())
	if err != nil {
		log.WithError(err).Error("failed to create discovery client")
		os.Exit(1)
	}

	// create app controller
	if err = (&controller.AppReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		log.WithField("controller", "app").WithError(err).Error("failed to create controller")
		os.Exit(1)
	}

	// create install controller
	if err = (&controller.InstallReconciler{
		Client:          mgr.GetClient(),
		DiscoveryClient: discoveryClient,
		Scheme:          mgr.GetScheme(),
		Config:          mgr.GetConfig(),
	}).SetupWithManager(mgr); err != nil {
		log.WithField("controller", "Install").WithError(err).Error("failed to create controller")
		os.Exit(1)
	}

	// add health/ready checks
	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		log.WithError(err).Error("failed to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		log.WithError(err).Error("failed to set up ready check")
		os.Exit(1)
	}

	// start manager
	log.Warn("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		log.WithError(err).Error("problem running manager")
		os.Exit(1)
	}
}
