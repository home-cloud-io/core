package execute

import (
	"context"

	"github.com/steady-bytes/draft/pkg/chassis"
)

// TODO: need to wire up talosctl access
// https://docs.siderolabs.com/kubernetes-guides/advanced-guides/talos-api-access-from-k8s

func Reboot(ctx context.Context, logger chassis.Logger) error {
	logger.Info("reboot command")
	if chassis.GetConfig().Env() == "test" {
		logger.Info("mocking reboot")
		return nil
	}
	// TODO: implement with talos
	// TODO: option to use "powercycle" mode?
	return nil
}

func Shutdown(ctx context.Context, logger chassis.Logger) error {
	logger.Info("shutdown command")
	if chassis.GetConfig().Env() == "test" {
		logger.Info("mocking shutdown")
		return nil
	}
	// TODO: implement with talos
	// TODO: option to force after timeout?
	return nil
}
