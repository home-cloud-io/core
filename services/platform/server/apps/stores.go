package apps

import (
	"context"
	"errors"
	"io"
	"net/http"

	"github.com/steady-bytes/draft/pkg/chassis"
	"gopkg.in/yaml.v3"

	v1 "github.com/home-cloud-io/core/api/platform/server/v1"
	opv1 "github.com/home-cloud-io/core/services/platform/operator/api/v1"
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

	DefaultAppStoreURL         = "https://apps.home-cloud.io/index.yaml"
	DefaultAppStoreRawChartURL = "https://raw.githubusercontent.com/home-cloud-io/store"
)

func (c *controller) stores(ctx context.Context, logger chassis.Logger) ([]*v1.AppStoreEntries, error) {
	results := []*v1.AppStoreEntries{}

	settings, err := c.k8sclient.Settings(ctx)
	if err != nil {
		logger.WithError(err).Error("failed to get settings")
		return nil, err
	}

	if settings.AppStores == nil || len(settings.AppStores) == 0 {
		settings.AppStores = []opv1.AppStore{
			{
				URL:         DefaultAppStoreURL,
				RawChartURL: DefaultAppStoreRawChartURL,
			},
		}
	}

	for _, store := range settings.AppStores {
		l := logger.WithField("app_store", store.URL)
		url := store.URL

		httpClient := &http.Client{}
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			l.WithError(err).Error("failed to create request to app store")
			return results, errors.New(ErrFailedToPopulateAppStore)
		}

		res, err := httpClient.Do(req)
		if err != nil {
			l.WithError(err).Error("failed to get entries from app store")
			return results, errors.New(ErrFailedToPopulateAppStore)
		}
		defer res.Body.Close()

		body, err := io.ReadAll(res.Body)
		if err != nil {
			l.WithError(err).Error("failed to read entries from app store")
			return results, errors.New(ErrFailedToPopulateAppStore)
		}

		appStoreResponse := &HelmIndex{}
		if err := yaml.Unmarshal(body, appStoreResponse); err != nil {
			l.WithError(err).WithError(err).Error("failed to unmarshal entries from app store")
			return results, errors.New(ErrFailedToPopulateAppStore)
		}

		entries := &v1.AppStoreEntries{
			ApiVersion:  appStoreResponse.ApiVersion,
			Generated:   appStoreResponse.Generated,
			RawChartUrl: store.RawChartURL,
			Entries:     map[string]*v1.Apps{},
		}
		for name, apps := range appStoreResponse.Entries {
			entries.Entries[name] = &v1.Apps{
				Apps: apps,
			}
		}

		results = append(results, entries)
	}

	return results, nil
}
