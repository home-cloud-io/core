package controller

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"slices"
	"time"

	"connectrpc.com/connect"
	"dario.cat/mergo"
	"gopkg.in/yaml.v3"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/release"
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

	dv1 "github.com/home-cloud-io/core/api/platform/daemon/v1"
	dv1connect "github.com/home-cloud-io/core/api/platform/daemon/v1/v1connect"
	v1 "github.com/home-cloud-io/core/services/platform/operator/api/v1"
	resources "github.com/home-cloud-io/core/services/platform/operator/internal/controller/resources"
)

// TODO: cancel install on crd update so that failed installs don't get stuck until timeout
// might also need a lock on reconcile?

// InstallReconciler reconciles a Install object
type InstallReconciler struct {
	client.Client
	DaemonClient dv1connect.DaemonServiceClient
	Scheme       *runtime.Scheme
	Config       *rest.Config
}

const (
	InstallFinalizer = "install.home-cloud.io/finalizer"
	ReleasesURL      = "https://github.com/home-cloud-io/core/releases/download/"
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

	l.Info("Reconciling Install")
	return ctrl.Result{}, r.reconcile(ctx, install)
}

func (r *InstallReconciler) reconcile(ctx context.Context, install *v1.Install) error {
	l := log.FromContext(ctx)
	var err error

	l.Info("reconciling gateway api crds")
	resp, err := http.Get(fmt.Sprintf("%s/%s/standard-install.yaml", install.Spec.GatewayAPI.Source, install.Spec.GatewayAPI.Version))
	if err != nil {
		return err
	}
	err = r.apply(ctx, resp.Body)
	if err != nil {
		return err
	}

	l.Info("reconciling namespaces")
	for _, o := range resources.NamespaceObjects(install) {
		err = kubeCreateOrUpdate(ctx, r.Client, o)
		if err != nil {
			return err
		}
	}

	if !install.Spec.Istio.Disable {
		l.Info("reconciling ingress gateway")
		err = r.installResources(ctx, resources.GatewayObjects(install))
		if err != nil {
			return err
		}

		l.Info("reconciling istio install")
		err = reconcileIstio(ctx, install)
		if err != nil {
			return err
		}
	} else {
		if install.Status.Istio.Version != "" {
			l.Info("istio is disabled: removing ingress gateway")
			err = r.uninstallResources(ctx, resources.GatewayObjects(install))
			if err != nil {
				return err
			}

			l.Info("istio is disabled: removing previous installation")
			err = uninstallIstio(ctx, install)
			if err != nil {
				return err
			}
		}
	}

	installed := install.Status.Server.Tag != ""
	err = r.reconcileObjects(ctx, "server", install.Spec.Server.Disable, installed, resources.ServerObjects(install))
	if err != nil {
		return err
	}

	installed = install.Status.MDNS.Tag != ""
	err = r.reconcileObjects(ctx, "mdns", install.Spec.MDNS.Disable, installed, resources.MDNSObjects(install))
	if err != nil {
		return err
	}

	installed = install.Status.Tunnel.Tag != ""
	err = r.reconcileObjects(ctx, "tunnel", install.Spec.Tunnel.Disable, installed, resources.TunnelObjects(install))
	if err != nil {
		return err
	}

	installed = install.Status.Daemon.Tag != ""
	err = r.reconcileObjects(ctx, "daemon", install.Spec.Daemon.Disable, installed, resources.DaemonObjects(install))
	if err != nil {
		return err
	}

	if !install.Spec.Daemon.Disable {
		if !install.Spec.Daemon.System.Disable {
			l.Info("reconciling system install")
			_, err := r.DaemonClient.Upgrade(ctx, connect.NewRequest(&dv1.UpgradeRequest{
				Source:  install.Spec.Daemon.System.Source,
				Version: install.Spec.Daemon.System.Version,
			}))
			if err != nil {
				return err
			}
		}

		if !install.Spec.Daemon.Kubernetes.Disable {
			l.Info("reconciling kubernetes install")
			_, err := r.DaemonClient.UpgradeKubernetes(ctx, connect.NewRequest(&dv1.UpgradeKubernetesRequest{
				Version: install.Spec.Daemon.Kubernetes.Version,
			}))
			if err != nil {
				return err
			}
		}
	}

	return r.updateStatus(ctx, install)
}

func reconcileIstio(ctx context.Context, install *v1.Install) error {

	cfg, err := createHelmAction(install.Spec.Istio.Namespace)
	if err != nil {
		return err
	}
	iAct := action.NewInstall(cfg)
	iAct.Version = install.Spec.Istio.Version
	iAct.Namespace = install.Spec.Istio.Namespace
	iAct.RepoURL = install.Spec.Istio.Source
	iAct.Wait = true
	iAct.Timeout = 5 * time.Minute

	uAct := action.NewUpgrade(cfg)
	uAct.Version = install.Spec.Istio.Version
	uAct.Namespace = install.Spec.Istio.Namespace
	uAct.RepoURL = install.Spec.Istio.Source
	uAct.Wait = true
	uAct.Timeout = 5 * time.Minute

	// istio base
	iAct.ReleaseName = "base"
	err = helmInstallOrUpgrade(ctx, cfg, iAct, uAct, install.Spec.Istio.Base.Values)
	if err != nil {
		return err
	}

	// istio istiod
	iAct.ReleaseName = "istiod"
	err = helmInstallOrUpgrade(ctx, cfg, iAct, uAct, install.Spec.Istio.Istiod.Values)
	if err != nil {
		return err
	}

	// istio cni
	iAct.ReleaseName = "cni"
	err = helmInstallOrUpgrade(ctx, cfg, iAct, uAct, install.Spec.Istio.CNI.Values)
	if err != nil {
		return err
	}

	// istio ztunnel
	iAct.ReleaseName = "ztunnel"
	err = helmInstallOrUpgrade(ctx, cfg, iAct, uAct, install.Spec.Istio.Ztunnel.Values)
	if err != nil {
		return err
	}

	return nil
}

func uninstallIstio(ctx context.Context, install *v1.Install) error {
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

func (r *InstallReconciler) uninstall(ctx context.Context, install *v1.Install) error {

	// NOTE: we do not delete any CRDs (gateway API/istio) as they could be in use by other applications

	err := r.uninstallResources(ctx, slices.Concat(
		resources.GatewayObjects(install),
		resources.ServerObjects(install),
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
	err = r.uninstallResources(ctx, resources.NamespaceObjects(install))
	if err != nil {
		return err
	}

	return nil
}

func (r *InstallReconciler) reconcileObjects(ctx context.Context, name string, disable bool, installed bool, objects []client.Object) error {
	l := log.FromContext(ctx)

	// uninstall if disabled and currently installed
	if disable && installed {
		if installed {
			l.Info(fmt.Sprintf("%s is disabled: removing previous installation", name))
			return r.uninstallResources(ctx, objects)
		}
	}

	// otherwise create/update
	l.Info(fmt.Sprintf("reconciling %s install", name))
	return r.installResources(ctx, objects)
}

func (r *InstallReconciler) installResources(ctx context.Context, objects []client.Object) error {
	for _, o := range objects {
		err := kubeCreateOrUpdate(ctx, r.Client, o)
		if err != nil {
			return err
		}
	}
	return nil
}

// TODO: move this to resources package and accept the kube client as a parameter?
func (r *InstallReconciler) uninstallResources(ctx context.Context, objects []client.Object) error {
	for _, o := range slices.Backward(objects) {
		err := r.Client.Delete(ctx, o)
		if client.IgnoreNotFound(err) != nil {
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

func kubeCreateOrUpdate(ctx context.Context, kube client.Client, obj client.Object) error {
	err := kube.Create(ctx, obj)
	if kerrors.IsAlreadyExists(err) {
		// this is a bit of a mess and might not be totally necessary but it creates a new instance
		// of the same underlying type in obj (which must be a pointer) so that we don't overwrite all
		// fields when we really only want the ResourceVersion
		c := reflect.New(reflect.TypeOf(obj).Elem()).Interface().(client.Object)
		err := kube.Get(ctx, client.ObjectKeyFromObject(obj), c)
		if err != nil {
			return err
		}
		obj.SetResourceVersion(c.GetResourceVersion())
		return kube.Update(ctx, obj)
	}
	return err
}

func helmExists(cfg *action.Configuration, releaseName string) (bool, error) {
	release, err := helmGet(cfg, releaseName)
	if err != nil {
		return false, err
	}
	if release != nil {
		return true, nil
	}
	return false, nil
}

func helmGet(cfg *action.Configuration, releaseName string) (*release.Release, error) {
	get := action.NewGet(cfg)
	release, err := get.Run(releaseName)
	if err != nil {
		if err.Error() == "release: not found" {
			return nil, nil
		}
		return nil, err
	}
	return release, nil
}

func (r *InstallReconciler) updateStatus(ctx context.Context, install *v1.Install) error {

	install.Status.Version = install.Spec.Version

	if !install.Spec.GatewayAPI.Disable {
		install.Status.GatewayAPI = v1.GatewayAPIStatus{
			URL:     install.Spec.GatewayAPI.Source,
			Version: install.Spec.GatewayAPI.Version,
		}
	} else {
		install.Status.GatewayAPI = v1.GatewayAPIStatus{}
	}

	if !install.Spec.Istio.Disable {
		install.Status.Istio = v1.IstioStatus{
			Repo:    install.Spec.Istio.Source,
			Version: install.Spec.Istio.Version,
		}
	} else {
		install.Status.Istio = v1.IstioStatus{}
	}

	if !install.Spec.Server.Disable {
		install.Status.Server = v1.ServerStatus{
			Image: install.Spec.Server.Image,
			Tag:   install.Spec.Server.Tag,
		}
	} else {
		install.Status.Server = v1.ServerStatus{}
	}

	if !install.Spec.MDNS.Disable {
		install.Status.MDNS = v1.MDNSStatus{
			Image: install.Spec.MDNS.Image,
			Tag:   install.Spec.MDNS.Tag,
		}
	} else {
		install.Status.MDNS = v1.MDNSStatus{}
	}

	if !install.Spec.Tunnel.Disable {
		install.Status.Tunnel = v1.TunnelStatus{
			Image: install.Spec.Tunnel.Image,
			Tag:   install.Spec.Tunnel.Tag,
		}
	} else {
		install.Status.Tunnel = v1.TunnelStatus{}
	}

	if !install.Spec.Daemon.Disable {
		install.Status.Daemon = v1.DaemonStatus{
			Image: install.Spec.Daemon.Image,
			Tag:   install.Spec.Daemon.Tag,
		}
	} else {
		install.Status.Daemon = v1.DaemonStatus{}
	}

	return r.Status().Update(ctx, install)
}

func helmInstallOrUpgrade(ctx context.Context, cfg *action.Configuration, iAct *action.Install, uAct *action.Upgrade, values string) error {
	l := log.FromContext(ctx)

	// get user values
	v := map[string]interface{}{}
	err := yaml.Unmarshal([]byte(values), &v)
	if err != nil {
		return err
	}

	// force ambient profile for istio charts
	if slices.Contains([]string{"base", "istiod", "cni"}, iAct.ReleaseName) {
		v["profile"] = "ambient"
	}

	// get existing release
	release, err := helmGet(cfg, iAct.ReleaseName)
	if err != nil {
		return err
	}

	// install if no release found
	if release == nil {
		return helmInstall(ctx, cfg, iAct, v)
	}

	// ignore if no changes
	if len(v) == 0 {
		// maps of length 0 != nil maps
		v = nil
	}
	if release.Chart.Metadata.Version == uAct.Version && (reflect.DeepEqual(release.Config, v)) {
		l.V(1).Info("ignoring unchanged helm release", "release", iAct.ReleaseName)
		return nil
	}

	// upgrade
	return helmUpgrade(ctx, cfg, iAct.ReleaseName, uAct, v)
}

func helmInstall(ctx context.Context, cfg *action.Configuration, act *action.Install, values map[string]interface{}) error {
	l := log.FromContext(ctx)
	l.Info("installing helm chart", "chart", act.ChartPathOptions.RepoURL)
	c, err := getChart(act.ChartPathOptions, act.ReleaseName)
	if err != nil {
		return err
	}
	_, err = act.RunWithContext(ctx, c, values)
	if err != nil {
		return err
	}
	return nil
}

func helmUpgrade(ctx context.Context, cfg *action.Configuration, releaseName string, act *action.Upgrade, values map[string]interface{}) error {
	l := log.FromContext(ctx)
	l.Info("upgrading helm release", "release", releaseName)
	c, err := getChart(act.ChartPathOptions, releaseName)
	if err != nil {
		return err
	}
	_, err = act.RunWithContext(ctx, releaseName, c, values)
	if err != nil {
		return err
	}
	return nil
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
		l.V(1).Info("applied YAML for object", "kind", obj.GetKind(), "name", obj.GetName())
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *InstallReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.Install{}).
		Complete(r)
}
