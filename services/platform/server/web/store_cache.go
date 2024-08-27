package web

import (
	"context"
	"errors"
	"io"
	"net/http"

	v1 "github.com/home-cloud-io/core/api/platform/server/v1"
	kvv1Connect "github.com/steady-bytes/draft/api/core/registry/key_value/v1/v1connect"
	"github.com/steady-bytes/draft/pkg/chassis"
	"k8s.io/apimachinery/pkg/util/yaml"
)

const (
	APP_STORE_URL         = "https://home-cloud-io.github.io/store/index.yaml"
	APP_STORE_ENTRIES_KEY = "app_store_entries"

	ErrFailedToPopulateAppStore = "failed to populate app store"
)

// NewStoreCache creates a new store cache that runs in the background
// keeping the app store up to date with the latest available apps
func NewStoreCache(logger chassis.Logger) error {
	kvClient := kvv1Connect.NewKeyValueServiceClient(http.DefaultClient, chassis.GetConfig().Entrypoint())

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

	// open the file and unmarshal if with the `AppStoreResponse` protobuf
	// then save it to the key-value store

	appStoreResponse := make(map[string]interface{})
	if err := yaml.Unmarshal(body, &appStoreResponse); err != nil {
		logger.Error("failed to unmarshal apps.yaml")
		return errors.New(ErrFailedToPopulateAppStore)
	}

	storeApps := mapToAppStoreResponse(appStoreResponse)

	// Insert the app store response into the key-value store so it can be used later

	// marshal to any pb
	setReq, err := buildSetRequest(APP_STORE_ENTRIES_KEY, storeApps)
	if err != nil {
		logger.Error("failed to marshal app store entries")
		return errors.New(ErrFailedToPopulateAppStore)
	}

	// store in blueprint
	val, err := kvClient.Set(context.Background(), setReq)
	if err != nil {
		logger.Error("failed to save app store entries")
		return errors.New(ErrFailedToPopulateAppStore)
	}

	logger.WithField("app_store", val).Info("blueprint has been populated with app store entries")
	logger.Info("app store cache has been created")

	return nil
}

// mapToAppStoreResponse maps the yaml file to the AppStoreResponse protobuf type.
// TODO: Make this work with the protobuf type so that `yaml.Unmarshal` can be used. I ran into some issues
// with maps.
func mapToAppStoreResponse(m map[string]interface{}) *v1.AppStoreEntries {
	var (
		data = &v1.AppStoreEntries{
			Entries: make(map[string]*v1.Entries),
		}
	)

	for k, v := range m {
		switch k {
		case "apiVersion":
			data.ApiVersion = v.(string)
		case "generated":
			data.Generated = v.(string)
		case "entries":
			for appKey, v := range v.(map[string]interface{}) {
				apps := []*v1.App{}
				for _, v := range v.([]interface{}) {
					app := &v1.App{}

					for k, v := range v.(map[string]interface{}) {
						switch k {
						case "appVersion":
							app.AppVersion = v.(string)
						case "created":
							app.CreatedAt = v.(string)
						case "name":
							app.Name = v.(string)
						case "version":
							app.Version = v.(string)
						case "description":
							app.Description = v.(string)
						case "icon":
							app.Icon = v.(string)
						case "urls":
							urls := []string{}
							for _, v := range v.([]interface{}) {
								urls = append(app.Urls, v.(string))
							}

							app.Urls = urls

						case "digest":
							app.Digest = v.(string)
						case "type":
							app.Type = v.(string)
						}
					}

					apps = append(apps, app)
				}

				data.Entries[appKey] = &v1.Entries{
					Apps: apps,
				}

				apps = nil
			}
		}
	}

	return data
}
