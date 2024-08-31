package web

import (
	"context"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"time"

	v1 "github.com/home-cloud-io/core/api/platform/server/v1"
	opv1 "github.com/home-cloud-io/core/services/platform/operator/api/v1"
	k8sclient "github.com/home-cloud-io/core/services/platform/server/k8s-client"
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
		InstallApp(ctx context.Context, logger chassis.Logger, request *v1.InstallAppRequest) error
		DeleteApp(ctx context.Context, logger chassis.Logger, request *v1.DeleteAppRequest) error
		UpdateApp(ctx context.Context, logger chassis.Logger, request *v1.UpdateAppRequest) error
	}

	controller struct {
		kvv1Connect.KeyValueServiceClient
		chassis.Logger
		k8sclient k8sclient.Client
	}
)

func NewController(logger chassis.Logger) Controller {
	return &controller{
		kvv1Connect.NewKeyValueServiceClient(http.DefaultClient, chassis.GetConfig().Entrypoint()),
		logger,
		k8sclient.NewClient(logger),
	}
}

const (
	ErrDeviceAlreadySetup     = "device already setup"
	ErrFailedToCreateSettings = "failed to create settings"
	ErrFailedToSaveSettings   = "failed to save settings"
	ErrFailedToGetSettings    = "failed to get settings"
	ErrFailedToGetApps        = "failed to get apps"

	ErrFailedToBuildSeedGetRequest = "failed to build get request for seed"
	ErrFailedToGetSeedValue        = "failed to get seed value"
	ErrFailedToUnmarshalSeedValue  = "failed to unmarshal seed value"

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
	seed, err := getSaltValue(ctx, c.KeyValueServiceClient)
	if err != nil {
		return "", err
	} else {
		// salt & hash the meat before you put in on the grill
		settings.AdminUser.Password = hashPassword(settings.AdminUser.Password, []byte(seed))
	}

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

func getSaltValue(ctx context.Context, c kvv1Connect.KeyValueServiceClient) (string, error) {
	seedVal := &kvv1.Value{}
	seedLookup, err := buildGetRequest(SEED_KEY, seedVal)
	if err != nil {
		return "", errors.New(ErrFailedToBuildSeedGetRequest)
	}

	getRes, err := c.Get(ctx, seedLookup)
	if err != nil {
		return "", errors.New(ErrFailedToGetSeedValue)
	}

	anypb := getRes.Msg.GetValue()
	if err := anypb.UnmarshalTo(seedVal); err != nil {
		return "", errors.New(ErrFailedToUnmarshalSeedValue)
	}

	return seedVal.GetData(), nil
}

func hashPassword(password string, salt []byte) string {
	// a little salt & hash before saving the password
	var (
		pwBytes        = []byte(password)
		sha512Hasher   = sha512.New()
		hashedPassword = sha512Hasher.Sum(nil)
	)

	pwBytes = append(pwBytes, []byte(salt)...)
	sha512Hasher.Write(pwBytes)

	return hex.EncodeToString(hashedPassword)
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

	salt, err := getSaltValue(ctx, c.KeyValueServiceClient)
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

func (c *controller) GetAppsInStore(ctx context.Context) ([]*v1.App, error) {
	var (
		logger   = c.WithField("method", "GetAppsInStore")
		err      error
		apps     []*v1.App
		appStore = &v1.AppStoreEntries{}
	)

	logger.Info("getting apps in store")

	req, err := buildGetRequest(APP_STORE_ENTRIES_KEY, appStore)
	if err != nil {
		logger.WithError(err).Error("failed to build get request")
		return nil, errors.New(ErrFailedToGetApps)
	}

	res, err := c.KeyValueServiceClient.Get(ctx, req)
	if err != nil {
		logger.WithError(err).Error("failed to get apps")
		return nil, errors.New(ErrFailedToGetApps)
	}

	if res.Msg.GetValue() == nil {
		logger.Info("no apps in store, this may or may not be an error")
		return apps, nil
	}

	if err := res.Msg.GetValue().UnmarshalTo(appStore); err != nil {
		logger.WithError(err).Error("failed to unmarshal apps")
		return nil, errors.New(ErrFailedToGetApps)
	}

	for _, v := range appStore.Entries {
		// TODO: get the latest, right now this assumes the `app` slice is already sorted by version
		// append the first app of the app store entry to to the `apps` slice
		if len(v.Apps) > 0 {
			apps = append(apps, v.Apps[0])
		}
	}

	return apps, nil
}

func (c *controller) InstallApp(ctx context.Context, logger chassis.Logger, request *v1.InstallAppRequest) error {
	// check dependencies for app from the store and install if needed
	apps, err := c.GetAppsInStore(ctx)
	if err != nil {
		return err
	}
	for _, app := range apps {
		if request.Chart == app.Name {
			for _, dep := range app.Dependencies {
				log := logger.WithField("dependency", dep.Name)
				log.Info("checking dependency")
				installed, err := c.k8sclient.AppInstalled(ctx, dep.Name)
				if err != nil {
					log.WithError(err).Error("failed to check if dependency is installed")
					return err
				}
				if !installed {
					log.Info("dependency is needed: installing")
					err := c.k8sclient.Install(ctx, opv1.AppSpec{
						Chart:   dep.Name,
						Repo:    strings.TrimPrefix(dep.Repository, "https://"),
						Release: dep.Name,
						Version: dep.Version,
					})
					if err != nil {
						log.WithError(err).Error("failed to install app")
						return err
					}

					// wait on dependency install
					timeCtx, cancel := context.WithTimeout(ctx, 3*time.Minute)
					err = c.waitForInstall(timeCtx, log, dep.Name)
					cancel()
					if err != nil {
						log.WithError(err).Error("failed to wait for install")
						return err
					}
				}
			}
		}
	}

	// install requested app
	logger.Info("installing requested app")
	err = c.k8sclient.Install(ctx, opv1.AppSpec{
		Chart:   request.Chart,
		Repo:    request.Repo,
		Release: request.Release,
		Values:  request.Values,
		Version: request.Version,
	})
	if err != nil {
		logger.WithError(err).Error("failed to install app")
		return err
	}

	return nil
}

func (c *controller) DeleteApp(ctx context.Context, logger chassis.Logger, request *v1.DeleteAppRequest) error {
	err := c.k8sclient.Delete(ctx, opv1.AppSpec{
		Release: request.Release,
	})
	if err != nil {
		return err
	}
	return nil
}

func (c *controller) UpdateApp(ctx context.Context, logger chassis.Logger, request *v1.UpdateAppRequest) error {
	err := c.k8sclient.Update(ctx, opv1.AppSpec{
		Chart:   request.Chart,
		Repo:    request.Repo,
		Release: request.Release,
		Values:  request.Values,
		Version: request.Version,
	})
	if err != nil {
		return err
	}
	return nil
}

func (c *controller) waitForInstall(ctx context.Context, logger chassis.Logger, appName string) error {
	for {
		if ctx.Err() != nil {
			logger.WithError(ctx.Err()).Error("context is done")
			return ctx.Err()
		}
		appsHealth, err := c.k8sclient.CheckAppsHealth(ctx)
		if err != nil {
			logger.WithError(err).Error("failed to check apps health")
			return err
		}
		for _, app := range appsHealth {
			if app.Name == appName {
				if app.Status == v1.AppStatus_APP_STATUS_HEALTHY {
					logger.Info("installation completed")
					return nil
				}
				break
			}
		}
		logger.Info("not yet installed")

		time.Sleep(5 * time.Second)
	}
}
