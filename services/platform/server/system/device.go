package system

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	dv1 "github.com/home-cloud-io/core/api/platform/daemon/v1"
	v1 "github.com/home-cloud-io/core/api/platform/server/v1"
	"github.com/home-cloud-io/core/services/platform/server/async"
	kvclient "github.com/home-cloud-io/core/services/platform/server/kv-client"

	"github.com/google/uuid"
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
		// GetComponentVersions returns all the versions of system components (server, daemon, etc.)
		GetComponentVersions(ctx context.Context, logger chassis.Logger) (*v1.GetComponentVersionsResponse, error)
		// GetDeviceLogs returns the logs from the daemon for the given seconds in the past
		GetDeviceLogs(ctx context.Context, logger chassis.Logger, sinceSeconds int64) ([]*dv1.Log, error)
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

func (c *controller) GetComponentVersions(ctx context.Context, logger chassis.Logger) (*v1.GetComponentVersionsResponse, error) {

	var (
		versions = []*dv1.ComponentVersion{}
	)

	k8sVersion, err := c.k8sclient.GetServerVersion(ctx)
	if err != nil {
		versions = append(versions, &dv1.ComponentVersion{
			Name:    "k8s",
			Domain:  "system",
			Version: err.Error(),
		})
	} else {
		versions = append(versions, &dv1.ComponentVersion{
			Name:    "k8s",
			Domain:  "system",
			Version: k8sVersion,
		})
	}

	imageVersions, err := c.k8sclient.CurrentImages(ctx)
	if err != nil {
		versions = append(versions, &dv1.ComponentVersion{
			Name:    "images",
			Domain:  "platform",
			Version: err.Error(),
		})
	} else {
		for _, image := range imageVersions {
			versions = append(versions, &dv1.ComponentVersion{
				Name:    componentFromImage(image.Image),
				Domain:  "platform",
				Version: image.Current,
			})
		}
	}

	logger.Info("requesting component versions from daemon")
	listener := async.RegisterListener(ctx, c.broadcaster, &async.ListenerOptions[*dv1.ComponentVersions]{
		Callback: func(event *dv1.ComponentVersions) (bool, error) {
			versions = append(versions, event.Components...)
			return true, nil
		},
		Timeout: 30 * time.Second,
	})
	err = com.Send(&dv1.ServerMessage{
		Message: &dv1.ServerMessage_RequestComponentVersionsCommand{
			RequestComponentVersionsCommand: &dv1.RequestComponentVersionsCommand{},
		},
	})
	if err != nil {
		versions = append(versions, &dv1.ComponentVersion{
			Name:    "daemon",
			Domain:  "platform",
			Version: err.Error(),
		}, &dv1.ComponentVersion{
			Name:    "nixos",
			Domain:  "system",
			Version: err.Error(),
		})
	} else {
		err = listener.Listen(ctx)
		if err != nil {
			versions = append(versions, &dv1.ComponentVersion{
				Name:    "daemon",
				Domain:  "platform",
				Version: err.Error(),
			}, &dv1.ComponentVersion{
				Name:    "nixos",
				Domain:  "system",
				Version: err.Error(),
			})
		}
		logger.Info("settings saved successfully")
	}

	// sort versions
	sort.Slice(versions, func(i, j int) bool {
		return versions[i].Name < versions[j].Name
	})

	return buildComponentVersionsResponse(logger, versions), nil
}

func (c *controller) GetDeviceLogs(ctx context.Context, logger chassis.Logger, sinceSeconds int64) ([]*dv1.Log, error) {

	var (
		logs      = []*dv1.Log{}
		requestId = uuid.New().String()
	)

	listener := async.RegisterListener(ctx, c.broadcaster, &async.ListenerOptions[*dv1.Logs]{
		Callback: func(event *dv1.Logs) (bool, error) {
			if event.RequestId == requestId {
				logs = event.Logs
				return true, nil
			}
			return false, nil
		},
		Timeout: 30 * time.Second,
	})
	err := com.Send(&dv1.ServerMessage{
		Message: &dv1.ServerMessage_RequestLogsCommand{
			RequestLogsCommand: &dv1.RequestLogsCommand{
				RequestId:    requestId,
				SinceSeconds: uint32(sinceSeconds),
			},
		},
	})
	if err != nil {
		logger.WithError(err).Error("failed to send logs request to daemon")
		return logs, err
	} else {
		err = listener.Listen(ctx)
		if err != nil {
			logger.WithError(err).Error("failed to receive logs from daemon")
			return logs, err
		}
	}

	return logs, nil
}

// HELPERS

func componentFromImage(image string) string {
	s := strings.Split(filepath.Base(image), "-")
	return s[len(s)-1]
}

func buildComponentVersionsResponse(logger chassis.Logger, versions []*dv1.ComponentVersion) *v1.GetComponentVersionsResponse {
	var (
		response = &v1.GetComponentVersionsResponse{
			Platform: []*dv1.ComponentVersion{},
			System:   []*dv1.ComponentVersion{},
		}
	)

	for _, v := range versions {
		switch v.Domain {
		case "platform":
			response.Platform = append(response.Platform, v)
		case "system":
			response.System = append(response.System, v)
		default:
			logger.WithField("component_version", v).Warn("unsupported component version received")
		}
	}

	return response
}
