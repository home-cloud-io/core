package system

import (
	"context"
	"errors"
	"fmt"

	dv1 "github.com/home-cloud-io/core/api/platform/daemon/v1"
	v1 "github.com/home-cloud-io/core/api/platform/server/v1"
	kvclient "github.com/home-cloud-io/core/services/platform/server/kv-client"
	"github.com/steady-bytes/draft/pkg/chassis"
)

type (
	Device interface {
		// GetServerSettings returns the current server settings after filtering out the
		// admin username and password.
		GetServerSettings(ctx context.Context) (*v1.DeviceSettings, error)
		// SetServerSettings updates the settings on the server with the given values
		SetServerSettings(ctx context.Context, logger chassis.Logger, settings *v1.DeviceSettings) error
		// IsDeviceSetup checks if the device is already setup by checking if the DEFAULT_DEVICE_SETTINGS_KEY key exists in the key-value store
		// with the default settings model
		IsDeviceSetup(ctx context.Context) (bool, error)
		// InitializeDevice initializes the device with the given settings. It first checks if the device is already setup
		// Uses the user-provided password to set the password for the "admin" user on the device
		// and save the remaining settings in the key-value store
		InitializeDevice(ctx context.Context, logger chassis.Logger, settings *v1.DeviceSettings) error
		// Login receives a username and password and returns a token.
		//
		// NOTE: the token is unimplemented so this only checks the validity of the password for now
		Login(ctx context.Context, username, password string) (string, error)
	}
)

const (
	ErrDeviceAlreadySetup = "device already setup"

	ErrFailedToCreateSettings = "failed to create device settings"
	ErrFailedToGetSettings    = "failed to get device settings"
	ErrFailedToSetSettings    = "failed to save device settings"
)

// DEVICE

func (c *controller) GetServerSettings(ctx context.Context) (*v1.DeviceSettings, error) {
	settings := &v1.DeviceSettings{}
	err := kvclient.Get(ctx, kvclient.DEFAULT_DEVICE_SETTINGS_KEY, settings)
	if err != nil {
		return nil, err
	}

	settings.AdminUser.Password = "" // don't return the password

	return settings, nil
}

func (c *controller) SetServerSettings(ctx context.Context, logger chassis.Logger, settings *v1.DeviceSettings) error {

	// set the device settings on the host (via the daemon)
	err := c.saveSettings(ctx, logger, &dv1.SaveSettingsCommand{
		AdminPassword:  settings.AdminUser.Password,
		TimeZone:       settings.Timezone,
		EnableSsh:      settings.EnableSsh,
		TrustedSshKeys: settings.TrustedSshKeys,
	})
	if err != nil {
		return err
	}

	// salt and hash given password if set
	// otherwise, get existing value from cache
	if settings.AdminUser.Password != "" {
		logger.Info("updating admin user password")
		// get seed salt value from blueprint
		seed, err := getSaltValue(ctx)
		if err != nil {
			return err
		} else {
			// salt & hash the meat before you put in on the grill
			settings.AdminUser.Password = hashPassword(settings.AdminUser.Password, []byte(seed))
		}
	} else {
		existingSettings := &v1.DeviceSettings{}
		err := kvclient.Get(ctx, kvclient.DEFAULT_DEVICE_SETTINGS_KEY, existingSettings)
		if err != nil {
			return fmt.Errorf(ErrFailedToGetSettings)
		}
		settings.AdminUser.Password = existingSettings.AdminUser.Password
	}

	_, err = kvclient.Set(ctx, kvclient.DEFAULT_DEVICE_SETTINGS_KEY, settings)
	if err != nil {
		return errors.New(ErrFailedToCreateSettings)
	}

	return nil
}

func (c *controller) IsDeviceSetup(ctx context.Context) (bool, error) {
	// list is used to get all the `DeviceSettings` objects in the key-value store
	// it will not fail if the key does not exist like `Get` would
	settings := &v1.DeviceSettings{}
	list, err := kvclient.List(ctx, settings)
	if err != nil {
		return false, errors.New(ErrFailedToGetSettings)
	}

	if len(list) == 0 {
		return false, nil
	}
	return true, nil
}

func (c *controller) InitializeDevice(ctx context.Context, logger chassis.Logger, settings *v1.DeviceSettings) error {
	yes, err := c.IsDeviceSetup(ctx)
	if err != nil {
		return errors.New(ErrFailedToGetSettings)
	} else if yes {
		logger.Warn("device is already set up")
		return errors.New(ErrDeviceAlreadySetup)
	}

	// set the device settings on the host (via the daemon)
	err = c.saveSettings(ctx, logger, &dv1.SaveSettingsCommand{
		AdminPassword:  settings.AdminUser.Password,
		TimeZone:       settings.Timezone,
		EnableSsh:      settings.EnableSsh,
		TrustedSshKeys: settings.TrustedSshKeys,
	})
	if err != nil {
		return err
	}

	// get seed salt value from blueprint
	seed, err := getSaltValue(ctx)
	if err != nil {
		return err
	} else {
		// salt & hash the meat before you put in on the grill
		settings.AdminUser.Password = hashPassword(settings.AdminUser.Password, []byte(seed))
	}

	_, err = kvclient.Set(ctx, kvclient.DEFAULT_DEVICE_SETTINGS_KEY, settings)
	if err != nil {
		return errors.New(ErrFailedToCreateSettings)
	}

	return nil
}

func (c *controller) Login(ctx context.Context, username, password string) (string, error) {
	settings := &v1.DeviceSettings{}
	err := kvclient.Get(ctx, kvclient.DEFAULT_DEVICE_SETTINGS_KEY, settings)
	if err != nil {
		return "", errors.New(ErrFailedToGetSettings)
	}

	salt, err := getSaltValue(ctx)
	if err != nil {
		return "", errors.New(ErrFailedToGetSettings)
	}

	// Check if the password is correct. If not, return an error
	if hashPassword(password, []byte(salt)) != settings.AdminUser.Password {
		return "", errors.New("invalid username or password")
	}

	// TODO: forge token

	return "JWT_TOKEN", nil
}
