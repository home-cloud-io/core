package apps

import (
	"context"
	"fmt"
	"slices"
	"time"

	"dario.cat/mergo"
	"golang.org/x/mod/semver"
	"gopkg.in/yaml.v3"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	v1 "github.com/home-cloud-io/core/api/crds/v1"
	"github.com/home-cloud-io/core/cmd/operator/controller/shared"
	"github.com/home-cloud-io/core/pkg/compare"
	"github.com/steady-bytes/draft/pkg/chassis"
)

const AppFinalizer = "apps.home-cloud.io/finalizer"

// AppReconciler reconciles a App object
type AppReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Config chassis.Config
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

	// upgrade as default
	l.Info("Upgrading App")
	return ctrl.Result{}, r.upgrade(ctx, app)
}

func (r *AppReconciler) ReconcileDisk() handler.EventHandler {
	return handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, obj client.Object) (requests []reconcile.Request) {
		l := log.FromContext(ctx)
		l.Info("Reconciling Disk for Apps")

		requests = []reconcile.Request{}

		install, err := shared.GetInstall(ctx, r.Client)
		if err != nil {
			l.Error(err, "failed to get install")
			return
		}

		for _, appName := range install.Spec.Settings.StorageApps {
			requests = append(requests, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      appName,
					Namespace: install.Namespace,
				},
			})
		}

		return
	})
}

// SetupWithManager sets up the controller with the Manager.
func (r *AppReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.App{}).
		// watch disks so we can trigger a reconcile on storage apps
		Watches(&v1.Disk{}, r.ReconcileDisk()).
		Complete(r)
}

func (r *AppReconciler) install(ctx context.Context, app *v1.App) error {

	// read combined app config from chart values and override values configured in the app
	appConfig, err := config(app)
	if err != nil {
		return err
	}

	err = r.createDependencies(ctx, app, appConfig)
	if err != nil {
		return err
	}

	// construct helm configuration
	actionConfiguration, err := shared.CreateHelmAction(app.Namespace)
	if err != nil {
		return err
	}
	act := action.NewInstall(actionConfiguration)
	act.Version = app.Spec.Version
	act.Namespace = app.Namespace
	act.RepoURL = repoURL(app)
	act.ReleaseName = app.Spec.Release
	chart, values, err := getChartAndValues(act.ChartPathOptions, app)
	if err != nil {
		return err
	}

	// override from any changes in createDependencies
	values, err = appConfig.ToValues(values)
	if err != nil {
		return err
	}

	// finally, install helm chart
	_, err = act.Run(chart, values)
	if err != nil {
		return err
	}

	// create routes (after helm so that Services already exist for DNS annotation)
	for _, route := range appConfig.Routes {
		err = r.createRoute(ctx, appConfig.Namespace, route)
		if err != nil {
			return err
		}
	}

	return r.updateStatus(ctx, app)
}

func (r *AppReconciler) upgrade(ctx context.Context, app *v1.App) error {

	// read combined app config from chart values and override values configured in the app
	appConfig, err := config(app)
	if err != nil {
		return err
	}

	err = r.createDependencies(ctx, app, appConfig)
	if err != nil {
		return err
	}

	// construct helm configuration
	actionConfiguration, err := shared.CreateHelmAction(app.Namespace)
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

	// override from any changes in createDependencies
	values, err = appConfig.ToValues(values)
	if err != nil {
		return err
	}

	_, err = act.Run(app.Spec.Release, chart, values)
	if err != nil {
		return err
	}

	// update routes (after helm so that Services already exist for DNS annotation)
	for _, route := range appConfig.Routes {
		err = r.createRoute(ctx, appConfig.Namespace, route)
		if err != nil {
			return err
		}
	}

	return r.updateStatus(ctx, app)
}

func (r *AppReconciler) uninstall(ctx context.Context, app *v1.App) error {
	actionConfiguration, err := shared.CreateHelmAction(app.Namespace)
	if err != nil {
		return err
	}

	act := action.NewUninstall(actionConfiguration)
	act.IgnoreNotFound = true

	_, err = act.Run(app.Spec.Release)
	if err != nil {
		return err
	}

	// read combined app config from chart values and override values configured in the app
	appConfig, err := config(app)
	if err != nil {
		return err
	}

	// delete all routes
	// we always delete routes so that we aren't routing to a missing app
	for _, route := range appConfig.Routes {
		err = r.deleteRoute(ctx, appConfig.Namespace, route.Name)
		if err != nil {
			return err
		}
	}

	// delete all other dependencies if requested (namespace, secrets, persistence, databases/users, etc.)
	if shared.IsAnnotationTrue(app.Annotations, v1.AnnotationAppCleanUninstall) {
		err = r.deleteDependencies(ctx, app, appConfig)
		if err != nil {
			return err
		}
	}

	return nil
}

// createDependencies creates app dependencies and saves any updated values to AppConfig (e.g. disk claims)
func (r *AppReconciler) createDependencies(ctx context.Context, app *v1.App, appConfig *AppConfig) error {
	var (
		err error
	)

	// create namespace before installing anything else
	err = r.Client.Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: appConfig.Namespace,
			Labels: map[string]string{
				"istio.io/dataplane-mode": "ambient",
			},
		},
	})
	if client.IgnoreAlreadyExists(err) != nil {
		return err
	}
	// TODO: probably a better way to do this
	err = r.Client.Update(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: appConfig.Namespace,
			Labels: map[string]string{
				"istio.io/dataplane-mode": "ambient",
			},
		},
	})
	if err != nil {
		return err
	}

	// create secrets
	for _, s := range appConfig.Secrets {
		err := r.createSecret(ctx, s, appConfig.Namespace)
		if err != nil {
			return err
		}
	}

	// create persistence (PV/PVCs)
	for _, p := range appConfig.Persistence {
		err := r.createPersistence(ctx, p, app, appConfig.Namespace)
		if err != nil {
			return err
		}
	}

	// if a storage app, create disk PV/PVCs
	install, err := shared.GetInstall(ctx, r.Client)
	if err != nil {
		return err
	}
	if slices.Contains(install.Spec.Settings.StorageApps, app.Name) {
		disks := &v1.DiskList{}
		err := r.Client.List(ctx, disks)
		if err != nil {
			return err
		}
		appConfig.Disks = []AppDisk{}
		for _, disk := range disks.Items {
			if disk.Spec.SystemDisk {
				continue
			}

			claimName, err := r.createDiskPersistence(ctx, disk, app, appConfig.Namespace)
			if err != nil {
				return err
			}

			// save created claim name for helm install/upgrade
			appConfig.Disks = append(appConfig.Disks, AppDisk{
				// use disk.Alias if available for user visibility
				Name:      compare.Default(disk.Spec.Alias, disk.Name),
				ClaimName: claimName,
			})
		}
	}

	// create database (and users/initialization scripts)
	for _, d := range appConfig.Databases {
		err := r.createDatabase(ctx, d, appConfig.Namespace)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *AppReconciler) deleteDependencies(ctx context.Context, app *v1.App, appConfig *AppConfig) error {
	var (
		err error
	)

	// delete secrets
	for _, s := range appConfig.Secrets {
		err := r.deleteSecret(ctx, s, appConfig.Namespace)
		if err != nil {
			return err
		}
	}

	// delete persistence (PV/PVCs)
	for _, p := range appConfig.Persistence {
		// TODO: this doesn't actually wipe the files...
		err := r.deletePersistence(ctx, p, app, appConfig.Namespace)
		if err != nil {
			return err
		}
	}

	// if a storage app, delete disk PV/PVCs
	install, err := shared.GetInstall(ctx, r.Client)
	if err != nil {
		return err
	}
	if slices.Contains(install.Spec.Settings.StorageApps, app.Name) {
		disks := &v1.DiskList{}
		err := r.Client.List(ctx, disks)
		if err != nil {
			return err
		}
		appConfig.Disks = make([]AppDisk, len(disks.Items))
		for _, disk := range disks.Items {
			if disk.Spec.SystemDisk {
				continue
			}
			_, err := r.deleteDiskPersistence(ctx, disk, app, app.Namespace)
			if err != nil {
				return err
			}
		}
	}

	// delete databases (and users)
	for _, d := range appConfig.Databases {
		err := r.deleteDatabase(ctx, d, appConfig.Namespace)
		if err != nil {
			return err
		}
	}

	// delete namespace
	err = r.Client.Delete(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: appConfig.Namespace,
		},
	})
	if err != nil {
		return err
	}

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

func config(app *v1.App) (config *AppConfig, err error) {
	// get chart from app spec
	actionConfiguration, err := shared.CreateHelmAction(app.Namespace)
	if err != nil {
		return nil, err
	}
	act := action.NewInstall(actionConfiguration)
	act.Version = app.Spec.Version
	act.Namespace = app.Namespace
	act.RepoURL = repoURL(app)
	act.ReleaseName = app.Spec.Release
	chart, _, err := getChartAndValues(act.ChartPathOptions, app)
	if err != nil {
		return nil, err
	}

	// convert values from chart into config
	values, err := yaml.Marshal(chart.Values)
	if err != nil {
		return nil, err
	}
	base, err := valuesToConfig(values)
	if err != nil {
		return nil, err
	}

	// convert values from app spec into config
	override, err := valuesToConfig([]byte(app.Spec.Values))
	if err != nil {
		return nil, err
	}

	// merge chart values and app spec values
	err = mergo.Merge(override, base)
	if err != nil {
		return nil, err
	}

	// TODO: should we rethink this?
	override.Namespace = app.Spec.Release
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
