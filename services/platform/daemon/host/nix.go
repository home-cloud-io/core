package host

import (
	"context"
	"os/exec"
	"sync"
	"syscall"

	"github.com/home-cloud-io/core/services/platform/daemon/execute"

	"github.com/steady-bytes/draft/pkg/chassis"
)

var (
	// osMutex makes sure we don't run nixos commands concurrently
	osMutex = sync.Mutex{}
)

func GetOSVersionDiff(ctx context.Context, logger chassis.Logger) (string, error) {
	osMutex.Lock()
	defer osMutex.Unlock()

	config := chassis.GetConfig()
	if config.Env() == "test" {
		logger.Info("mocking os version diff")
		return "fake os version diff", nil
	}

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
func RebuildAndSwitchOS(ctx context.Context, logger chassis.Logger) error {
	osMutex.Lock()
	defer osMutex.Unlock()

	config := chassis.GetConfig()
	if config.Env() == "test" {
		logger.Info("mocking nixos rebuild")
		return nil
	}

	var (
		cmd *exec.Cmd
		err error
	)

	logger.Info("building and switching to updated nixos")
	cmd = exec.Command("nixos-rebuild", "switch")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}
	err = execute.ExecuteCommand(ctx, cmd)
	if err != nil {
		logger.WithError(err).Error("failed to run `nixos-rebuild switch`")
		return err
	}
	logger.Info("nixos rebuild command completed")

	return nil
}
