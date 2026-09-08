package mdns

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	v1 "github.com/home-cloud-io/core/api/crds/v1"
)

type Reconciler struct {
	client.Client
	Scheme *runtime.Scheme

	Server   Server
	services map[types.NamespacedName][]string
}

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := log.FromContext(ctx)

	// Get the object that triggered reconciliation
	obj := &corev1.Service{}
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

func (r *Reconciler) reconcile(ctx context.Context, req ctrl.Request, obj *corev1.Service) error {
	l := log.FromContext(ctx)
	l.Info("reconciling service", "nn", req.NamespacedName)

	// remove if marked for deletion
	if obj.GetDeletionTimestamp() != nil {
		return r.remove(ctx, obj, req.NamespacedName)
	}

	// remove if not annotated
	value, exists := obj.Annotations[v1.AnnotationDNSHostnames]
	if !exists {
		return r.remove(ctx, obj, req.NamespacedName)
	}

	values := strings.Split(value, ",")
	if len(values) == 0 {
		l.Error(fmt.Errorf("no hostname provided in annotation"), "invalid annotation value")
		// return nil as there's no sense in retrying this error
		return nil
	}

	// track
	r.services[req.NamespacedName] = values

	// add to mdns server
	for _, value := range values {
		err := r.Server.AddHost(ctx, value)
		if err != nil {
			return err
		}
	}

	return nil
}

// remove attempts to remove a host but simply skips if it's not being tracked
func (r *Reconciler) remove(ctx context.Context, obj *corev1.Service, nn types.NamespacedName) error {
	values, tracking := r.services[nn]
	if tracking {
		for _, value := range values {
			err := r.Server.RemoveHost(ctx, value)
			if err != nil {
				return err
			}
		}

		// remove from local map when fully removed
		delete(r.services, nn)
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.services = map[types.NamespacedName][]string{}
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Service{}).
		Complete(r)
}
