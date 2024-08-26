package web

import (
	"fmt"
	"io"
	"net/http"

	v1 "github.com/home-cloud-io/core/api/platform/server/v1"
	kvv1Connect "github.com/steady-bytes/draft/api/core/registry/key_value/v1/v1connect"
	"github.com/steady-bytes/draft/pkg/chassis"
	"k8s.io/apimachinery/pkg/util/yaml"
)

type ()

const (
	APP_STORE_URL = "https://raw.githubusercontent.com/home-cloud-io/core/main/apps.yaml"
)

// NewStoreCache creates a new store cache that runs in the background
// keeping the app store up to date with the latest available apps
func NewStoreCache(logger chassis.Logger) {
	_ = kvv1Connect.NewKeyValueServiceClient(http.DefaultClient, chassis.GetConfig().Entrypoint())

	httpClient := &http.Client{}
	req, err := http.NewRequest(http.MethodGet, APP_STORE_URL, nil)
	if err != nil {
		logger.Error("failed to create request for apps.yaml")
		return
	}

	res, err := httpClient.Do(req)
	if err != nil {
		logger.Error("failed to get apps.yaml")
		return
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		logger.Error("failed to read apps.yaml")
		return
	}

	fmt.Println(string(body))

	// open the file and unmarshal if with the `AppStoreResponse` protobuf
	// then save it to the key-value store

	appStoreResponse := &v1.AppStoreResponse{}
	yaml.Unmarshal(body, appStoreResponse)
	if err != nil {
		logger.Error("failed to unmarshal apps.yaml")
		return
	}

	fmt.Println(appStoreResponse)
}
