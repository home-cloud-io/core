package controller

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	v1 "github.com/home-cloud-io/core/services/platform/operator/api/v1"

	"github.com/imdario/mergo"
	ntv1 "github.com/steady-bytes/draft/api/core/control_plane/networking/v1"
	"golang.org/x/mod/semver"
	"gopkg.in/yaml.v3"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/registry"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const AppFinalizer = "apps.home-cloud.io/finalizer"

// AppReconciler reconciles a App object
type AppReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// HelmRepositoryIndex represents the index.yaml file that holds the information of helm charts within a helm repo
type HelmRepositoryIndex struct {
	APIVersion string                        `yaml:"apiVersion"`
	Entries    map[string][]HelmChartVersion `yaml:"entries"`
	Generated  time.Time                     `yaml:"generated"`
}

// HelmChartVersion represents the versions of the "entries" within a HelmRepositoryIndex
type HelmChartVersion struct {
	APIVersion  string    `yaml:"apiVersion"`
	AppVersion  string    `yaml:"appVersion"`
	Created     time.Time `yaml:"created"`
	Description string    `yaml:"description"`
	Digest      string    `yaml:"digest"`
	Name        string    `yaml:"name"`
	Type        string    `yaml:"type"`
	Urls        []string  `yaml:"urls"`
	Version     string    `yaml:"version"`
}

//+kubebuilder:rbac:groups=home-cloud.io,resources=apps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=home-cloud.io,resources=apps/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=home-cloud.io,resources=apps/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.16.3/pkg/reconcile
func (r *AppReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := log.FromContext(ctx)
	l.Info("Reconciling App")

	// Get the CRD that triggered reconciliation
	app := &v1.App{}
	err := r.Get(ctx, req.NamespacedName, app)
	if err != nil {
		if errors.IsNotFound(err) {
			l.Info("App resource not found. Assuming this means the resource was deleted and so ignoring.")
			return ctrl.Result{}, nil
		}
		l.Info("Failed to get App resource. Re-running reconcile.")
		return ctrl.Result{}, err
	}

	// if marked for deletion, try to delete/uninstall
	if app.GetDeletionTimestamp() != nil {
		l.Info("Uninstalling App")
		return ctrl.Result{}, r.tryDeletions(ctx, app)
	}

	// if the version isn't set in the status, installation is needed
	if app.Status.Version == "" {
		l.Info("Installing App")
		return ctrl.Result{}, r.install(ctx, app)
	}

	// upgrade if conditions are met
	if shouldUpgrade(app) {
		l.Info("Upgrading App")
		return ctrl.Result{}, r.upgrade(ctx, app)
	}

	// Run on a timer
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *AppReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.App{}).
		Complete(r)
}

func (r *AppReconciler) install(ctx context.Context, app *v1.App) error {
	actionConfiguration, err := createHelmAction(app.Namespace)
	if err != nil {
		return err
	}

	// construct helm configuration
	act := action.NewInstall(actionConfiguration)
	act.Version = app.Spec.Version
	act.Namespace = app.Namespace
	act.RepoURL = repoURL(app)
	act.ReleaseName = app.Spec.Release
	namespace := app.Spec.Release
	chart, values, err := getChartAndValues(act.ChartPathOptions, app)
	if err != nil {
		return err
	}

	// create namespace before installing anything else
	err = r.Client.Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	})
	if client.IgnoreAlreadyExists(err) != nil {
		return err
	}

	// read combined app config from chart values and override values configured in the app
	appConfig, err := config(chart, app)
	if err != nil {
		return err
	}

	// create secrets
	for _, s := range appConfig.Secrets {
		err := r.createSecret(ctx, s, namespace)
		if err != nil {
			return err
		}
	}

	// create persistence (PV/PVCs)
	for _, p := range appConfig.Persistence {
		err := r.createPersistence(ctx, p, app, namespace)
		if err != nil {
			return err
		}
	}

	// create database (and users/initialization scripts)
	for _, d := range appConfig.Databases {
		err := r.createDatabase(ctx, d, namespace)
		if err != nil {
			return err
		}
	}

	// create routes
	for _, route := range appConfig.Routes {
		err = r.createRoute(ctx, &ntv1.Route{
			Name: route.Name,
			Match: &ntv1.RouteMatch{
				Prefix: "/",
				Host:   fmt.Sprintf("%s.local", route.Name),
			},
			Endpoint: &ntv1.Endpoint{
				Host: fmt.Sprintf("%s.%s.svc.cluster.local", route.Service.Name, namespace),
				Port: route.Service.Port,
			},
		})
		if err != nil {
			return err
		}
	}

	// finally, install helm chart
	_, err = act.Run(chart, values)
	if err != nil {
		return err
	}

	return r.updateStatus(ctx, app)
}

func (r *AppReconciler) upgrade(ctx context.Context, app *v1.App) error {
	actionConfiguration, err := createHelmAction(app.Namespace)
	if err != nil {
		return err
	}

	act := action.NewUpgrade(actionConfiguration)
	act.Version = app.Spec.Version
	act.Namespace = app.Namespace
	act.RepoURL = repoURL(app)

	chart, values, err := getChartAndValues(act.ChartPathOptions, app)
	if err != nil {
		return err
	}

	_, err = act.Run(app.Spec.Release, chart, values)
	if err != nil {
		return err
	}

	return r.updateStatus(ctx, app)
}

func (r *AppReconciler) uninstall(ctx context.Context, app *v1.App) error {
	actionConfiguration, err := createHelmAction(app.Namespace)
	if err != nil {
		return err
	}

	act := action.NewUninstall(actionConfiguration)
	act.IgnoreNotFound = true

	_, err = act.Run(app.Spec.Release)
	if err != nil {
		return err
	}

	// TODO: option to hard-delete all add-on components (namespace, secrets, PV/PVCs, etc.)

	return nil
}

func (r *AppReconciler) updateStatus(ctx context.Context, app *v1.App) error {
	app.Status.Version = app.Spec.Version
	app.Status.Values = app.Spec.Values
	return r.Status().Update(ctx, app)
}

func (r *AppReconciler) tryDeletions(ctx context.Context, app *v1.App) error {
	if controllerutil.ContainsFinalizer(app, AppFinalizer) {
		err := r.uninstall(ctx, app)
		if err != nil {
			return err
		}

		controllerutil.RemoveFinalizer(app, AppFinalizer)
		err = r.Update(ctx, app)
		if err != nil {
			return err
		}
	}
	return nil
}

// HELPERS

// createHelmAction creates a helm action configuration with the given namespace.
func createHelmAction(namespace string) (*action.Configuration, error) {
	settings := cli.New()
	settings.SetNamespace(namespace)
	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(settings.RESTClientGetter(), namespace, os.Getenv("HELM_DRIVER"), klog.Infof); err != nil {
		return nil, err
	}

	registryClient, err := registry.NewClient(registry.ClientOptWriter(io.Discard))
	if err != nil {
		return nil, err
	}

	actionConfig.RegistryClient = registryClient
	return actionConfig, nil
}

// getChartAndValues returns the chart and values for a given app by downloading the chart from the registry and converting the values
// from the string in the CRD to a map.
func getChartAndValues(opt action.ChartPathOptions, app *v1.App) (*chart.Chart, map[string]interface{}, error) {
	// download the chart to the file system
	path, err := opt.LocateChart(app.Spec.Chart, cli.New())
	if err != nil {
		return nil, nil, err
	}
	// load chart from file
	chart, err := loader.Load(path)
	if err != nil {
		return nil, nil, err
	}

	// values for the chart install
	values := make(map[string]interface{})
	if len(app.Spec.Values) != 0 {
		err := yaml.Unmarshal([]byte(app.Spec.Values), &values)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to unmarshal values: %v", err)
		}
	}

	return chart, values, nil
}

// shouldUpgrade determines if the given app needs upgrading based on the version and values.
func shouldUpgrade(app *v1.App) bool {
	installedVersion := app.Status.Version
	if installedVersion != "" {
		installedVersion = "v" + installedVersion
	}
	requestedVersion := app.Spec.Version
	if requestedVersion != "" {
		requestedVersion = "v" + requestedVersion
	}
	// UPGRADE
	// if the requested version is greater than the installed version
	// OR
	// if the current values in the spec are different than those in the status
	return semver.Compare(requestedVersion, installedVersion) != 0 || app.Spec.Values != app.Status.Values
}

func repoURL(app *v1.App) string {
	return "https://" + app.Spec.Repo
}

func config(chart *chart.Chart, app *v1.App) (config *AppConfig, err error) {
	values, err := yaml.Marshal(chart.Values)
	if err != nil {
		return nil, err
	}

	base, err := valuesToConfig(values)
	if err != nil {
		return nil, err
	}

	override, err := valuesToConfig([]byte(app.Spec.Values))
	if err != nil {
		return nil, err
	}

	err = mergo.Merge(override, base)
	if err != nil {
		return nil, err
	}

	return override, nil
}

func valuesToConfig(values []byte) (*AppConfig, error) {
	if len(values) == 0 {
		return &AppConfig{}, nil
	}
	appValues := AppValues{}
	err := yaml.Unmarshal(values, &appValues)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal app values: %v", err)
	}
	return &appValues.Config, nil
}
