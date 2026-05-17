package mdns

import (
	"context"

	v1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	DNSAnnotation = "home-cloud.io/dns"
)

type Reconciler struct {
	client.Client
	Scheme *runtime.Scheme

	Server   Server
	services map[types.NamespacedName]string
}

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := log.FromContext(ctx)

	// Get the object that triggered reconciliation
	obj := &v1.Service{}
	err := r.Get(ctx, req.NamespacedName, obj)
	if err != nil {
		if kerrors.IsNotFound(err) {
			l.Info("Service resource not found. Assuming this means the resource was deleted and so ignoring.")
			return ctrl.Result{}, nil
		}
		l.Info("Failed to get Service resource. Re-running reconcile.")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, r.reconcile(ctx, req, obj)
}

func (r *Reconciler) reconcile(ctx context.Context, req ctrl.Request, obj *v1.Service) error {
	l := log.FromContext(ctx)
	l.Info("reconciling service", "nn", req.NamespacedName)

	// remove if marked for deletion
	if obj.GetDeletionTimestamp() != nil {
		return r.remove(ctx, obj, req.NamespacedName)
	}

	// remove if not annotated
	hostname, exists := obj.Annotations[DNSAnnotation]
	if !exists {
		return r.remove(ctx, obj, req.NamespacedName)
	}

	// track
	r.services[req.NamespacedName] = hostname

	// add to mdns server
	return r.Server.AddHost(ctx, hostname)
}

// remove attempts to remove a host but simply skips if it's not being tracked
func (r *Reconciler) remove(ctx context.Context, obj *v1.Service, nn types.NamespacedName) error {
	current, tracking := r.services[nn]
	if tracking {
		delete(r.services, nn)
		return r.Server.RemoveHost(ctx, current)
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.services = map[types.NamespacedName]string{}
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.Service{}).
		Complete(r)
}
