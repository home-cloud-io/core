package versioning

import (
	"bufio"
	"context"
	"fmt"
	"io"
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
		cmd *exec.Cmd
		err error
	)

	f, err := os.Open(daemonNixFile)
	if err != nil {
		logger.WithError(err).Error("failed to read daemon nix file")
		return err
	}
	defer f.Close()

	// create temp file
	tmp, err := os.CreateTemp("", "default-*.nix")
	if err != nil {
		logger.WithError(err).Error("failed to create temp daemon nix file")
		return err
	}
	defer tmp.Close()

	// replace existing version with new version in def
	err = lineByLineReplace(f, tmp, def)
	if err != nil {
		logger.WithError(err).Error("failed to replace version")
		return err
	}

	// make sure the temp file was successfully written to
	err = tmp.Close()
	if err != nil {
		logger.WithError(err).Error("failed to write temp daemon nix file")
		return err
	}

	// close original file
	err = f.Close()
	if err != nil {
		logger.WithError(err).Error("failed to close existing daemon nix file")
		return err
	}

	// overwrite the original file with the temp file
	err = os.Rename(tmp.Name(), daemonNixFile)
	if err != nil {
		logger.WithError(err).Error("failied to overwrite daemon nix file")
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

func lineByLineReplace(r io.Reader, w io.Writer, def *v1.ChangeDaemonVersionCommand) error {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "version =") {
			line = fmt.Sprintf("  version = %s;", def.Version)
		}
		if strings.Contains(line, "vendorHash =") {
			line = fmt.Sprintf("  vendorHash = %s;", def.VendorHash)
		}
		if strings.Contains(line, "hash =") {
			line = fmt.Sprintf("    hash = %s;", def.SrcHash)
		}
		_, err := io.WriteString(w, line+"\n")
		if err != nil {
			return err
		}
	}
	return scanner.Err()
}

func GetOSVersionDiff(ctx context.Context, logger chassis.Logger) (string, error) {
	var (
		cmd    *exec.Cmd
		output string
		err    error
	)

	logger.Info("updating nix channel")
	cmd = exec.Command("nix-channel", "--update")
	err = execute.ExecuteCommand(ctx, cmd)
	if err != nil {
		logger.WithError(err).Error("failed to run `nix-channel --update`")
		return "", err
	}
	logger.Info("updating nix channel: DONE")

	logger.Info("building updated nixos")
	cmd = exec.Command("nixos-rebuild", "build")
	err = execute.ExecuteCommand(ctx, cmd)
	if err != nil {
		logger.WithError(err).Error("failed to run `nixos-rebuild build`")
		return "", err
	}
	logger.Info("building updated nixos: DONE")

	logger.Info("calculating system diff")
	cmd = exec.Command("nvd", "diff", "/run/current-system", "./result")
	output, err = execute.ExecuteCommandReturnStdout(ctx, cmd)
	if err != nil {
		logger.WithError(err).Error("failed to run `nvd diff /run/current-system ./result`")
		return "", err
	}
	logger.Info("calculating system diff: DONE")

	return output, nil
}

// NOTE: must call this after calling GetOSVersionDiff if you want to perform a channel update.
func InstallOSUpdate(ctx context.Context, logger chassis.Logger) error {
	var (
		cmd *exec.Cmd
		err error
	)

	logger.Info("building and switching to updated nixos")
	cmd = exec.Command("nixos-rebuild", "switch")
	err = execute.ExecuteCommand(ctx, cmd)
	if err != nil {
		logger.WithError(err).Error("failed to run `nixos-rebuild switch`")
		return err
	}
	logger.Info("building and switching to updated nixos: DONE")

	return nil
}
