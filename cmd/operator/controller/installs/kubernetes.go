package installs

import (
	"context"

	"connectrpc.com/connect"
	"sigs.k8s.io/controller-runtime/pkg/log"

	v1 "github.com/home-cloud-io/core/api/crds/v1"
	dv1 "github.com/home-cloud-io/core/api/platform/daemon/v1"
	"github.com/home-cloud-io/core/cmd/operator/controller/daemon"
)

func reconcileKubernetes(ctx context.Context, r *InstallReconciler, install *v1.Install) error {
	l := log.FromContext(ctx)

	// skip if daemon or kubernetes is disabled
	if install.Spec.Daemon.Disable || install.Spec.Daemon.Kubernetes.Disable {
		l.V(1).Info("daemon or kubernetes disabled: skipping reconcile")
		install.Status.Daemon.Kubernetes = nil
		return nil
	}

	// only upgrade if not installed or the version is changed
	if install.Status.Daemon.Kubernetes == nil ||
		install.Spec.Daemon.Kubernetes.Version != install.Status.Daemon.Kubernetes.Version {
		l.V(1).Info("reconciling kubernetes install")
		daemonClient := daemon.DaemonClient(install.Spec.Daemon.Address)

		// first check the spec version against the cluster since an upgrade may have broken the previous reconcile
		// call before status could be written and we want to avoid an infinite loop of triggering the same upgrade over and over
		version, err := r.DiscoveryClient.ServerVersion()
		if err != nil {
			return err
		}
		if version.GitVersion == install.Spec.Daemon.Kubernetes.Version {
			l.V(1).Info("kubernetes version already installed: updating status")
			install.Status.Daemon.Kubernetes = &v1.KubernetesStatus{
				Version: install.Spec.Daemon.Kubernetes.Version,
			}
			return nil
		}

		l.Info("upgrading kubernetes install")
		_, err = daemonClient.UpgradeKubernetes(ctx, connect.NewRequest(&dv1.UpgradeKubernetesRequest{
			Version: install.Spec.Daemon.Kubernetes.Version,
		}))
		if err != nil {
			return err
		}
		l.Info("kubernetes upgrade complete")
		install.Status.Daemon.Kubernetes = &v1.KubernetesStatus{
			Version: install.Spec.Daemon.Kubernetes.Version,
		}
	}

	l.V(1).Info("unchanged kubernetes install: skipping reconcile")
	return nil
}
