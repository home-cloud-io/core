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
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	dv1 "github.com/home-cloud-io/core/api/platform/daemon/v1"
	v1 "github.com/home-cloud-io/core/services/platform/operator/api/v1"
	"github.com/home-cloud-io/core/services/platform/operator/resources"
)

// TODO: cancel install on crd update so that failed installs don't get stuck until timeout
// might also need a lock on reconcile?

// InstallReconciler reconciles a Install object
type InstallReconciler struct {
	client.Client
	DiscoveryClient *discovery.DiscoveryClient
	Scheme          *runtime.Scheme
	Config          *rest.Config
	// global cancel function to shutdown the manager (useful for operator upgrades)
	Cancel          context.CancelFunc
}

const (
	InstallFinalizer     = "install.home-cloud.io/finalizer"
	ReleasesURL          = "https://github.com/home-cloud-io/core/releases/download/"
	DefaultDaemonAddress = "http://daemon.home-cloud-system"
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
		if reflect.DeepEqual(install.Status.Version, *oldStatus) {
			err := r.Status().Update(ctx, install)
			if err != nil {
				panic(err)
			}
		}
	}()

	return ctrl.Result{}, r.reconcile(ctx, install)
}

func (r *InstallReconciler) reconcile(ctx context.Context, install *v1.Install) error {
	l := log.FromContext(ctx)

	// OPERATOR
	// install the operator before the other components since it may be necessary to patch a bug in itself to
	// prevent getting locked up on other components
	installed := install.Status.Operator != nil
	err := r.reconcileObjects(ctx, "operator", install.Spec.Operator.Disable, installed, resources.OperatorObjects(install))
	if err != nil {
		return err
	}
	if !install.Spec.Operator.Disable {
		if install.Spec.Operator.Tag != install.Status.Operator.Tag ||
			install.Spec.Operator.Image != install.Status.Operator.Image {
			install.Status.Operator = &v1.OperatorStatus{
				Image: install.Spec.Operator.Image,
				Tag:   install.Spec.Operator.Tag,
			}
			err := r.Status().Update(ctx, install)
			if err != nil {
				return err
			}

			// shutdown if operator has updated so that the new replica can take over
			r.Cancel()
		}
	} else {
		install.Status.Operator = nil
	}

	// Home Cloud CRDs
	err = r.reconcileHomeCloudCRDs(ctx, install)
	if err != nil {
		return err
	}

	// GATEWAY API
	err = r.reconcileGatewayAPI(ctx, install)
	if err != nil {
		return err
	}

	// NAMESPACES
	l.Info("reconciling namespaces")
	for _, o := range resources.NamespaceObjects(install) {
		err = kubeCreateOrUpdate(ctx, r.Client, o)
		if err != nil {
			return err
		}
	}
	// no status update

	// ISTIO
	if !install.Spec.Istio.Disable {
		// NOTE: we can't simply skip an istio install if the version hasn't changed since the values
		// might have changed with no version bump

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

		install.Status.Istio = &v1.IstioStatus{
			Source:  install.Spec.Istio.Source,
			Version: install.Spec.Istio.Version,
		}
	} else {
		// only try and uninstall if currently installed
		if install.Status.Istio != nil {
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
		install.Status.Istio = nil
	}

	// SERVER
	installed = install.Status.Server != nil
	err = r.reconcileObjects(ctx, "server", install.Spec.Server.Disable, installed, resources.ServerObjects(install))
	if err != nil {
		return err
	}
	if !install.Spec.Server.Disable {
		install.Status.Server = &v1.ServerStatus{
			Image: install.Spec.Server.Image,
			Tag:   install.Spec.Server.Tag,
		}
	} else {
		install.Status.Server = nil
	}

	// MDNS
	installed = install.Status.MDNS != nil
	err = r.reconcileObjects(ctx, "mdns", install.Spec.MDNS.Disable, installed, resources.MDNSObjects(install))
	if err != nil {
		return err
	}
	if !install.Spec.MDNS.Disable {
		install.Status.MDNS = &v1.MDNSStatus{
			Image: install.Spec.MDNS.Image,
			Tag:   install.Spec.MDNS.Tag,
		}
	} else {
		install.Status.MDNS = nil
	}

	// TUNNEL
	installed = install.Status.Tunnel != nil
	err = r.reconcileObjects(ctx, "tunnel", install.Spec.Tunnel.Disable, installed, resources.TunnelObjects(install))
	if err != nil {
		return err
	}
	if !install.Spec.Tunnel.Disable {
		install.Status.Tunnel = &v1.TunnelStatus{
			Image: install.Spec.Tunnel.Image,
			Tag:   install.Spec.Tunnel.Tag,
		}
	} else {
		install.Status.Tunnel = nil
	}

	// DAEMON
	installed = install.Status.Daemon != nil
	err = r.reconcileObjects(ctx, "daemon", install.Spec.Daemon.Disable, installed, resources.DaemonObjects(install))
	if err != nil {
		return err
	}
	if !install.Spec.Daemon.Disable {
		install.Status.Daemon = &v1.DaemonStatus{
			Image: install.Spec.Daemon.Image,
			Tag:   install.Spec.Daemon.Tag,
		}
	} else {
		install.Status.Daemon = nil
	}

	// SYSTEM
	err = r.reconcileSystem(ctx, install)
	if err != nil {
		return err
	}

	// KUBERNETES
	err = r.reconcileKubernetes(ctx, install)
	if err != nil {
		return err
	}

	l.Info("reconcile complete")
	return nil
}

func (r *InstallReconciler) reconcileHomeCloudCRDs(ctx context.Context, install *v1.Install) error {
	l := log.FromContext(ctx)

	if install.Spec.Version != install.Status.Version {
		l.Info("reconciling home cloud crds")

		resp, err := http.Get(fmt.Sprintf("%s/%s/crds.yaml", ReleasesURL, install.Spec.Version))
		if err != nil {
			return err
		}
		err = r.apply(ctx, resp.Body)
		if err != nil {
			return err
		}

		install.Status.Version = install.Spec.Version
	} else {
		l.V(1).Info("unchanged home cloud crds: skipping reconcile")
		return nil
	}

	return nil
}

func (r *InstallReconciler) reconcileGatewayAPI(ctx context.Context, install *v1.Install) error {
	l := log.FromContext(ctx)

	if !install.Spec.GatewayAPI.Disable {
		if install.Status.GatewayAPI == nil ||
			install.Spec.GatewayAPI.Source != install.Status.GatewayAPI.Source ||
			install.Spec.GatewayAPI.Version != install.Status.GatewayAPI.Version {
			l.Info("reconciling gateway api crds")

			resp, err := http.Get(fmt.Sprintf("%s/%s/standard-install.yaml", install.Spec.GatewayAPI.Source, install.Spec.GatewayAPI.Version))
			if err != nil {
				return err
			}
			err = r.apply(ctx, resp.Body)
			if err != nil {
				return err
			}

			install.Status.GatewayAPI = &v1.GatewayAPIStatus{
				Source:  install.Spec.GatewayAPI.Source,
				Version: install.Spec.GatewayAPI.Version,
			}
		} else {
			l.V(1).Info("unchanged gateway api install: skipping reconcile")
			return nil
		}

	} else {
		install.Status.GatewayAPI = nil
	}
	return nil
}

func (r *InstallReconciler) reconcileSystem(ctx context.Context, install *v1.Install) error {
	l := log.FromContext(ctx)

	// skip if daemon or system is disabled
	if install.Spec.Daemon.Disable || install.Spec.Daemon.System.Disable {
		l.V(1).Info("daemon or system disabled: skipping reconcile")
		install.Status.Daemon.System = nil
		return nil
	}

	// only upgrade if not installed or the source/version is changed
	if install.Status.Daemon.System == nil ||
		install.Spec.Daemon.System.Source != install.Status.Daemon.System.Source ||
		install.Spec.Daemon.System.Version != install.Status.Daemon.System.Version {

		l.V(1).Info("reconciling system install")
		daemonClient := DaemonClient(install.Spec.Daemon.Address)

		// first check the spec version against the daemon since an upgrade may have broken the previous reconcile iteration
		// before status could be written and we want to avoid an infinite loop of triggering the same upgrade over and over
		versionResp, err := daemonClient.Version(ctx, connect.NewRequest(&dv1.VersionRequest{}))
		if err != nil {
			return err
		}
		if versionResp.Msg.Version == install.Spec.Daemon.System.Version {
			l.V(1).Info("new system version already installed: updating status")
			install.Status.Daemon.System = &v1.SystemStatus{
				// TODO: should Version() return source?
				Source:  install.Spec.Daemon.System.Source,
				Version: install.Spec.Daemon.System.Version,
			}
			return nil
		}

		l.Info("upgrading system install")
		_, err = daemonClient.Upgrade(ctx, connect.NewRequest(&dv1.UpgradeRequest{
			Source:  install.Spec.Daemon.System.Source,
			Version: install.Spec.Daemon.System.Version,
		}))
		if err != nil {
			return err
		}
		l.Info("system upgrade complete")
		install.Status.Daemon.System = &v1.SystemStatus{
			Source:  install.Spec.Daemon.System.Source,
			Version: install.Spec.Daemon.System.Version,
		}
	}

	l.V(1).Info("unchanged system install: skipping reconcile")
	return nil
}

func (r *InstallReconciler) reconcileKubernetes(ctx context.Context, install *v1.Install) error {
	l := log.FromContext(ctx)

	// skip if daemon or kubernetes is disabled
	if install.Spec.Daemon.Disable || install.Spec.Daemon.Kubernetes.Disable {
		l.V(1).Info("daemon or kubernetes disabled: skipping reconcile")
		install.Status.Daemon.Kubernetes = nil
		return nil
	}

	// only upgrade if not installed or the version is changed
	if install.Status.Daemon.Kubernetes == nil ||
		install.Spec.Daemon.Kubernetes.Version != install.Status.Daemon.Kubernetes.Version {
		l.V(1).Info("reconciling kubernetes install")
		daemonClient := DaemonClient(install.Spec.Daemon.Address)

		// first check the spec version against the cluster since an upgrade may have broken the previous reconcile
		// call before status could be written and we want to avoid an infinite loop of triggering the same upgrade over and over
		version, err := r.DiscoveryClient.ServerVersion()
		if err != nil {
			return err
		}
		if version.GitVersion == install.Spec.Daemon.Kubernetes.Version {
			l.V(1).Info("kubernetes version already installed: updating status")
			install.Status.Daemon.Kubernetes = &v1.KubernetesStatus{
				Version: install.Spec.Daemon.Kubernetes.Version,
			}
			return nil
		}

		l.Info("upgrading kubernetes install")
		_, err = daemonClient.UpgradeKubernetes(ctx, connect.NewRequest(&dv1.UpgradeKubernetesRequest{
			Version: install.Spec.Daemon.Kubernetes.Version,
		}))
		if err != nil {
			return err
		}
		l.Info("kubernetes upgrade complete")
		install.Status.Daemon.Kubernetes = &v1.KubernetesStatus{
			Version: install.Spec.Daemon.Kubernetes.Version,
		}
	}

	l.V(1).Info("unchanged kubernetes install: skipping reconcile")
	return nil
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

	// NOTE: we do not delete any CRDs as they could be in use by other applications
	// 			 we also do not uninstall operator resources for obvious reasons

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

	if disable {
		// uninstall if disabled and currently installed
		if installed {
			l.Info(fmt.Sprintf("%s is disabled: removing previous installation", name))
			return r.uninstallResources(ctx, objects)
		}
		return nil
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
