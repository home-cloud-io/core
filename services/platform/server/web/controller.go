package web

import (
	"context"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"

	v1 "github.com/home-cloud-io/core/api/platform/server/v1"
	kvv1 "github.com/steady-bytes/draft/api/core/registry/key_value/v1"
	kvv1Connect "github.com/steady-bytes/draft/api/core/registry/key_value/v1/v1connect"

	"connectrpc.com/connect"
	"github.com/steady-bytes/draft/pkg/chassis"
	"google.golang.org/protobuf/types/known/anypb"
)

type (
	Controller interface {
		kvv1Connect.KeyValueServiceClient
		chassis.Logger

		IsDeviceSetup(ctx context.Context) (bool, error)
		InitializeDevice(ctx context.Context, settings *v1.DeviceSettings) (string, error)
		Login(ctx context.Context, username, password string) (string, error)
		GetAppsInStore(ctx context.Context) ([]*v1.App, error)
	}

	controller struct {
		kvv1Connect.KeyValueServiceClient
		chassis.Logger
	}
)

func NewController(logger chassis.Logger) Controller {
	return &controller{
		kvv1Connect.NewKeyValueServiceClient(http.DefaultClient, chassis.GetConfig().Entrypoint()),
		logger,
	}
}

const (
	ErrDeviceAlreadySetup     = "device already setup"
	ErrFailedToCreateSettings = "failed to create settings"
	ErrFailedToSaveSettings   = "failed to save settings"
	ErrFailedToGetSettings    = "failed to get settings"
	ErrFailedToGetApps        = "failed to get apps"

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

	// TODO: Get seed salt value from `blue_print`
	var seedVal *kvv1.Value
	seedLookup, err := buildGetRequest(SEED_KEY, seedVal)
	if err != nil {
		return "", errors.New(ErrFailedToGetSettings)
	}

	getRes, err := c.KeyValueServiceClient.Get(ctx, seedLookup)
	if err != nil {
		return "", errors.New(ErrFailedToGetSettings)
	}

	if err := getRes.Msg.GetValue().UnmarshalTo(seedVal); err != nil {
		return "", errors.New(ErrFailedToGetSettings)
	}

	// a little salt & hash before saving the password
	var (
		pwBytes        = []byte(settings.GetAdminUser().GetPassword())
		sha512Hasher   = sha512.New()
		hashedPassword = sha512Hasher.Sum(nil)
	)

	pwBytes = append(pwBytes, []byte(seedVal.GetData())...)
	sha512Hasher.Write(pwBytes)
	settings.AdminUser.Password = hex.EncodeToString(hashedPassword)

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

func (c *controller) Login(ctx context.Context, username, password string) (string, error) {
	settings := &v1.DeviceSettings{}
	req, err := buildGetRequest(DEFAULT_DEVICE_SETTINGS_KEY, settings)
	if err != nil {
		return "", errors.New(ErrFailedToGetSettings)
	}

	val, err := c.KeyValueServiceClient.Get(ctx, req)
	if err != nil {
		return "", errors.New(ErrFailedToGetSettings)
	}

	if val.Msg.GetValue() == nil {
		return "", errors.New(ErrFailedToGetSettings)
	}

	if err := val.Msg.GetValue().UnmarshalTo(settings); err != nil {
		return "", errors.New(ErrFailedToGetSettings)
	}

	if settings.AdminUser.Password != password || settings.AdminUser.Username != username {
		return "", errors.New("invalid username or password")
	}

	// TODO: forge token

	return "JWT_TOKEN", nil
}

func (c *controller) GetAppsInStore(ctx context.Context) ([]*v1.App, error) {
	var (
		logger   = c.WithField("method", "GetAppsInStore")
		err      error
		apps     []*v1.App
		appStore = &v1.AppStoreEntries{}
	)

	logger.Info("getting apps in store")

	req, err := buildGetRequest(APP_STORE_ENTRIES_KEY, &v1.App{})
	if err != nil {
		logger.Error("failed to build get request")
		return nil, errors.New(ErrFailedToGetApps)
	}

	res, err := c.KeyValueServiceClient.Get(ctx, req)
	if err != nil {
		logger.Error("failed to get apps")
		return nil, errors.New(ErrFailedToGetApps)
	}

	if res.Msg.GetValue() == nil {
		logger.Info("no apps in store, this may or may not be an error")
		return apps, nil
	}

	if err := res.Msg.GetValue().UnmarshalTo(appStore); err != nil {
		logger.Error("failed to unmarshal apps")
		return nil, errors.New(ErrFailedToGetApps)
	}

	// map to the `App` type

	for k, v := range appStore.Entries {
		fmt.Println("key: %s", "value: %s", k, v)
	}

	return apps, nil
}
