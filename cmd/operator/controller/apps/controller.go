package apps

import (
	"context"
	"fmt"
	"slices"
	"time"

	"dario.cat/mergo"
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
	"github.com/home-cloud-io/core/pkg/helm"
	"github.com/steady-bytes/draft/pkg/chassis"
)

// AppReconciler reconciles a App object
type AppReconciler struct {
	client.Client
	Scheme                *runtime.Scheme
	Config                chassis.Config
	DependencyReconcilers []dependencyReconcilerFunc
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

type dependencyReconcilerFunc func(ctx context.Context, r *AppReconciler, app *v1.App, config *AppConfig) error

const (
	AppFinalizer = "apps.home-cloud.io/finalizer"
)

var (
	preReconcilers = []dependencyReconcilerFunc{
		reconcileNamespaces,
		reconcileSecrets,
		reconcilePersistence,
		reconcileDisks,
		reconcileDatabases,
	}
	postReconcilers = []dependencyReconcilerFunc{
		reconcileRoutes,
	}
)

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

	// skip reconcile if told to ignore
	if shared.IsAnnotationTrue(app.Annotations, v1.AnnotationAppIgnore) {
		return ctrl.Result{}, nil
	}

	// if marked for deletion, try to delete/uninstall
	if app.GetDeletionTimestamp() != nil {
		l.Info("Uninstalling App")
		return ctrl.Result{}, r.tryDeletions(ctx, app)
	}

	return ctrl.Result{}, r.reconcile(ctx, app)
}

func (r *AppReconciler) HandleDiskEvent() handler.EventHandler {
	return handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, obj client.Object) (requests []reconcile.Request) {
		l := log.FromContext(ctx)
		l.Info("Reconciling Disk for Apps")

		install, err := shared.GetInstall(ctx, r.Client)
		if err != nil {
			l.Error(err, "failed to get install")
			return
		}

		requests = []reconcile.Request{}
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

func (r *AppReconciler) HandleInstallEvent() handler.EventHandler {
	return handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, obj client.Object) (requests []reconcile.Request) {
		l := log.FromContext(ctx)
		l.Info("Reconciling Install for Apps")

		apps := &v1.AppList{}
		err := r.Client.List(ctx, apps)
		if err != nil {
			l.Error(err, "failed to list apps")
			return
		}

		requests = make([]reconcile.Request, len(apps.Items))
		for i, app := range apps.Items {
			requests[i] = reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      app.Name,
					Namespace: app.Namespace,
				},
			}
		}

		return requests
	})
}

// SetupWithManager sets up the controller with the Manager.
func (r *AppReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.App{}).
		Watches(&v1.Disk{}, r.HandleDiskEvent()).
		Watches(&v1.Install{}, r.HandleInstallEvent()).
		Complete(r)
}

func (r *AppReconciler) reconcile(ctx context.Context, app *v1.App) error {

	// read combined app config from chart values and override values configured in the app
	appConfig, err := config(app)
	if err != nil {
		return err
	}

	// run pre- reconcilers
	err = r.reconcileDeps(ctx, app, appConfig, preReconcilers)
	if err != nil {
		return err
	}

	// construct helm configuration
	actionConfiguration, err := helm.ActionConfiguration(app.Namespace)
	if err != nil {
		return err
	}
	iAct := action.NewInstall(actionConfiguration)
	iAct.ReleaseName = app.Spec.Release
	iAct.Version = app.Spec.Version
	iAct.Namespace = app.Namespace
	iAct.RepoURL = repoURL(app)
	// TODO: Wait and Timeout?

	uAct := action.NewUpgrade(actionConfiguration)
	uAct.Version = app.Spec.Version
	uAct.Namespace = app.Namespace
	uAct.RepoURL = repoURL(app)
	// TODO: Wait and Timeout?

	_, values, err := getChartAndValues(iAct.ChartPathOptions, app)
	if err != nil {
		return err
	}

	// override from any changes in createDependencies
	values, err = appConfig.ToValues(values)
	if err != nil {
		return err
	}

	// install/upgrade helm chart
	err = helm.InstallOrUpgrade(ctx, actionConfiguration, iAct, uAct, values)
	if err != nil {
		return err
	}

	// run post- reconcilers
	err = r.reconcileDeps(ctx, app, appConfig, postReconcilers)
	if err != nil {
		return err
	}

	return r.updateStatus(ctx, app)
}

func (r *AppReconciler) uninstall(ctx context.Context, app *v1.App) error {
	actionConfiguration, err := helm.ActionConfiguration(app.Namespace)
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

func (r *AppReconciler) reconcileDeps(ctx context.Context, app *v1.App, config *AppConfig, funcs []dependencyReconcilerFunc) error {

	// run all reconcile functions (in order)
	for _, f := range funcs {
		err := f(ctx, r, app, config)
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

func repoURL(app *v1.App) string {
	return "https://" + app.Spec.Repo
}

func config(app *v1.App) (config *AppConfig, err error) {
	// get chart from app spec
	actionConfiguration, err := helm.ActionConfiguration(app.Namespace)
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
