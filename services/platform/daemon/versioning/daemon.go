package versioning

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	v1 "github.com/home-cloud-io/core/api/platform/daemon/v1"
	"github.com/home-cloud-io/core/services/platform/daemon/execute"
	"github.com/steady-bytes/draft/pkg/chassis"
	"golang.org/x/mod/semver"
)

const (
	daemonNixFile = "/etc/nixos/home-cloud/daemon/default.nix"
)

var (
	// source: https://semver.org/#is-there-a-suggested-regular-expression-regex-to-check-a-semver-string
	semverRegex = regexp.MustCompile(`(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+([0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?`)
)

func GetDaemonVersion(logger chassis.Logger) (string, error) {
	f, err := os.Open(daemonNixFile)
	if err != nil {
		logger.WithError(err).Error("failed to read daemon nix file")
		return "", err
	}
	defer f.Close()

	var version string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "version =") {
			version = semverRegex.FindString(line)
			break
		}
	}

	if version == "" {
		return "", fmt.Errorf("failed to find daemon version")
	}
	version = "v" + version

	if !semver.IsValid(version) {
		return "", fmt.Errorf("invalid daemon version: %s", version)
	}

	return version, nil
}

// TODO-RC2: There's a bit of a race condition with this right now. If you call GetOSVersionDiff and then call this
// method you'll accidentally upgrade the entire OS with any changes that were pulled in from the `nix-channel --update`
// that was run during GetOSVersionDiff. This can be avoided by running `nix-channel --rollback` but will require some
// stateful logic which checks if a rollback is really needed. It's out of scope for RC1 but should be revisited later.
func ChangeDaemonVersion(ctx context.Context, logger chassis.Logger, def *v1.ChangeDaemonVersionCommand) error {
	var (
		cmd       *exec.Cmd
		err       error
		replacers = []replacer{
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

	err = lineByLineReplace(daemonNixFile, replacers)
	if err != nil {
		logger.WithError(err).Error("failed to replace version")
		return err
	}

	logger.Info("building nixos with new daemon version")
	cmd = exec.Command("nixos-rebuild", "switch")
	err = execute.ExecuteCommand(ctx, cmd)
	if err != nil {
		logger.WithError(err).Error("failed to run `nixos-rebuild switch`")
		return err
	}
	logger.Info("building nixos with new daemon version: DONE")

	return nil
}