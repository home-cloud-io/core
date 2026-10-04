package helm

import (
	"context"
	"fmt"
	"io"
	"os"
	"reflect"

	"gopkg.in/yaml.v3" // is this the correct package?
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/registry"
	"helm.sh/helm/v3/pkg/release"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// UnmarshalValues unmarshals the given string into a values map usable for passing
// to helm commands.
func UnmarshalValues(values string) (map[string]interface{}, error) {
	v := map[string]interface{}{}
	err := yaml.Unmarshal([]byte(values), &v)
	if err != nil {
		return v, err
	}
	return v, nil
}

// ActionConfiguration returns a helm action configuration for the given namespace.
func ActionConfiguration(namespace string) (*action.Configuration, error) {
	settings := cli.New()
	settings.SetNamespace(namespace)
	actionConfig := new(action.Configuration)

	// action.DebugLog wrapper around logr.Logger
	l := func(format string, args ...any) {
		log.Log.V(-1).Info(fmt.Sprintf(format, args...))
	}

	if err := actionConfig.Init(settings.RESTClientGetter(), namespace, os.Getenv("HELM_DRIVER"), l); err != nil {
		return nil, err
	}

	registryClient, err := registry.NewClient(registry.ClientOptWriter(io.Discard))
	if err != nil {
		return nil, err
	}

	actionConfig.RegistryClient = registryClient
	return actionConfig, nil
}

// getChart downloads the given remote chart and loads it into the returned struct.
func getChart(opt action.ChartPathOptions, chart string) (*chart.Chart, error) {
	// download the chart to the file system
	path, err := opt.LocateChart(chart, cli.New())
	if err != nil {
		return nil, err
	}
	// load chart from file
	c, err := loader.Load(path)
	if err != nil {
		return nil, err
	}
	return c, nil
}

// Get runs `helm get`
func Get(_ context.Context, cfg *action.Configuration, releaseName string) (*release.Release, error) {
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

func InstallOrUpgrade(ctx context.Context, cfg *action.Configuration, iAct *action.Install, uAct *action.Upgrade, values map[string]interface{}) error {
	l := log.FromContext(ctx)

	// get existing release
	release, err := Get(ctx, cfg, iAct.ReleaseName)
	if err != nil {
		return err
	}

	// install if no release found
	if release == nil {
		return Install(ctx, cfg, iAct, values)
	}

	// ignore if no changes
	if len(values) == 0 {
		// maps of length 0 != nil maps
		values = nil
	}
	if release.Chart.Metadata.Version == uAct.Version && (reflect.DeepEqual(release.Config, values)) {
		l.V(1).Info("ignoring unchanged helm release", "release", iAct.ReleaseName)
		return nil
	}

	// upgrade
	return Upgrade(ctx, cfg, iAct.ReleaseName, uAct, values)
}

func Install(ctx context.Context, cfg *action.Configuration, act *action.Install, values map[string]interface{}) error {
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

func Upgrade(ctx context.Context, cfg *action.Configuration, releaseName string, act *action.Upgrade, values map[string]interface{}) error {
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
