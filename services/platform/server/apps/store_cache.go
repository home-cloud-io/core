package apps

import (
	"context"
	"errors"
	"io"
	"net/http"
	"time"

	v1 "github.com/home-cloud-io/core/api/platform/server/v1"
	kvclient "github.com/home-cloud-io/core/services/platform/server/kv-client"

	"github.com/steady-bytes/draft/pkg/chassis"
	"k8s.io/apimachinery/pkg/util/yaml"
)

type (
	// HelmIndex represents the exact shape of a Helm repository index file: e.g. https://apps.home-cloud.io/index.yaml
	//
	// NOTE: This is a local type instead of a protobuf because protobufs cannot represent a map with an
	// array/slice as the value but that is needed to unmarshal the yaml from the Helm index.
	HelmIndex struct {
		ApiVersion string
		Generated  string
		Entries    map[string][]*v1.App
	}
)

const (
	APP_STORE_URL               = "https://apps.home-cloud.io/index.yaml"
	ErrFailedToPopulateAppStore = "failed to populate app store"
	storeCacheInterval          = 60 * time.Hour
)

// AppStoreCache creates a new store cache that runs in the background
// keeping the app store up to date with the latest available apps
func AppStoreCache(logger chassis.Logger) {
	for {
		err := refresh(logger)
		if err != nil {
			logger.WithError(err).Error("failed to refresh app store cache")
		}
		time.Sleep(storeCacheInterval)
	}
}
func refresh(logger chassis.Logger) error {
	ctx := context.Background()
	httpClient := &http.Client{}
	req, err := http.NewRequest(http.MethodGet, APP_STORE_URL, nil)
	if err != nil {
		logger.Error("failed to create request for apps.yaml")
		return errors.New(ErrFailedToPopulateAppStore)
	}

	res, err := httpClient.Do(req)
	if err != nil {
		logger.Error("failed to get apps.yaml")
		return errors.New(ErrFailedToPopulateAppStore)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		logger.Error("failed to read apps.yaml")
		return errors.New(ErrFailedToPopulateAppStore)
	}

	// open the file and unmarshal it then save it to the key-value store
	appStoreResponse := &HelmIndex{}
	if err := yaml.Unmarshal(body, appStoreResponse); err != nil {
		logger.WithError(err).Error("failed to unmarshal apps.yaml")
		return errors.New(ErrFailedToPopulateAppStore)
	}

	entries := &v1.AppStoreEntries{
		ApiVersion: appStoreResponse.ApiVersion,
		Generated:  appStoreResponse.Generated,
		Entries:    map[string]*v1.Apps{},
	}
	for name, apps := range appStoreResponse.Entries {
		entries.Entries[name] = &v1.Apps{
			Apps: apps,
		}
	}

	// store in blueprint
	val, err := kvclient.Set(ctx, kvclient.APP_STORE_ENTRIES_KEY, entries)
	if err != nil {
		logger.Error("failed to save app store entries")
		return errors.New(ErrFailedToPopulateAppStore)
	}

	logger.WithField("app_store", val).Info("blueprint has been populated with app store entries")
	logger.Info("app store cache has been created")

	return nil
}