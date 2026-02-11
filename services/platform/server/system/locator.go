package system

import (
	"context"
	"slices"

	opv1 "github.com/home-cloud-io/core/services/platform/operator/api/v1"
	k8sclient "github.com/home-cloud-io/core/services/platform/server/k8s-client"

	"k8s.io/apimachinery/pkg/types"
)

type (
	Locators interface {
		AddLocator(ctx context.Context, wgInterfaceName string, locatorAddress string) (err error)
		RemoveLocator(ctx context.Context, wgInterfaceName string, locatorAddress string) error
	}
)

func (c *controller) AddLocator(ctx context.Context, wgInterfaceName string, locatorAddress string) (err error) {
	// get wireguard resource
	wgInterface := &opv1.Wireguard{}
	err = c.k8sclient.Get(ctx, types.NamespacedName{
		Name:      wgInterfaceName,
		Namespace: k8sclient.DefaultHomeCloudNamespace,
	}, wgInterface)
	if err != nil {
		return err
	}

	// add locator if new
	if slices.Contains(wgInterface.Spec.Locators, locatorAddress) {
		return nil
	}
	wgInterface.Spec.Locators = append(wgInterface.Spec.Locators, locatorAddress)

	// update resource
	return c.k8sclient.Update(ctx, wgInterface)
}

func (c *controller) RemoveLocator(ctx context.Context, wgInterfaceName string, locatorAddress string) (err error) {
	// get wireguard resource
	wgInterface := &opv1.Wireguard{}
	err = c.k8sclient.Get(ctx, types.NamespacedName{
		Name:      wgInterfaceName,
		Namespace: k8sclient.DefaultHomeCloudNamespace,
	}, wgInterface)
	if err != nil {
		return err
	}

	// remove locator
	wgInterface.Spec.Locators = slices.DeleteFunc(wgInterface.Spec.Locators, func(s string) bool {
		return s == locatorAddress
	})

	// update resource
	return c.k8sclient.Update(ctx, wgInterface)
}
