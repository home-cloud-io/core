package k8sclient

import (
	"context"
	"strings"

	webv1 "github.com/home-cloud-io/core/api/platform/server/v1"
	opv1 "github.com/home-cloud-io/core/services/platform/operator/api/v1"

	"github.com/steady-bytes/draft/pkg/chassis"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/clientcmd"
	crclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type (
	Client interface {
		Apps
		System
	}
	Apps interface {
		// Install will install the given app
		Install(ctx context.Context, spec opv1.AppSpec) error
		// Delete will delete the given app
		Delete(ctx context.Context, spec opv1.AppSpec) error
		// Update will update the given app
		//
		// NOTE: An empty value on the spec will be applied as empty and will NOT
		// default to the existing value.
		Update(ctx context.Context, spec opv1.AppSpec) error
		// Installed returns whether or not the app with the given name is currently installed
		Installed(ctx context.Context, name string) (installed bool, err error)
		// Healthcheck will retrieve the health of all installed apps
		Healthcheck(ctx context.Context) ([]*webv1.AppHealth, error)
		// InstalledApps will retrive all installed apps
		InstalledApps(ctx context.Context) ([]opv1.App, error)
	}
	System interface {
		// CurrentImages will retrieve the current images of all system containers. System
		// containers are considered to be those in the `home-cloud-system` and `draft-system`
		// namespaces.
		CurrentImages(ctx context.Context) ([]*webv1.ImageVersion, error)
	}

	client struct {
		client crclient.Client
	}
)

const (
	homeCloudNamespace = "home-cloud-system"
	draftNamespace     = "draft-system"
)

func NewClient(logger chassis.Logger) Client {
	// NOTE: this will attempt first to build the config from the path given in the draft config and will
	// fallback on the in-cluster config if no path is given
	config := chassis.GetConfig()
	kubeConfig, err := clientcmd.BuildConfigFromFlags(config.GetString("server.k8s.master_url"), config.GetString("server.k8s.config_path"))
	if err != nil {
		logger.WithError(err).Error("failed to read kube config")
	}

	c, err := crclient.New(kubeConfig, crclient.Options{})
	if err != nil {
		logger.WithError(err).Error("failed to create new k8s client")
	} else {
		opv1.AddToScheme(c.Scheme())
	}

	return &client{
		client: c,
	}
}

func (c *client) Install(ctx context.Context, spec opv1.AppSpec) error {
	app := &opv1.App{
		ObjectMeta: metav1.ObjectMeta{
			Name:       spec.Release,
			Namespace:  homeCloudNamespace,
			Finalizers: []string{"apps.home-cloud.io/finalizer"},
		},
		Spec: spec,
	}
	return c.client.Create(ctx, app)
}

func (c *client) Delete(ctx context.Context, spec opv1.AppSpec) error {
	app := &opv1.App{
		ObjectMeta: metav1.ObjectMeta{
			Name:      spec.Release,
			Namespace: homeCloudNamespace,
		},
	}
	return c.client.Delete(ctx, app)
}

func (c *client) Update(ctx context.Context, spec opv1.AppSpec) error {
	app := &opv1.App{}
	err := c.client.Get(ctx, types.NamespacedName{
		Name:      spec.Release,
		Namespace: homeCloudNamespace,
	}, app)
	if err != nil {
		return err
	}
	app.Spec = spec
	return c.client.Update(ctx, app)
}

func (c *client) CurrentImages(ctx context.Context) ([]*webv1.ImageVersion, error) {
	var (
		// processing as a map keeps from having duplicates
		images = map[string]*webv1.ImageVersion{}
		err    error
	)

	// draft containers
	err = c.getCurrentImageVersions(ctx, draftNamespace, images)
	if err != nil {
		return nil, err
	}

	// home-cloud containers
	err = c.getCurrentImageVersions(ctx, homeCloudNamespace, images)
	if err != nil {
		return nil, err
	}

	// convert map to slice
	imagesSlice := make([]*webv1.ImageVersion, len(images))
	index := 0
	for _, image := range images {
		imagesSlice[index] = image
		index++
	}

	return imagesSlice, nil
}

func (c *client) Healthcheck(ctx context.Context) ([]*webv1.AppHealth, error) {

	// get all installed apps
	apps := &opv1.AppList{}
	err := c.client.List(ctx, apps, &crclient.ListOptions{
		Namespace: homeCloudNamespace,
	})
	if err != nil {
		return nil, err
	}

	// process each app and check all app pods for status
	checks := make([]*webv1.AppHealth, len(apps.Items))
	for index, app := range apps.Items {
		checks[index] = &webv1.AppHealth{
			Name:   app.Name,
			Status: webv1.AppStatus_APP_STATUS_HEALTHY,
		}

		// get all pods in app namespace
		pods := &corev1.PodList{}
		err := c.client.List(ctx, pods, &crclient.ListOptions{
			Namespace: app.Name,
		})
		if err != nil {
			return nil, err
		}

		for _, pod := range pods.Items {
			// if any pod isn't in running status mark app as unhealthy and break
			if pod.Status.Phase != corev1.PodRunning {
				checks[index].Status = webv1.AppStatus_APP_STATUS_UNHEALTHY
				break
			}
		}
	}

	return checks, nil
}

func (c *client) Installed(ctx context.Context, name string) (installed bool, err error) {
	apps := &opv1.App{}
	err = c.client.Get(ctx, types.NamespacedName{
		Namespace: homeCloudNamespace,
		Name:      name,
	}, apps)
	if err != nil {
		// if the error is NotFound, then the app is NOT installed
		if crclient.IgnoreNotFound(err) == nil {
			return false, nil
		}
		// unknown error so return it
		return false, err
	}

	return true, nil
}

func (c *client) InstalledApps(ctx context.Context) ([]opv1.App, error) {
	apps := &opv1.AppList{}
	err := c.client.List(ctx, apps, &crclient.ListOptions{
		Namespace: homeCloudNamespace,
	})
	if err != nil {
		return nil, err
	}

	return apps.Items, nil
}

func (c *client) getCurrentImageVersions(ctx context.Context, namespace string, images map[string]*webv1.ImageVersion) error {
	pods := &corev1.PodList{}
	err := c.client.List(ctx, pods, &crclient.ListOptions{
		Namespace: namespace,
	})
	if err != nil {
		return err
	}

	for _, p := range pods.Items {
		for _, c := range p.Spec.Containers {
			name := strings.Split(c.Image, ":")[0]
			currentVersion := strings.Split(c.Image, ":")[1]
			images[name] = &webv1.ImageVersion{
				Image:   name,
				Current: currentVersion,
			}
		}
	}

	return nil
}
