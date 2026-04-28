package controller

import (
	"context"
	"os"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/steady-bytes/draft/pkg/chassis"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/discovery"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	v1 "github.com/home-cloud-io/core/services/platform/operator/api/v1"
	"github.com/home-cloud-io/core/services/platform/operator/controller/talos"
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

func Start(l chassis.Logger) {
	defer Stop()

	// clean copy of logger for controllers
	logr := logger.NewLogger(l.WithFields(nil))
	ctrl.SetLogger(logr)

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
		l.WithError(err).Error("failed to create manager")
		return
	}

	discoveryClient, err := discovery.NewDiscoveryClientForConfig(mgr.GetConfig())
	if err != nil {
		l.WithError(err).Error("failed to create discovery client")
		return
	}

	// create app controller
	if err = (&AppReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		l.WithField("controller", "app").WithError(err).Error("failed to create controller")
		return
	}

	// global context to allow the install reconciler to stop the manager when a shutdown
	// is needed on operator upgrade
	ctx, cancel := context.WithCancel(ctrl.SetupSignalHandler())

	// create install controller
	if err = (&InstallReconciler{
		Client:          mgr.GetClient(),
		DiscoveryClient: discoveryClient,
		Scheme:          mgr.GetScheme(),
		Config:          mgr.GetConfig(),
		Cancel:          cancel,
	}).SetupWithManager(mgr); err != nil {
		l.WithField("controller", "Install").WithError(err).Error("failed to create controller")
		return
	}

	// start manager
	l.Info("starting manager")
	err = mgr.Start(ctx)
	if err != nil {
		l.WithError(err).Error("problem running manager")
		return
	}
}

// send interupt to the chassis
func Stop() {
	chassis.Closer() <- os.Interrupt
}
