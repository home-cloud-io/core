package host

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"

	v1 "github.com/home-cloud-io/core/api/platform/daemon/v1"

	"github.com/steady-bytes/draft/pkg/chassis"
	"golang.org/x/mod/semver"
)

const (
	versionPrefix    = "  version = \""
	vendorHashPrefix = "  vendorHash = \""
	srcHashPrefix    = "    hash = \""
	nixLineSuffix    = "\";"
)

var (
	// source: https://semver.org/#is-there-a-suggested-regular-expression-regex-to-check-a-semver-string
	semverRegex = regexp.MustCompile(`v(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+([0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?`)
)

func GetDaemonVersion(logger chassis.Logger) (*v1.CurrentDaemonVersion, error) {
	f, err := os.Open(DaemonNixFile)
	if err != nil {
		logger.WithError(err).Error("failed to read daemon nix file")
		return nil, err
	}
	defer f.Close()

	current := &v1.CurrentDaemonVersion{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, versionPrefix) {
			current.Version = semverRegex.FindString(line)
			continue
		}
		if strings.HasPrefix(line, vendorHashPrefix) {
			current.VendorHash = strings.TrimSuffix(strings.TrimPrefix(line, vendorHashPrefix), nixLineSuffix)
			continue
		}
		if strings.HasPrefix(line, srcHashPrefix) {
			current.SrcHash = strings.TrimSuffix(strings.TrimPrefix(line, srcHashPrefix), nixLineSuffix)
			continue
		}
	}

	if !semver.IsValid(current.Version) {
		return nil, fmt.Errorf("invalid daemon version: %s", current.Version)
	}

	return current, nil
}

// TODO-RC2: There's a bit of a race condition with this right now. If you call GetOSVersionDiff and then call this
// method you'll accidentally upgrade the entire OS with any changes that were pulled in from the `nix-channel --update`
// that was run during GetOSVersionDiff. This can be avoided by running `nix-channel --rollback` but will require some
// stateful logic which checks if a rollback is really needed. It's out of scope for RC1 but should be revisited later.
func ChangeDaemonVersion(ctx context.Context, logger chassis.Logger, def *v1.ChangeDaemonVersionCommand) error {
	var (
		err       error
		replacers = []Replacer{
			func(line string) string {
				if strings.Contains(line, "version =") {
					line = fmt.Sprintf("  version = \"%s\";", def.Version)
				}
				return line
			},
			func(line string) string {
				if strings.Contains(line, "vendorHash =") {
					line = fmt.Sprintf("  vendorHash = \"%s\";", def.VendorHash)
				}
				return line
			},
			func(line string) string {
				if strings.Contains(line, "hash =") {
					// NOTE: the double indentation is deliberate
					line = fmt.Sprintf("    hash = \"%s\";", def.SrcHash)
				}
				return line
			},
		}
	)

	err = LineByLineReplace(DaemonNixFile, replacers)
	if err != nil {
		logger.WithError(err).Error("failed to replace version")
		return err
	}

	err = RebuildAndSwitchOS(ctx, logger)
	if err != nil {
		return err
	}

	return nil
}
