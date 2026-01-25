package apps

import (
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/steady-bytes/draft/pkg/chassis"
	"gopkg.in/yaml.v3"

	v1 "github.com/home-cloud-io/core/api/platform/server/v1"
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

func (c *controller) AppStoreCache(logger chassis.Logger) {
	config := chassis.GetConfig()
	config.SetDefault(storeCacheIntervalConfigKey, 15)
	config.SetDefault(storeUrlConfigKey, "https://apps.home-cloud.io/index.yaml")
	interval := config.GetInt(storeCacheIntervalConfigKey)
	logger.WithField("interval_minutes", interval).Info("setting app store cache update interval")
	for {
		err := c.refresh(logger)
		if err != nil {
			logger.WithError(err).Error("failed to refresh app store cache")
		}
		time.Sleep(time.Duration(interval) * time.Minute)
	}
}
func (c *controller) refresh(logger chassis.Logger) error {
	logger.Info("refreshing app store cache")
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

	// cache in memory
	c.storeCache = entries

	logger.Info("app store cache has been refreshed")
	return nil
}
