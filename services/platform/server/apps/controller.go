package apps

import (
	"cmp"
	"context"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/steady-bytes/draft/pkg/chassis"
	"golang.org/x/mod/semver"

	v1 "github.com/home-cloud-io/core/api/platform/server/v1"
	opv1 "github.com/home-cloud-io/core/services/platform/operator/api/v1"
	k8sclient "github.com/home-cloud-io/core/services/platform/server/k8s-client"
	hstrings "github.com/home-cloud-io/core/services/platform/server/utils/strings"
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
		// PrettyHealthcheck will retrieve the health of all installed apps and include the displayName
		// from the chart, icon, and readme/description.
		PrettyHealthcheck(ctx context.Context, logger chassis.Logger) ([]*v1.AppHealth, error)
		// AutoUpdate will check for and install app updates on a schedule.
		AutoUpdate(ctx context.Context, logger chassis.Logger)
		// GetAppStorage will retrieve the app storage volumes for all installed apps.
		GetAppStorage(ctx context.Context, logger chassis.Logger) ([]*v1.AppStorage, error)
	}

	controller struct {
		k8sclient k8sclient.Apps
		cronID    cron.EntryID
		cr        *cron.Cron
	}
)

func NewController(kclient k8sclient.Apps) Controller {
	return &controller{
		k8sclient: kclient,
	}
}

const (
	ErrFailedToGetApps              = "failed to get apps"
	ErrFailedToGetAppStorage        = "failed to get app storage"
	ErrFailedToGetComponentVersions = "failed to get component versions"
	ErrFailedToGetLogs              = "failed to get logs"

	DefaultAutoUpdateAppsSchedule = "0 3 * * *"
)

// TODO: there is no deduplication currently so if two stores have the same applications they will simply be repeated
func (c *controller) Store(ctx context.Context, logger chassis.Logger) ([]*v1.App, error) {
	var (
		err  error
		apps []*v1.App
	)

	stores, err := c.stores(ctx, logger)
	// return immediately if no app store returned results
	if err != nil && len(stores) == 0 {
		return apps, err
	}

	for _, store := range stores {

		for _, v := range store.Entries {
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

			resp, err := http.Get(fmt.Sprintf("%s/%s-%s/charts/%s/README.md", store.RawChartUrl, app.Name, app.Version, app.Name))
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
					err := c.k8sclient.InstallApp(ctx, opv1.AppSpec{
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
	err = c.k8sclient.InstallApp(ctx, opv1.AppSpec{
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
	err := c.k8sclient.DeleteApp(ctx, opv1.AppSpec{
		Release: request.Release,
	})
	if err != nil {
		return err
	}
	return nil
}

func (c *controller) Update(ctx context.Context, logger chassis.Logger, request *v1.UpdateAppRequest) error {
	err := c.k8sclient.UpdateApp(ctx, opv1.AppSpec{
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

func (c *controller) PrettyHealthcheck(ctx context.Context, logger chassis.Logger) ([]*v1.AppHealth, error) {
	// get app health
	apps, err := c.k8sclient.Healthcheck(ctx)
	if err != nil {
		return nil, err
	}

	// add in the display info from the store
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

func (c *controller) AutoUpdate(ctx context.Context, logger chassis.Logger) {
	f := func() {
		err := c.UpdateAll(context.Background(), logger)
		if err != nil {
			logger.WithError(err).Error("failed to run auto app update job")
		}
	}

	// create new if no current entry, otherwise remove old entry
	if c.cronID == 0 {
		c.cr = cron.New()
	} else {
		c.cr.Remove(c.cronID)
	}

	// get schedule from settings
	settings, err := c.k8sclient.Settings(ctx)
	if err != nil {
		logger.WithError(err).Panic("failed to get settings")
	}
	cron := hstrings.Default(settings.AutoUpdateAppsSchedule, DefaultAutoUpdateAppsSchedule)

	// add new entry
	logger.WithField("cron", cron).Info("setting apps auto-update interval")
	id, err := c.cr.AddFunc(cron, f)
	if err != nil {
		logger.WithError(err).Panic("failed to initialize auto-update for apps")
	}
	c.cronID = id

	// no-op if already started
	c.cr.Start()
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
