package system

import (
	"context"
	"fmt"
	"net/http"

	dv1connect "github.com/home-cloud-io/core/api/platform/daemon/v1/v1connect"
	v1 "github.com/home-cloud-io/core/api/platform/server/v1"
	k8sclient "github.com/home-cloud-io/core/services/platform/server/k8s-client"

	"github.com/containers/image/v5/docker"
	"github.com/steady-bytes/draft/pkg/chassis"
	"golang.org/x/mod/semver"
)

type (
	Controller interface {
		Containers
		Daemon
		Device
		Locators
		OS
		Peering
	}

	controller struct {
		k8sclient    k8sclient.System
		daemonClient dv1connect.DaemonServiceClient
	}
)

func NewController(logger chassis.Logger) Controller {
	config := chassis.GetConfig()
	config.SetDefault(osAutoUpdateCronConfigKey, "0 1 * * *")
	config.SetDefault(containerAutoUpdateCronConfigKey, "0 2 * * *")
	return &controller{
		k8sclient: k8sclient.NewClient(logger),
		// TODO: derive this address? get it from blueprint?
		daemonClient: dv1connect.NewDaemonServiceClient(http.DefaultClient, "daemon.home-cloud-system"),
	}
}

const (
	ErrFailedToBuildSeedGetRequest = "failed to build get request for seed"

	osAutoUpdateCronConfigKey        = "server.updates.os_auto_update_cron"
	containerAutoUpdateCronConfigKey = "server.updates.containers_auto_update_cron"

	// Currently only a single interface is supported and defaults to this value. In the future we
	// will probably want to support multiple interfaces (e.g. one for trusted mobile clients and another for federated servers)
	DefaultWireguardInterface = "wg0"
	// TODO: make this configurable
	DefaultSTUNServerAddress = "locator1.home-cloud.io:3478"
)

// helper functions

func getLatestImageTags(ctx context.Context, images []*v1.ImageVersion) ([]*v1.ImageVersion, error) {
	for _, image := range images {
		latest, err := getLatestImageTag(ctx, image.Image)
		if err != nil {
			return nil, err
		}
		image.Latest = latest
	}
	return images, nil
}

func getLatestImageTag(ctx context.Context, image string) (string, error) {
	ref, err := docker.ParseReference(fmt.Sprintf("//%s", image))
	if err != nil {
		return "", err
	}

	tags, err := docker.GetRepositoryTags(ctx, nil, ref)
	if err != nil {
		return "", err
	}

	semverTags := []string{}
	for _, t := range tags {
		if !semver.IsValid(t) {
			continue
		}
		semverTags = append(semverTags, t)
	}

	var latestVersion string
	if len(semverTags) > 0 {
		semver.Sort(semverTags)
		latestVersion = semverTags[len(semverTags)-1]
	}

	return latestVersion, nil
}
