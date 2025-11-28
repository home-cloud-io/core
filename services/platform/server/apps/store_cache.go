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
	"gopkg.in/yaml.v3"
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
	ErrFailedToPopulateAppStore = "failed to populate app store"

	storeCacheIntervalConfigKey = "server.apps.store_cache_update_interval_minutes"
	storeUrlConfigKey           = "server.apps.store_url"
)

// AppStoreCache creates a new store cache that runs in the background
// keeping the app store up to date with the latest available apps
func AppStoreCache(logger chassis.Logger) {
	config := chassis.GetConfig()
	config.SetDefault(storeCacheIntervalConfigKey, 15)
	config.SetDefault(storeUrlConfigKey, "https://apps.home-cloud.io/index.yaml")
	interval := config.GetInt(storeCacheIntervalConfigKey)
	logger.WithField("interval_minutes", interval).Info("setting app store cache update interval")
	for {
		err := refresh(logger)
		if err != nil {
			logger.WithError(err).Error("failed to refresh app store cache")
		}
		time.Sleep(time.Duration(interval) * time.Minute)
	}
}
func refresh(logger chassis.Logger) error {
	logger.Info("refreshing app store cache")
	ctx := context.Background()
	httpClient := &http.Client{}
	url := chassis.GetConfig().GetString(storeUrlConfigKey)
	req, err := http.NewRequest(http.MethodGet, url, nil)
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
	_, err = kvclient.Set(ctx, kvclient.APP_STORE_ENTRIES_KEY, entries)
	if err != nil {
		logger.Error("failed to save app store entries")
		return errors.New(ErrFailedToPopulateAppStore)
	}

	logger.Info("app store cache has been refreshed")
	return nil
}
