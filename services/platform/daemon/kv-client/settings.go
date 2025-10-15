package kvclient

import (
	"context"
	"errors"

	sv1 "github.com/home-cloud-io/core/api/platform/server/v1"
)

const (
	ErrFailedToGetSettings = "failed to get device settings"
)

func Settings(ctx context.Context) (*sv1.DeviceSettings, error) {
	settings := &sv1.DeviceSettings{}
	err := Get(ctx, DEFAULT_DEVICE_SETTINGS_KEY, settings)
	if err != nil {
		return nil, errors.New(ErrFailedToGetSettings)
	}
	return settings, err
}
