package installs

import (
	"context"

	"connectrpc.com/connect"
	"sigs.k8s.io/controller-runtime/pkg/log"

	v1 "github.com/home-cloud-io/core/api/crds/v1"
	dv1 "github.com/home-cloud-io/core/api/platform/daemon/v1"
	"github.com/home-cloud-io/core/cmd/operator/controller/daemon"
)

func reconcileSystem(ctx context.Context, r *InstallReconciler, install *v1.Install) error {
	l := log.FromContext(ctx)

	// skip if daemon or system is disabled
	if install.Spec.Daemon.Disable || install.Spec.Daemon.System.Disable {
		l.V(1).Info("daemon or system disabled: skipping reconcile")
		install.Status.Daemon.System = nil
		return nil
	}

	// only upgrade if not installed or the source/version is changed
	if install.Status.Daemon.System == nil ||
		install.Spec.Daemon.System.Source != install.Status.Daemon.System.Source ||
		install.Spec.Daemon.System.Version != install.Status.Daemon.System.Version {

		l.V(1).Info("reconciling system install")
		daemonClient := daemon.DaemonClient(install.Spec.Daemon.Address)

		// first check the spec version against the daemon since an upgrade may have broken the previous reconcile iteration
		// before status could be written and we want to avoid an infinite loop of triggering the same upgrade over and over
		versionResp, err := daemonClient.Version(ctx, connect.NewRequest(&dv1.VersionRequest{}))
		if err != nil {
			return err
		}
		if versionResp.Msg.Version == install.Spec.Daemon.System.Version &&
			versionResp.Msg.Source == install.Spec.Daemon.System.Source {
			l.V(1).Info("new system version already installed: updating status")
			install.Status.Daemon.System = &v1.SystemStatus{
				Source:  install.Spec.Daemon.System.Source,
				Version: install.Spec.Daemon.System.Version,
			}
			return nil
		}

		l.Info("upgrading system install")
		_, err = daemonClient.Upgrade(ctx, connect.NewRequest(&dv1.UpgradeRequest{
			Source:  install.Spec.Daemon.System.Source,
			Version: install.Spec.Daemon.System.Version,
		}))
		if err != nil {
			return err
		}
		l.Info("system upgrade complete")
		install.Status.Daemon.System = &v1.SystemStatus{
			Source:  install.Spec.Daemon.System.Source,
			Version: install.Spec.Daemon.System.Version,
		}
	}

	l.V(1).Info("unchanged system install: skipping reconcile")
	return nil
}
