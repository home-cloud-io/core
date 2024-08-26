package versioning

import (
	"context"
	"strings"

	v1 "github.com/home-cloud-io/core/api/platform/daemon/v1"

	"github.com/steady-bytes/draft/pkg/chassis"
)

var (
	systemKubernetesManifests = []string{
		"/var/lib/rancher/k3s/server/manifests/draft.yaml",
		"/var/lib/rancher/k3s/server/manifests/mdns.yaml",
		"/var/lib/rancher/k3s/server/manifests/operator.yaml",
		"/var/lib/rancher/k3s/server/manifests/server.yaml",
	}
)

func SetSystemImage(ctx context.Context, logger chassis.Logger, def *v1.SetSystemImageCommand) error {
	var (
		replacers = []replacer{
			func(line string) string {
				return strings.ReplaceAll(line, def.CurrentImage, def.RequestedImage)
			},
		}
	)

	for _, filename := range systemKubernetesManifests {
		err := lineByLineReplace(filename, replacers)
		if err != nil {
			return err
		}
	}

	return nil
}
