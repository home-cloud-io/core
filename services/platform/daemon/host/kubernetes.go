package host

import (
	"context"
	"strings"

	v1 "github.com/home-cloud-io/core/api/platform/daemon/v1"

	"github.com/steady-bytes/draft/pkg/chassis"
)

var (
	systemKubernetesManifests = []string{
		DraftManifestFile,
		OperatorManifestFile,
		ServerManifestFile,
	}
)

func SetSystemImage(ctx context.Context, logger chassis.Logger, def *v1.SetSystemImageCommand) error {
	var (
		replacers = []Replacer{
			func(line string) string {
				return strings.ReplaceAll(line, def.CurrentImage, def.RequestedImage)
			},
		}
	)

	for _, filename := range systemKubernetesManifests {
		err := LineByLineReplace(filename, replacers)
		if err != nil {
			return err
		}
	}

	return nil
}
