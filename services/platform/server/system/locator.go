package system

import (
	"context"
	"fmt"

	dv1 "github.com/home-cloud-io/core/api/platform/daemon/v1"
	"github.com/home-cloud-io/core/services/platform/server/async"
)

type (
	Locators interface {
		AddLocator(ctx context.Context, locatorAddress string) (locator *dv1.Locator, err error)
		RemoveLocator(ctx context.Context, locatorAddress string) error
		DisableAllLocators(ctx context.Context) error
	}
)

func (c *controller) AddLocator(ctx context.Context, locatorAddress string) (locator *dv1.Locator, err error) {

	listener := async.RegisterListener(ctx, c.broadcaster, &async.ListenerOptions[*dv1.LocatorServerAdded]{
		Callback: func(event *dv1.LocatorServerAdded) (bool, error) {
			if event.Error != nil {
				return true, fmt.Errorf(event.Error.Error)
			}
			// not done yet if the locator doesn't match
			if event.Locator.Address != locatorAddress {
				return false, nil
			}
			locator = event.Locator
			return true, nil
		},
	})
	err = com.Send(&dv1.ServerMessage{
		Message: &dv1.ServerMessage_AddLocatorServerCommand{
			AddLocatorServerCommand: &dv1.AddLocatorServerCommand{
				LocatorAddress: locatorAddress,
			},
		},
	})
	if err != nil {
		return nil, err
	}
	err = listener.Listen(ctx)
	if err != nil {
		return nil, err
	}

	return locator, nil
}

func (c *controller) RemoveLocator(ctx context.Context, locatorAddress string) (err error) {

	listener := async.RegisterListener(ctx, c.broadcaster, &async.ListenerOptions[*dv1.LocatorServerRemoved]{
		Callback: func(event *dv1.LocatorServerRemoved) (bool, error) {
			if event.Error != nil {
				return true, fmt.Errorf(event.Error.Error)
			}
			// not done yet if the locator doesn't match
			if event.Address != locatorAddress {
				return false, nil
			}
			return true, nil
		},
	})
	err = com.Send(&dv1.ServerMessage{
		Message: &dv1.ServerMessage_RemoveLocatorServerCommand{
			RemoveLocatorServerCommand: &dv1.RemoveLocatorServerCommand{
				LocatorAddress: locatorAddress,
			},
		},
	})
	if err != nil {
		return err
	}
	err = listener.Listen(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (c *controller) DisableAllLocators(ctx context.Context) (err error) {

	listener := async.RegisterListener(ctx, c.broadcaster, &async.ListenerOptions[*dv1.AllLocatorsDisabled]{
		Callback: func(event *dv1.AllLocatorsDisabled) (bool, error) {
			if event.Error != nil {
				return true, fmt.Errorf(event.Error.Error)
			}
			return true, nil
		},
	})
	err = com.Send(&dv1.ServerMessage{
		Message: &dv1.ServerMessage_DisableAllLocatorsCommand{
			DisableAllLocatorsCommand: &dv1.DisableAllLocatorsCommand{},
		},
	})
	if err != nil {
		return err
	}
	err = listener.Listen(ctx)
	if err != nil {
		return err
	}

	return nil
}
