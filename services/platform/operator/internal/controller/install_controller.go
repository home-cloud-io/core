package controller

import (
	"context"

	"helm.sh/helm/v3/pkg/action"
	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	v1 "github.com/home-cloud-io/core/services/platform/operator/api/v1"
)

// InstallReconciler reconciles a Install object
type InstallReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

const (
	InstallFinalizer = "install.home-cloud.io/finalizer"

	defaultIstioVersion   = "1.27.1"
	defaultIstioNamespace = "istio-system"
	defaultIstioRepoURL   = "https://istio-release.storage.googleapis.com/charts"
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
		if errors.IsNotFound(err) {
			l.Info("Install resource not found. Assuming this means the resource was deleted and so ignoring.")
			return ctrl.Result{}, nil
		}
		l.Info("Failed to get Install resource. Re-running reconcile.")
		return ctrl.Result{}, err
	}

	// if marked for deletion, try to delete/uninstall
	if install.GetDeletionTimestamp() != nil {
		l.Info("Uninstalling Install")
		return ctrl.Result{}, r.tryDeletions(ctx, install)
	}

	// if the version isn't set in the status, installation is needed
	if install.Status.Version == "" {
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

	actionConfiguration, err := createHelmAction(defaultIstioNamespace)
	if err != nil {
		return err
	}


	// TODO: genericize this
	act := action.NewInstall(actionConfiguration)
	act.Version = defaultIstioVersion
	act.Namespace = defaultIstioNamespace
	act.RepoURL = defaultIstioRepoURL
	act.CreateNamespace = true


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
		// TODO: need this for k3s install
		// global:
		//   platform: k3s
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

	// ingress gateway
	l.Info("installing ingress gateway")
	gateway := &gwv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{
			Name: "ingress-gateway",
			Namespace: defaultIstioNamespace,
		},
		Spec: gwv1.GatewaySpec{
			GatewayClassName: "istio",
			Listeners: []gwv1.Listener{
				{
					Name: "http",
					Port: 8080,
					Protocol: gwv1.HTTPProtocolType,
					AllowedRoutes: &gwv1.AllowedRoutes{
						Namespaces: &gwv1.RouteNamespaces{
							From: ptr.To[gwv1.FromNamespaces]("All"),
						},
					},
				},
			},
		},
	}
	err = r.Client.Create(ctx, gateway)
	if client.IgnoreAlreadyExists(err) != nil {
		return err
	}

	l.Info("installing default http route")
	httpRoute := &gwv1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name: "demo",
			Namespace: "default",
		},
		Spec: gwv1.HTTPRouteSpec{
			CommonRouteSpec: gwv1.CommonRouteSpec{
				ParentRefs: []gwv1.ParentReference{
					{
						Name: "ingress-gateway",
						Namespace: ptr.To[gwv1.Namespace](defaultIstioNamespace),
					},
				},
			},
			Hostnames: []gwv1.Hostname{
				"localhost",
			},
			Rules: []gwv1.HTTPRouteRule{
				{
					BackendRefs: []gwv1.HTTPBackendRef{
						{
							BackendRef: gwv1.BackendRef{
								BackendObjectReference: gwv1.BackendObjectReference{
									Name: "server",
									Port: ptr.To[gwv1.PortNumber](80),
								},
							},
						},
					},
				},
			},
		},
	}
	err = r.Client.Create(ctx, httpRoute)
	if client.IgnoreAlreadyExists(err) != nil {
		return err
	}

	// home cloud server
	l.Info("installing (fake) home cloud server")
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "demo-deployment",
			Namespace: "default",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: ptr.To[int32](1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "demo",
				},
			},
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "demo",
					},
				},
				Spec: apiv1.PodSpec{
					Containers: []apiv1.Container{
						{
							Name:  "web",
							Image: "nginx:1.12",
							Ports: []apiv1.ContainerPort{
								{
									Name:          "http",
									Protocol:      apiv1.ProtocolTCP,
									ContainerPort: 80,
								},
							},
						},
					},
				},
			},
		},
	}
	err = r.Client.Create(ctx, deployment)
	if client.IgnoreAlreadyExists(err) != nil {
		return err
	}

	l.Info("installing (fake) home cloud server service")
	service := &apiv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: "demo",
			Namespace: "default",
		},
		Spec: apiv1.ServiceSpec{
			Type: apiv1.ServiceTypeClusterIP,
			Ports: []apiv1.ServicePort{
				{
					Name: "http",
					Port: 80,
				},
			},
			Selector: map[string]string{
				"app": "demo",
			},
		},
	}
	err = r.Client.Create(ctx, service)
	if client.IgnoreAlreadyExists(err) != nil {
		return err
	}

	return r.updateStatus(ctx, install)
}

func (r *InstallReconciler) upgrade(ctx context.Context, install *v1.Install) error {
	// TODO

	return r.updateStatus(ctx, install)
}

func (r *InstallReconciler) uninstall(ctx context.Context, install *v1.Install) error {
	actionConfiguration, err := createHelmAction(defaultIstioNamespace)
	if err != nil {
		return err
	}

	act := action.NewUninstall(actionConfiguration)
	act.IgnoreNotFound = true

	_, err = act.Run("istio-base")
	if err != nil {
		return err
	}

	_, err = act.Run("istio-istiod")
	if err != nil {
		return err
	}

	_, err = act.Run("istio-cni")
	if err != nil {
		return err
	}

	_, err = act.Run("istio-ztunnel")
	if err != nil {
		return err
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
	install.Status.Version = install.Spec.Version
	// install.Status.Values = install.Spec.Values
	return r.Status().Update(ctx, install)
}

// SetupWithManager sets up the controller with the Manager.
func (r *InstallReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.Install{}).
		Complete(r)
}
