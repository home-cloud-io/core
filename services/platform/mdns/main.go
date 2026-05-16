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
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"

	"github.com/home-cloud-io/core/services/platform/mdns/mdns"
	"github.com/home-cloud-io/core/services/platform/operator/logger"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	// initialize scheme
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
}

func main() {
	// configure logger
	log := zerolog.New()
	_ = chassis.New(log).DisableMux()
	log.SetLevel(chassis.GetConfig().LogLevel())
	ctrl.SetLogger(logger.NewLogger(log))

	// configure manager
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		Metrics: server.Options{
			BindAddress: "0",
		},
		HealthProbeBindAddress: "",
		// no need for election since mdns needs to run as a single replica StatefulSet
		LeaderElection: false,
	})
	if err != nil {
		setupLog.Error(err, "failed to create manager")
		os.Exit(1)
	}

	// create service controller
	if err = (&mdns.Reconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
		Server: mdns.New(log),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "failed to create controller", "controller", "mDNS")
		os.Exit(1)
	}

	// start manager
	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
