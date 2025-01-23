package apps

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"time"

	v1 "github.com/home-cloud-io/core/api/platform/server/v1"
	opv1 "github.com/home-cloud-io/core/services/platform/operator/api/v1"
	k8sclient "github.com/home-cloud-io/core/services/platform/server/k8s-client"
	kvclient "github.com/home-cloud-io/core/services/platform/server/kv-client"

	"github.com/robfig/cron/v3"
	"github.com/steady-bytes/draft/pkg/chassis"
	"golang.org/x/mod/semver"
)

type (
	Controller interface {
		// Install will install the requested app.
		Install(ctx context.Context, logger chassis.Logger, request *v1.InstallAppRequest) error
		// Delete will delete the requested app.
		Delete(ctx context.Context, logger chassis.Logger, request *v1.DeleteAppRequest) error
		// Update will update the requested app.
		//
		// NOTE: An empty value on the spec will be applied as empty and will NOT
		// default to the existing value.
		Update(ctx context.Context, logger chassis.Logger, request *v1.UpdateAppRequest) error
		// Store returns all apps currently available in the store.
		Store(ctx context.Context, logger chassis.Logger) ([]*v1.App, error)
		// UpdateAll will update all apps to the latest available version in the store.
		UpdateAll(ctx context.Context, logger chassis.Logger) error
		// Healthcheck will retrieve the health of all installed apps.
		Healthcheck(ctx context.Context, logger chassis.Logger) ([]*v1.AppHealth, error)
		// AutoUpdate will check for and install app updates on a schedule. It is designed to
		// be called at bootup.
		AutoUpdate(logger chassis.Logger)
		// GetAppStorage will retrieve the app storage volumes for all installed apps.
		GetAppStorage(ctx context.Context, logger chassis.Logger) ([]*v1.AppStorage, error)
	}

	controller struct {
		k8sclient k8sclient.Apps
	}
)

func NewController(logger chassis.Logger) Controller {
	chassis.GetConfig().SetDefault(autoUpdateCronConfigKey, "0 3 * * *")
	return &controller{
		k8sclient: k8sclient.NewClient(logger),
	}
}

const (
	ErrFailedToGetApps              = "failed to get apps"
	ErrFailedToGetAppStorage        = "failed to get app storage"
	ErrFailedToGetComponentVersions = "failed to get component versions"

	autoUpdateCronConfigKey = "server.updates.apps_auto_update_cron"
	rawChartBaseUrl         = "https://raw.githubusercontent.com/home-cloud-io/store"
)

func (c *controller) Store(ctx context.Context, logger chassis.Logger) ([]*v1.App, error) {
	var (
		err      error
		apps     []*v1.App
		appStore = &v1.AppStoreEntries{}
	)

	logger.Info("getting apps in store (from cache)")
	err = kvclient.Get(ctx, kvclient.APP_STORE_ENTRIES_KEY, appStore)
	if err != nil {
		logger.WithError(err).Error("failed to get app store entries from cache")
		return nil, errors.New(ErrFailedToGetApps)
	}

	for _, v := range appStore.Entries {
		// TODO: get the latest, right now this assumes the `app` slice is already sorted by version
		// append the first app of the app store entry to to the `apps` slice
		if len(v.Apps) > 0 {
			apps = append(apps, v.Apps[0])
		}
	}

	healths, err := c.k8sclient.Healthcheck(ctx)
	if err != nil {
		logger.WithError(err).Error("failed to check installed app health during store query")
		return nil, err
	}

	// add extra information not from the index (e.g. readme and installed flag)
	for _, app := range apps {
		resp, err := http.Get(fmt.Sprintf("%s/%s-%s/charts/%s/README.md", rawChartBaseUrl, app.Name, app.Version, app.Name))
		if err != nil {
			logger.WithFields(chassis.Fields{
				"app":         app.Name,
				"app_version": app.Version,
			}).WithError(err).Error("failed to get readme for app")
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			logger.WithFields(chassis.Fields{
				"app":         app.Name,
				"app_version": app.Version,
			}).WithError(err).Error("failed to read body of response while getting readme for app")
		}
		app.Readme = string(body)

		for _, health := range healths {
			if app.Name == health.Name {
				app.Installed = true
			}
		}
	}

	// sort apps by name
	slices.SortFunc(apps, func(a, b *v1.App) int {
		return cmp.Compare(a.Name, b.Name)
	})

	return apps, nil
}

func (c *controller) Install(ctx context.Context, logger chassis.Logger, request *v1.InstallAppRequest) error {
	// check dependencies for app from the store and install if needed
	store, err := c.Store(ctx, logger)
	if err != nil {
		return err
	}
	for _, app := range store {
		if request.Chart == app.Name {
			for _, dep := range app.Dependencies {
				log := logger.WithField("dependency", dep.Name)
				log.Info("checking dependency")
				installed, err := c.k8sclient.Installed(ctx, dep.Name)
				if err != nil {
					log.WithError(err).Error("failed to check if dependency is installed")
					return err
				}
				if !installed {

					// get latest version of dep
					var latest string
					for _, storeDep := range store {
						if storeDep.Name == dep.Name {
							latest = storeDep.Version
						}
					}

					log.Info("dependency is needed: installing")
					err := c.k8sclient.Install(ctx, opv1.AppSpec{
						Chart:   dep.Name,
						Repo:    strings.TrimPrefix(dep.Repository, "https://"),
						Release: dep.Name,
						Version: latest,
					})
					if err != nil {
						log.WithError(err).Error("failed to install app")
						return err
					}

					// wait on dependency install
					timeCtx, cancel := context.WithTimeout(ctx, 3*time.Minute)
					err = c.waitForInstall(timeCtx, log, dep.Name)
					cancel()
					if err != nil {
						log.WithError(err).Error("failed to wait for dependency install")
						return err
					}
				}
			}
		}
	}

	// install requested app
	logger.Info("installing requested app")
	err = c.k8sclient.Install(ctx, opv1.AppSpec{
		Chart:   request.Chart,
		Repo:    request.Repo,
		Release: request.Release,
		Values:  request.Values,
		Version: request.Version,
	})
	if err != nil {
		logger.WithError(err).Error("failed to install app")
		return err
	}

	// wait on app install
	timeCtx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	err = c.waitForInstall(timeCtx, logger, request.Release)
	cancel()
	if err != nil {
		logger.WithError(err).Error("failed to wait for app install")
		return err
	}

	return nil
}

func (c *controller) Delete(ctx context.Context, logger chassis.Logger, request *v1.DeleteAppRequest) error {
	err := c.k8sclient.Delete(ctx, opv1.AppSpec{
		Release: request.Release,
	})
	if err != nil {
		return err
	}
	return nil
}

func (c *controller) Update(ctx context.Context, logger chassis.Logger, request *v1.UpdateAppRequest) error {
	err := c.k8sclient.Update(ctx, opv1.AppSpec{
		Chart:   request.Chart,
		Repo:    request.Repo,
		Release: request.Release,
		Values:  request.Values,
		Version: request.Version,
	})
	if err != nil {
		return err
	}
	return nil
}

func (c *controller) UpdateAll(ctx context.Context, logger chassis.Logger) error {
	logger.Info("updating all apps")
	settings := &v1.DeviceSettings{}
	err := kvclient.Get(ctx, kvclient.DEFAULT_DEVICE_SETTINGS_KEY, settings)
	if err != nil {
		logger.WithError(err).Error("failed to get device settings")
		return err
	}

	if !settings.AutoUpdateApps {
		logger.Info("auto update apps not enabled")
		return nil
	}

	storeApps, err := c.Store(ctx, logger)
	if err != nil {
		logger.WithError(err).Error("failed to get apps in store")
		return err
	}

	installedApps, err := c.k8sclient.InstalledApps(ctx)
	if err != nil {
		logger.WithError(err).Error("failed to get installed apps")
		return err
	}

	// check each installed app for an update and install it if needed
	for _, installed := range installedApps {
		logger.WithField("app", installed.Name).Info("processing installed app")
		for _, store := range storeApps {
			if installed.Name == store.Name {
				log := logger.WithFields(chassis.Fields{
					"app":               installed.Name,
					"installed_version": installed.Spec.Version,
					"latest_version":    store.Version,
				})
				log.Info("checking if update is needed")
				if semver.Compare("v"+store.Version, "v"+installed.Spec.Version) == 1 {
					log.Info("update is needed")
					err := c.Update(ctx, logger, &v1.UpdateAppRequest{
						// keep everything the same except the version
						Chart:   installed.Spec.Chart,
						Repo:    installed.Spec.Repo,
						Release: installed.Spec.Release,
						Values:  installed.Spec.Values,
						Version: store.Version,
					})
					if err != nil {
						log.WithFields(chassis.Fields{
							"app":               installed.Name,
							"installed_version": installed.Spec.Version,
							"latest_version":    store.Version,
						}).WithError(err).Error("failed to update app")
						// don't return, try to update the other apps
					}
				} else {
					log.Info("no update needed")
				}
			}
		}
	}

	logger.Info("finished updating all apps")
	return nil
}

func (c *controller) Healthcheck(ctx context.Context, logger chassis.Logger) ([]*v1.AppHealth, error) {
	// get app health
	apps, err := c.k8sclient.Healthcheck(ctx)
	if err != nil {
		return nil, err
	}

	// add in the display info from the store
	// TODO: should move this to a different method separate from Healthcheck
	store, err := c.Store(ctx, logger)
	if err != nil {
		return nil, err
	}
	for _, app := range apps {
		for _, storeApp := range store {
			if app.Name == storeApp.Name {
				name := app.Name
				displayName, ok := storeApp.Annotations["displayName"]
				if ok {
					name = displayName
				}
				app.Display = &v1.AppDisplay{
					Name:        name,
					IconUrl:     storeApp.Icon,
					Description: storeApp.Description,
				}
			}
		}
	}

	// sort apps by name
	slices.SortFunc(apps, func(a, b *v1.AppHealth) int {
		return cmp.Compare(a.Name, b.Name)
	})

	return apps, nil
}

func (c *controller) AutoUpdate(logger chassis.Logger) {
	cr := cron.New()
	f := func() {
		ctx := context.Background()
		err := c.UpdateAll(ctx, logger)
		if err != nil {
			logger.WithError(err).Error("failed to run auto app update job")
		}
	}
	cron := chassis.GetConfig().GetString(autoUpdateCronConfigKey)
	logger.WithField("cron", cron).Info("setting apps auto-update interval")
	_, err := cr.AddFunc(cron, f)
	if err != nil {
		logger.WithError(err).Panic("failed to initialize auto-update for apps")
	}
	cr.Start()
}

func (c *controller) GetAppStorage(ctx context.Context, logger chassis.Logger) ([]*v1.AppStorage, error) {
	apps, err := c.k8sclient.InstalledApps(ctx)
	if err != nil {
		logger.WithError(err).Error("failed to get installed apps")
		return nil, err
	}

	storage, err := c.k8sclient.AppStorage(ctx, apps)
	if err != nil {
		logger.WithError(err).Error("failed to get app storage")
		return nil, err
	}

	return storage, err
}

func (c *controller) waitForInstall(ctx context.Context, logger chassis.Logger, appName string) error {
	for {
		if ctx.Err() != nil {
			logger.WithError(ctx.Err()).Error("context is done")
			return ctx.Err()
		}
		appsHealth, err := c.k8sclient.Healthcheck(ctx)
		if err != nil {
			logger.WithError(err).Error("failed to check apps health")
			return err
		}
		for _, app := range appsHealth {
			if app.Name == appName {
				if app.Status == v1.AppStatus_APP_STATUS_HEALTHY {
					logger.Info("installation completed")
					return nil
				}
				break
			}
		}
		logger.Info("not yet installed")

		time.Sleep(5 * time.Second)
	}
}
