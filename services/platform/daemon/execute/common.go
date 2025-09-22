package execute

import (
	"context"
	"os/exec"

	"github.com/steady-bytes/draft/pkg/chassis"
)

func Restart(ctx context.Context, logger chassis.Logger) {
	logger.Info("restart command")
	if chassis.GetConfig().Env() == "test" {
		logger.Info("mocking restart")
		return
	}
	err := ExecuteCommand(ctx, exec.Command("reboot", "now"))
	if err != nil {
		logger.WithError(err).Error("failed to execute restart command")
		// TODO: send error back to server
	}
}

func Shutdown(ctx context.Context, logger chassis.Logger) {
	logger.Info("shutdown command")
	if chassis.GetConfig().Env() == "test" {
		logger.Info("mocking shutdown")
		return
	}
	err := ExecuteCommand(ctx, exec.Command("shutdown", "now"))
	if err != nil {
		logger.WithError(err).Error("failed to execute shutdown command")
		// TODO: send error back to server
	}
}