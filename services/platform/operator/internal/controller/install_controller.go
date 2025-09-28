package controller

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"time"

	"dario.cat/mergo"
	"gopkg.in/yaml.v3"
	"helm.sh/helm/v3/pkg/action"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	v1 "github.com/home-cloud-io/core/services/platform/operator/api/v1"
	resources "github.com/home-cloud-io/core/services/platform/operator/internal/controller/resources"
)

// InstallReconciler reconciles a Install object
type InstallReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Config *rest.Config
}

const (
	InstallFinalizer = "install.home-cloud.io/finalizer"
)

//+kubebuilder:rbac:groups=home-cloud.io,resources=installs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=home-cloud.io,resources=installs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=home-cloud.io,resources=installs/finalizers,verbs=update

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

	// set defaults
	err = mergo.Merge(install, resources.DefaultInstall)
	if err != nil {
		return ctrl.Result{}, err
	}

	// if marked for deletion, try to delete/uninstall
	if install.GetDeletionTimestamp() != nil {
		l.Info("Uninstalling Install")
		return ctrl.Result{}, r.tryDeletions(ctx, install)
	}

	// if the status doesn't signal installed, needs install
	if !install.Status.Installed {
		l.Info("Installing Install")
		return ctrl.Result{}, r.install(ctx, install)
	}

	// upgrade if conditions are met
	if shouldUpgradeInstall(install) {
		l.Info("Upgrading Install")
		return ctrl.Result{}, r.upgrade(ctx, install)
	}

	return ctrl.Result{}, nil
}

func (r *InstallReconciler) install(ctx context.Context, install *v1.Install) error {
	l := log.FromContext(ctx)

	// install gateway api
	resp, err := http.Get(fmt.Sprintf("https://github.com/kubernetes-sigs/gateway-api/releases/download/%s/standard-install.yaml", install.Spec.GatewayAPI.Version))
	if err != nil {
		return err
	}
	err = r.apply(ctx, resp.Body)
	if err != nil {
		return err
	}

	// helm installs for istio
	actionConfiguration, err := createHelmAction(install.Spec.Istio.Namespace)
	if err != nil {
		return err
	}
	act := action.NewInstall(actionConfiguration)
	act.Version = install.Spec.Istio.Version
	act.Namespace = install.Spec.Istio.Namespace
	act.RepoURL = install.Spec.Istio.Repo
	act.CreateNamespace = true
	act.Wait = true
	act.Timeout = 5 * time.Minute

	// istio base
	act.ReleaseName = "base"
	values := map[string]interface{}{}
	exists, err := helmExists(actionConfiguration, act.ReleaseName)
	if err != nil {
		return err
	}
	if !exists {
		l.Info("installing istio chart", "chart", act.ReleaseName)
		c, err := getChart(act.ChartPathOptions, act.ReleaseName)
		if err != nil {
			return err
		}
		_, err = act.Run(c, values)
		if err != nil {
			return err
		}
	}

	// istio istiod
	act.ReleaseName = "istiod"
	values = map[string]interface{}{
		"profile": "ambient",
	}
	exists, err = helmExists(actionConfiguration, act.ReleaseName)
	if err != nil {
		return err
	}
	if !exists {
		l.Info("installing istio chart", "chart", act.ReleaseName)
		c, err := getChart(act.ChartPathOptions, act.ReleaseName)
		if err != nil {
			return err
		}
		_, err = act.Run(c, values)
		if err != nil {
			return err
		}
	}

	// istio cni
	act.ReleaseName = "cni"
	values = map[string]interface{}{
		"profile": "ambient",
		// TODO: need this only for k3s install
		"global": map[string]interface{}{
			"platform": "k3s",
		},
	}
	exists, err = helmExists(actionConfiguration, act.ReleaseName)
	if err != nil {
		return err
	}
	if !exists {
		l.Info("installing istio chart", "chart", act.ReleaseName)
		c, err := getChart(act.ChartPathOptions, act.ReleaseName)
		if err != nil {
			return err
		}
		_, err = act.Run(c, values)
		if err != nil {
			return err
		}
	}

	// istio ztunnel
	act.ReleaseName = "ztunnel"
	values = map[string]interface{}{}
	exists, err = helmExists(actionConfiguration, act.ReleaseName)
	if err != nil {
		return err
	}
	if !exists {
		l.Info("installing istio chart", "chart", act.ReleaseName)
		c, err := getChart(act.ChartPathOptions, act.ReleaseName)
		if err != nil {
			return err
		}
		_, err = act.Run(c, values)
		if err != nil {
			return err
		}
	}

	l.Info("installing common components")
	for _, o := range resources.CommonObjects(install) {
		err = r.Client.Create(ctx, o)
		if client.IgnoreAlreadyExists(err) != nil {
			return err
		}
	}

	l.Info("installing draft components")
	for _, o := range resources.DraftObjects(install) {
		err = r.Client.Create(ctx, o)
		if client.IgnoreAlreadyExists(err) != nil {
			return err
		}
	}

	l.Info("installing home cloud server components")
	for _, o := range resources.HomeCloudServerObjects(install) {
		err = r.Client.Create(ctx, o)
		if client.IgnoreAlreadyExists(err) != nil {
			return err
		}
	}

	return r.updateStatus(ctx, install)
}

func (r *InstallReconciler) upgrade(ctx context.Context, install *v1.Install) error {
	// TODO

	return r.updateStatus(ctx, install)
}

func (r *InstallReconciler) uninstall(ctx context.Context, install *v1.Install) error {

	// NOTE: we do not delete any CRDs (gateway API/istio) as they could be in use by other applications

	for _, o := range slices.Backward(resources.HomeCloudServerObjects(install)) {
		err := r.Client.Delete(ctx, o)
		if client.IgnoreNotFound(err) != nil {
			return err
		}
	}

	for _, o := range slices.Backward(resources.DraftObjects(install)) {
		err := r.Client.Delete(ctx, o)
		if client.IgnoreNotFound(err) != nil {
			return err
		}
	}

	for _, o := range slices.Backward(resources.CommonObjects(install)) {
		err := r.Client.Delete(ctx, o)
		if client.IgnoreNotFound(err) != nil {
			return err
		}
	}

	actionConfiguration, err := createHelmAction(install.Spec.Istio.Namespace)
	if err != nil {
		return err
	}

	act := action.NewUninstall(actionConfiguration)
	act.IgnoreNotFound = true
	act.Wait = true
	act.Timeout = 5 * time.Minute

	releases := []string{"ztunnel", "cni", "istiod", "base"}
	for _, release := range releases {
		_, err = act.Run(release)
		if err != nil {
			return err
		}
	}

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

func helmExists(actionConfiguration *action.Configuration, releaseName string) (bool, error) {
	get := action.NewGet(actionConfiguration)
	_, err := get.Run(releaseName)
	if err != nil {
		if err.Error() == "release: not found" {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (r *InstallReconciler) updateStatus(ctx context.Context, install *v1.Install) error {
	install.Status.Installed = true
	// TODO: save current spec?
	// install.Status.Values = install.Spec.Values
	return r.Status().Update(ctx, install)
}

// Apply applies the given YAML manifests to kubernetes
func (r *InstallReconciler) apply(ctx context.Context, reader io.Reader) error {
	l := log.FromContext(ctx)
	dynamicClient, err := dynamic.NewForConfig(r.Config)
	if err != nil {
		return err
	}
	dec := yaml.NewDecoder(reader)
	for {
		// parse the YAML doc
		obj := &unstructured.Unstructured{Object: map[string]interface{}{}}
		err := dec.Decode(obj.Object)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
		if obj.Object == nil {
			l.Info("skipping empty document")
			continue
		}
		// get GroupVersionResource to invoke the dynamic client
		gvk := obj.GroupVersionKind()
		restMapping, err := r.Client.RESTMapper().RESTMapping(gvk.GroupKind(), gvk.Version)
		if err != nil {
			return err
		}
		gvr := restMapping.Resource
		// apply the YAML doc
		namespace := obj.GetNamespace()
		if len(namespace) == 0 {
			namespace = "default"
		}

		applyOpts := metav1.ApplyOptions{FieldManager: "home-cloud-operator"}
		_, err = dynamicClient.Resource(gvr).Apply(context.TODO(), obj.GetName(), obj, applyOpts)
		if err != nil {
			l.Error(err, "failed to apply object", "kind", obj.GetKind(), "name", obj.GetName())
			return err
		}
		l.Info("applied YAML for object", "kind", obj.GetKind(), "name", obj.GetName())
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *InstallReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.Install{}).
		Complete(r)
}
