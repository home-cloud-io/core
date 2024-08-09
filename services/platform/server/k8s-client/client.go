package k8sclient

import (
	"context"

	opv1 "github.com/home-cloud-io/core/services/platform/operator/api/v1"

	"github.com/steady-bytes/draft/pkg/chassis"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/clientcmd"
	crclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type (
	Client interface {
		Install(ctx context.Context, spec opv1.AppSpec) error
		Delete(ctx context.Context, spec opv1.AppSpec) error
		Update(ctx context.Context, spec opv1.AppSpec) error
	}

	client struct {
		client crclient.Client
	}
)

func NewClient(logger chassis.Logger) Client {
	// NOTE: this will attempt first to build the config from the path given in the draft config and will
	// fallback on the in-cluster config if no path is given
	config := chassis.GetConfig()
	kubeConfig, err := clientcmd.BuildConfigFromFlags(config.GetString("server.k8s.master_url"), config.GetString("server.k8s.config_path"))
	if err != nil {
		logger.WithError(err).Panic("failed to build kube config")
	}

	c, err := crclient.New(kubeConfig, crclient.Options{})
	if err != nil {
		logger.WithError(err).Panic("failed to create new k8s client")
	}
	opv1.AddToScheme(c.Scheme())

	return &client{
		client: c,
	}
}

func (c *client) Install(ctx context.Context, spec opv1.AppSpec) error {
	app := &opv1.App{
		ObjectMeta: metav1.ObjectMeta{
			Name:       spec.Release,
			Namespace:  "home-cloud-system",
			Finalizers: []string{"apps.home-cloud.io/finalizer"},
		},
		Spec: spec,
	}
	return c.client.Create(ctx, app)
}

func (c *client) Delete(ctx context.Context, spec opv1.AppSpec) error {
	app := &opv1.App{
		ObjectMeta: metav1.ObjectMeta{
			Name:       spec.Release,
			Namespace:  "home-cloud-system",
		},
	}
	return c.client.Delete(ctx, app)
}

func (c *client) Update(ctx context.Context, spec opv1.AppSpec) error {
	app := &opv1.App{}
	err := c.client.Get(ctx, types.NamespacedName{
		Name:       spec.Release,
		Namespace:  "home-cloud-system",
	}, app)
	if err != nil {
		return err
	}
	app.Spec = spec
	return c.client.Update(ctx, app)
}
