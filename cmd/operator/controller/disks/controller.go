package disks

import (
	"context"
	"time"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	v1 "github.com/home-cloud-io/core/api/crds/v1"
)

// TODO: need to create these resources on boot somehow

// DiskReconciler reconciles a Disk object
type DiskReconciler struct {
	client.Client
	DiscoveryClient *discovery.DiscoveryClient
	Scheme          *runtime.Scheme
	Config          *rest.Config
}

func (r *DiskReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := log.FromContext(ctx)
	l.Info("Reconciling Disk")

	// Get the CRD that triggered reconciliation
	disk := &v1.Disk{}
	err := r.Get(ctx, req.NamespacedName, disk)
	if err != nil {
		if kerrors.IsNotFound(err) {
			l.Info("Disk resource not found. Assuming this means the resource was deleted and so ignoring.")
			return ctrl.Result{}, nil
		}
		l.Info("Failed to get Disk resource. Re-running reconcile.")
		return ctrl.Result{}, err
	}

	return ctrl.Result{
		// poll the daemon for disk changes
		RequeueAfter: time.Minute,
	}, r.reconcile(ctx, disk)
}

func (r *DiskReconciler) reconcile(ctx context.Context, disk *v1.Disk) error {
	l := log.FromContext(ctx)

	// get disks from daemon
	// search for our disk (must be hydrated with the Talos UserVolume name since this is used in the PV)
	// - set status condition if not hydrated/loaded?
	// remove if not found -> return
	// update if found

	l.Info("reconcile complete")
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DiskReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.Disk{}).
		Complete(r)
}
