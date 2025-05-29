package host

import (
	"context"
	"strings"

	v1 "github.com/home-cloud-io/core/api/platform/daemon/v1"

	"github.com/steady-bytes/draft/pkg/chassis"
)

func SetSystemImage(ctx context.Context, logger chassis.Logger, def *v1.SetSystemImageCommand) error {
	var (
		replacers = []Replacer{
			func(line ReplacerLine) string {
				return strings.ReplaceAll(line.Current, def.CurrentImage, def.RequestedImage)
			},
		}
		systemKubernetesManifests = []string{
			DraftManifestFile(),
			OperatorManifestFile(),
			ServerManifestFile(),
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
