package system

import (
	"context"
	"path/filepath"
	"sort"
	"strings"

	dv1 "github.com/home-cloud-io/core/api/platform/daemon/v1"
	v1 "github.com/home-cloud-io/core/api/platform/server/v1"

	"github.com/steady-bytes/draft/pkg/chassis"
)

type (
	Device interface {
		// GetServerSettings returns the current server settings after filtering out the
		// admin username and password.
		GetServerSettings(ctx context.Context) (*v1.DeviceSettings, error)
		// SetServerSettings updates the settings on the server with the given values
		SetServerSettings(ctx context.Context, logger chassis.Logger, settings *v1.DeviceSettings) error
		// GetComponentVersions returns all the versions of system components (server, daemon, etc.)
		GetComponentVersions(ctx context.Context, logger chassis.Logger) (*v1.GetComponentVersionsResponse, error)
	}
)

const (
	ErrDeviceAlreadySetup = "device already setup"

	ErrFailedToCreateSettings = "failed to create device settings"
	ErrFailedToGetSettings    = "failed to get device settings"
	ErrFailedToSetSettings    = "failed to save device settings"
)

// DEVICE

func (c *controller) GetServerSettings(ctx context.Context) (*v1.DeviceSettings, error) {
	settings, err := c.k8sclient.Settings(ctx)
	if err != nil {
		return nil, err
	}
	return &v1.DeviceSettings{
		AutoUpdateApps: settings.AutoUpdateApps,
		AutoUpdateOs:   settings.AutoUpdateSystem,
	}, nil
}

func (c *controller) SetServerSettings(ctx context.Context, logger chassis.Logger, settings *v1.DeviceSettings) error {
	// TODO
	return nil
}

// TODO: need to redo this using opv1.Install.Status
func (c *controller) GetComponentVersions(ctx context.Context, logger chassis.Logger) (*v1.GetComponentVersionsResponse, error) {

	var (
		versions = []*dv1.ComponentVersion{}
	)

	k8sVersion, err := c.k8sclient.GetServerVersion(ctx)
	if err != nil {
		versions = append(versions, &dv1.ComponentVersion{
			Name:    "k8s",
			Domain:  "system",
			Version: err.Error(),
		})
	} else {
		versions = append(versions, &dv1.ComponentVersion{
			Name:    "k8s",
			Domain:  "system",
			Version: k8sVersion,
		})
	}

	// TODO: does this get all the new images?
	imageVersions, err := c.k8sclient.CurrentImages(ctx)
	if err != nil {
		versions = append(versions, &dv1.ComponentVersion{
			Name:    "images",
			Domain:  "platform",
			Version: err.Error(),
		})
	} else {
		for _, image := range imageVersions {
			versions = append(versions, &dv1.ComponentVersion{
				Name:    componentFromImage(image.Image),
				Domain:  "platform",
				Version: image.Current,
			})
		}
	}

	// sort versions
	sort.Slice(versions, func(i, j int) bool {
		return versions[i].Name < versions[j].Name
	})

	return buildComponentVersionsResponse(logger, versions), nil
}

// HELPERS

func componentFromImage(image string) string {
	s := strings.Split(filepath.Base(image), "-")
	return s[len(s)-1]
}

func buildComponentVersionsResponse(logger chassis.Logger, versions []*dv1.ComponentVersion) *v1.GetComponentVersionsResponse {
	var (
		response = &v1.GetComponentVersionsResponse{
			Platform: []*dv1.ComponentVersion{},
			System:   []*dv1.ComponentVersion{},
		}
	)

	for _, v := range versions {
		switch v.Domain {
		case "platform":
			response.Platform = append(response.Platform, v)
		case "system":
			response.System = append(response.System, v)
		default:
			logger.WithField("component_version", v).Warn("unsupported component version received")
		}
	}

	return response
}
