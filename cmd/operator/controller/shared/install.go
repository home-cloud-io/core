package shared

import (
	"context"

	"dario.cat/mergo"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/home-cloud-io/core/api/crds/v1"
	"github.com/home-cloud-io/core/pkg/install/resources"
)

func GetInstall(ctx context.Context, c client.Client) (*v1.Install, error) {
	// get current install config
	install := &v1.Install{}
	err := c.Get(ctx, types.NamespacedName{
		Name:      "install",
		Namespace: "home-cloud-system",
	}, install)
	if err != nil {
		return nil, err
	}

	// set defaults: any values set on the resource will override the defaults
	err = mergo.Merge(install, resources.DefaultInstall)
	if err != nil {
		return nil, err
	}

	return install, nil
}