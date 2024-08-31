package web

import (
	"context"
	"errors"
	"io"
	"net/http"
	"time"

	v1 "github.com/home-cloud-io/core/api/platform/server/v1"
	kvclient "github.com/home-cloud-io/core/services/platform/server/kv-client"
	kvv1Connect "github.com/steady-bytes/draft/api/core/registry/key_value/v1/v1connect"

	"github.com/steady-bytes/draft/pkg/chassis"
	"k8s.io/apimachinery/pkg/util/yaml"
)

type (
	StoreCache interface {
		Refresh()
	}
	storeCache struct {
		logger   chassis.Logger
		kvClient kvv1Connect.KeyValueServiceClient
	}
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
	APP_STORE_URL         = "https://apps.home-cloud.io/index.yaml"

	ErrFailedToPopulateAppStore = "failed to populate app store"

	storeCacheInterval = 60 * time.Hour
)

func NewStoreCache(logger chassis.Logger) StoreCache {
	return &storeCache{
		logger: logger,
	}
}

func (c *storeCache) Refresh() {
	c.kvClient = kvv1Connect.NewKeyValueServiceClient(http.DefaultClient, chassis.GetConfig().Entrypoint())
	for {
		err := c.refresh()
		if err != nil {
			c.logger.WithError(err).Error("failed to refresh app store cache")
		}
		time.Sleep(storeCacheInterval)
	}
}

// RefreshCache creates a new store cache that runs in the background
// keeping the app store up to date with the latest available apps
func (c *storeCache) refresh() error {
	httpClient := &http.Client{}
	req, err := http.NewRequest(http.MethodGet, APP_STORE_URL, nil)
	if err != nil {
		c.logger.Error("failed to create request for apps.yaml")
		return errors.New(ErrFailedToPopulateAppStore)
	}

	res, err := httpClient.Do(req)
	if err != nil {
		c.logger.Error("failed to get apps.yaml")
		return errors.New(ErrFailedToPopulateAppStore)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		c.logger.Error("failed to read apps.yaml")
		return errors.New(ErrFailedToPopulateAppStore)
	}

	// open the file and unmarshal it then save it to the key-value store
	appStoreResponse := &HelmIndex{}
	if err := yaml.Unmarshal(body, appStoreResponse); err != nil {
		c.logger.WithError(err).Error("failed to unmarshal apps.yaml")
		return errors.New(ErrFailedToPopulateAppStore)
	}

	entries := &v1.AppStoreEntries{
		ApiVersion: appStoreResponse.ApiVersion,
		Generated: appStoreResponse.Generated,
		Entries: map[string]*v1.Apps{},
	}
	for name, apps := range appStoreResponse.Entries {
		entries.Entries[name] = &v1.Apps{
			Apps: apps,
		}
	}

	// marshal to any pb
	setReq, err := kvclient.BuildSetRequest(kvclient.APP_STORE_ENTRIES_KEY, entries)
	if err != nil {
		c.logger.Error("failed to marshal app store entries")
		return errors.New(ErrFailedToPopulateAppStore)
	}

	// store in blueprint
	val, err := c.kvClient.Set(context.Background(), setReq)
	if err != nil {
		c.logger.Error("failed to save app store entries")
		return errors.New(ErrFailedToPopulateAppStore)
	}

	c.logger.WithField("app_store", val).Info("blueprint has been populated with app store entries")
	c.logger.Info("app store cache has been created")

	return nil
}
