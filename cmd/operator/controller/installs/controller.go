package installs

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"slices"

	"dario.cat/mergo"
	"gopkg.in/yaml.v3"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	v1 "github.com/home-cloud-io/core/api/crds/v1"
	"github.com/home-cloud-io/core/pkg/install/resources"
	"github.com/home-cloud-io/core/pkg/kube"
)

// InstallReconciler reconciles a Install object
type InstallReconciler struct {
	client.Client
	DiscoveryClient *discovery.DiscoveryClient
	Scheme          *runtime.Scheme
	Config          *rest.Config
	// global cancel function to shutdown the manager (useful for operator upgrades)
	Cancel context.CancelFunc
}

type reconcilerFunc func(ctx context.Context, r *InstallReconciler, install *v1.Install) error

const (
	InstallFinalizer     = "install.home-cloud.io/finalizer"
	ReleasesURL          = "https://github.com/home-cloud-io/core/releases/download/"
	DefaultDaemonAddress = "http://daemon.home-cloud-system"
)

var (
	reconcilers = []reconcilerFunc{
		// reconcile crds before operator so the operator is always using the latest versions
		reconcileCRDs,
		// reconcile operator before the other components since it may be necessary to patch a bug in itself to
		// prevent getting locked up on other components
		reconcileOperator,
		reconcileGatewayAPI,
		reconcileNamespaces,
		reconcileCertManager,
		reconcileIstio,
		reconcileBlocky,
		reconcileMDNS,
		reconcileTunnel,
		reconcileDaemon,
		reconcileGenericDevicePlugin,
		// TODO: shouldn't we actually reconcile system/kube first so that we can upgrade everything in-place?
		// 		 for example: if we upgraded istio to something that only supports kube 1.38 we'd want to upgrade to kube 1.38
		// 		 before istio?
		// 		 Need to check talos docs to confirm precedence here.
		reconcileSystem,
		reconcileKubernetes,
	}
)

func (r *InstallReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := log.FromContext(ctx)
	l.Info("Reconciling Install")

	// Get the CRD that triggered reconciliation
	install := &v1.Install{}
	err := r.Get(ctx, req.NamespacedName, install)
	if err != nil {
		if kerrors.IsNotFound(err) {
			l.Info("Install resource not found. Assuming this means the resource was deleted and so ignoring.")
			return ctrl.Result{}, nil
		}
		l.Info("Failed to get Install resource. Re-running reconcile.")
		return ctrl.Result{}, err
	}

	// get version manifest from repo
	resp, err := http.Get(fmt.Sprintf("%s/%s/manifest.yaml", ReleasesURL, install.Spec.Version))
	if err != nil {
		return ctrl.Result{}, err
	}

	// populate versions into default install spec
	dec := yaml.NewDecoder(resp.Body)
	err = dec.Decode(&resources.DefaultInstall.Spec)
	if err != nil {
		return ctrl.Result{}, err
	}

	// set defaults: any values set on the resource will override the defaults, including versions
	err = mergo.Merge(install, resources.DefaultInstall)
	if err != nil {
		return ctrl.Result{}, err
	}

	// if marked for deletion, try to delete/uninstall
	if install.GetDeletionTimestamp() != nil {
		l.Info("Uninstalling Install")
		return ctrl.Result{}, r.tryDeletions(ctx, install)
	}

	// update status as reconcile ends so that we always have the latest status before next
	// reconcile iteration. this way we don't try and install components that are already installed
	oldStatus := install.Status.DeepCopy()
	defer func() {
		// guard against infinite reconcile loop with updating same status
		if reflect.DeepEqual(install.Status, *oldStatus) {
			return
		}

		// TODO: this panics sometimes
		err := r.Status().Update(ctx, install)
		if err != nil {
			panic(err)
		}
	}()

	return ctrl.Result{}, r.reconcile(ctx, install)
}

func (r *InstallReconciler) reconcile(ctx context.Context, install *v1.Install) error {
	l := log.FromContext(ctx)

	// run all reconcile functions (in order)
	for _, f := range reconcilers {
		err := f(ctx, r, install)
		if err != nil {
			return err
		}
	}

	l.Info("reconcile complete")
	return nil
}

func (r *InstallReconciler) tryDeletions(ctx context.Context, install *v1.Install) error {
	if controllerutil.ContainsFinalizer(install, InstallFinalizer) {
		err := r.uninstall(ctx, install)
		if err != nil {
			return err
		}

		controllerutil.RemoveFinalizer(install, InstallFinalizer)
		err = r.Update(ctx, install)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *InstallReconciler) uninstall(ctx context.Context, install *v1.Install) error {

	// NOTE: we do not delete any CRDs as they could be in use by other applications
	// 			 we also do not uninstall operator resources for obvious reasons

	err := kube.UninstallResources(ctx, r.Client, slices.Concat(
		resources.GatewayObjects(install),
		resources.MDNSObjects(install),
		resources.TunnelObjects(install),
		resources.DaemonObjects(install),
		resources.TalosObjects(install),
	))
	if err != nil {
		return err
	}

	err = uninstallIstio(ctx, install)
	if err != nil {
		return err
	}

	// delete namespaces last
	err = uninstallNamespaces(ctx, r, install)
	if err != nil {
		return err
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *InstallReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.Install{}).
		Complete(r)
}
