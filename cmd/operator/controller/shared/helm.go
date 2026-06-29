package shared

import (
	"fmt"
	"io"
	"os"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/registry"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// CreateHelmAction creates a helm action configuration with the given namespace.
func CreateHelmAction(namespace string) (*action.Configuration, error) {
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

func GetChart(opt action.ChartPathOptions, chart string) (*chart.Chart, error) {
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