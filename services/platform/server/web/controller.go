package web

import (
	"context"
	"errors"
	"net/http"

	v1 "github.com/home-cloud-io/core/api/platform/server/v1"
	kvv1 "github.com/steady-bytes/draft/api/core/registry/key_value/v1"
	kvv1Connect "github.com/steady-bytes/draft/api/core/registry/key_value/v1/v1connect"

	"connectrpc.com/connect"
	"github.com/steady-bytes/draft/pkg/chassis"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

type (
	Controller interface {
		kvv1Connect.KeyValueServiceClient

		IsDeviceSetup(ctx context.Context) (bool, error)
		InitializeDevice(ctx context.Context, settings *v1.DeviceSettings) (string, error)
	}

	controller struct {
		kvv1Connect.KeyValueServiceClient
	}
)

func NewController() Controller {
	return &controller{
		kvv1Connect.NewKeyValueServiceClient(http.DefaultClient, chassis.GetConfig().Entrypoint()),
	}
}

const (
	ErrDeviceAlreadySetup     = "device already setup"
	ErrFailedToCreateSettings = "failed to create settings"
	ErrFailedToSaveSettings   = "failed to save settings"
	ErrFailedToGetSettings    = "failed to get settings"

	DEFAULT_DEVICE_SETTINGS_KEY = "device"
)

// IsDeviceSetup checks if the device is already setup by checking if the DEFAULT_DEVICE_SETTINGS_KEY key exists in the key-value store
// with the default settings model
func (c *controller) IsDeviceSetup(ctx context.Context) (bool, error) {
	pb, _ := anypb.New(&v1.DeviceSettings{})

	// list is used to get all the `DeviceSettings` objects in the key-value store
	// it will not fail if the key does not exist like `Get` would
	val, err := c.KeyValueServiceClient.List(ctx, connect.NewRequest(&kvv1.ListRequest{Value: pb}))
	if err != nil {
		return false, errors.New(ErrFailedToGetSettings)
	}

	if len(val.Msg.GetValues()) < 1 {
		return false, nil
	} else {
		return true, nil
	}
}

// InitializeDevice initializes the device with the given settings. It first checks if the device is already setup
// Uses the user-provided password to set the password for the "admin" user on the device
// and save the remaining settings in the key-value store
func (c *controller) InitializeDevice(ctx context.Context, settings *v1.DeviceSettings) (string, error) {
	yes, err := c.IsDeviceSetup(ctx)
	if err != nil {
		return "", errors.New(ErrFailedToGetSettings)
	} else if yes {
		return "", errors.New(ErrDeviceAlreadySetup)
	}

	// TODO: set the password for the "admin" user on the device (call to daemon)

	msg, err := buildSetRequest(DEFAULT_DEVICE_SETTINGS_KEY, settings)
	if err != nil {
		return "", errors.New(ErrFailedToCreateSettings)
	}

	id, err := c.KeyValueServiceClient.Set(ctx, msg)
	if err != nil {
		return "", errors.New(ErrFailedToSaveSettings)
	}

	return id.Msg.Key, nil
}

// buildSetRequest is a utility function to create a set request for the key-value store
func buildSetRequest(key string, value proto.Message) (*connect.Request[kvv1.SetRequest], error) {
	// Create the setting object for the device
	pb, err := anypb.New(value)
	if err != nil {
		return nil, errors.New(ErrFailedToCreateSettings)
	}

	set := &kvv1.SetRequest{
		Key:   key,
		Value: pb,
	}

	return connect.NewRequest(set), nil
}
