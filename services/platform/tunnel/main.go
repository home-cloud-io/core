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
	"sigs.k8s.io/controller-runtime/pkg/healthz"

	v1 "github.com/home-cloud-io/core/services/platform/operator/api/v1"
	"github.com/home-cloud-io/core/services/platform/operator/logger"
	"github.com/home-cloud-io/services/platform/tunnel/wireguard"
)

var (
	scheme = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	// initialize scheme
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	// add home-cloud.io crds
	utilruntime.Must(v1.AddToScheme(scheme))
}

func main() {
	// configure logger
	log := zerolog.New()
	_ = chassis.New(log).DisableMux()
	log.SetLevel(chassis.GetConfig().LogLevel())
	ctrl.SetLogger(logger.NewLogger(log))

	// configure manager
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                        scheme,
		HealthProbeBindAddress:        ":" + chassis.GetConfig().GetString("service.network.bind_port"),
		LeaderElection:                true,
		LeaderElectionNamespace:       "home-cloud-system",
		LeaderElectionID:              "tunnel.home-cloud.io",
		LeaderElectionReleaseOnCancel: true,
	})
	if err != nil {
		setupLog.Error(err, "failed to create manager")
		os.Exit(1)
	}

	// create wireguard controller
	if err = (&wireguard.WireguardReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "failed to create controller", "controller", "Wireguard")
		os.Exit(1)
	}

	// add health/ready checks
	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "failed to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "failed to set up ready check")
		os.Exit(1)
	}

	// start manager
	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
