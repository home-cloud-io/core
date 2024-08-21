package versioning

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

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

func GetOSVersionDiff(ctx context.Context, logger chassis.Logger) (string, error) {
	var (
		cmd    *exec.Cmd
		output string
		err    error
	)

	cmd = exec.Command("cat", "/etc/nixos/home-cloud/daemon/default.nix")
	_ = execute.ExecuteCommand(ctx, cmd)

	logger.Info("updating nix channel")
	cmd = exec.Command("/run/current-system/sw/bin/nix-channel", "--update")
	err = execute.ExecuteCommand(ctx, cmd)
	if err != nil {
		logger.WithError(err).Error("failed to run `nix-channel --update`")
		return "", err
	}
	logger.Info("updating nix channel: DONE")

	logger.Info("building updated nixos")
	cmd = exec.Command("/run/current-system/sw/bin/nixos-rebuild", "build")
	err = execute.ExecuteCommand(ctx, cmd)
	if err != nil {
		logger.WithError(err).Error("failed to run `nixos-rebuild build`")
		return "", err
	}
	logger.Info("building updated nixos: DONE")

	logger.Info("calculating system diff")
	cmd = exec.Command("/run/current-system/sw/bin/nvd", "diff", "/run/current-system", "./result")
	err = execute.ExecuteCommand(ctx, cmd)
	if err != nil {
		logger.WithError(err).Error("failed to run `nvd diff /run/current-system ./result`")
		return "", err
	}
	logger.Info("calculating system diff: DONE")

	return output, nil
}
