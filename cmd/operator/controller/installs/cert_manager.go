package installs

import (
	"context"
	"time"

	"helm.sh/helm/v3/pkg/action"
	"sigs.k8s.io/controller-runtime/pkg/log"

	v1 "github.com/home-cloud-io/core/api/crds/v1"
	"github.com/home-cloud-io/core/pkg/helm"
	"github.com/home-cloud-io/core/pkg/install/resources"
	"github.com/home-cloud-io/core/pkg/kube"
)

func reconcileCertManager(ctx context.Context, r *InstallReconciler, install *v1.Install) error {
	l := log.FromContext(ctx)

	if !install.Spec.CertManager.Disable {
		// NOTE: we can't simply skip an install if the version hasn't changed since the values
		// might have changed with no version bump

		l.Info("reconciling cert-manager install")
		err := helmCertManager(ctx, install)
		if err != nil {
			return err
		}

		l.Info("reconciling certificates")
		err = kube.InstallResources(ctx, r.Client, resources.Certificates(install))
		if err != nil {
			return err
		}

		install.Status.CertManager = &v1.CertManagerStatus{
			Source:  install.Spec.CertManager.Source,
			Version: install.Spec.CertManager.Version,
		}
	} else {
		// only try and uninstall if currently installed
		if install.Status.CertManager != nil {

			// TODO: remove certificates? I think this should only be on a force/clean uninstall option

			l.Info("cert-manager is disabled: removing previous installation")
			err := uninstallCertManager(ctx, install)
			if err != nil {
				return err
			}
		}
		install.Status.CertManager = nil
	}

	return nil
}

func helmCertManager(ctx context.Context, install *v1.Install) error {

	cfg, err := helm.ActionConfiguration(install.Spec.CertManager.Namespace)
	if err != nil {
		return err
	}
	iAct := action.NewInstall(cfg)
	iAct.ReleaseName = "cert-manager"
	iAct.Version = install.Spec.CertManager.Version
	iAct.Namespace = install.Spec.CertManager.Namespace
	iAct.RepoURL = install.Spec.CertManager.Source
	iAct.Wait = true
	iAct.Timeout = 5 * time.Minute

	uAct := action.NewUpgrade(cfg)
	uAct.Version = install.Spec.CertManager.Version
	uAct.Namespace = install.Spec.CertManager.Namespace
	uAct.RepoURL = install.Spec.CertManager.Source
	uAct.Wait = true
	uAct.Timeout = 5 * time.Minute

	values, err := helm.UnmarshalValues(install.Spec.Istio.CNI.Values)
	if err != nil {
		return err
	}
	err = helm.InstallOrUpgrade(ctx, cfg, iAct, uAct, values)
	if err != nil {
		return err
	}

	return nil
}

func uninstallCertManager(_ context.Context, install *v1.Install) error {
	actionConfiguration, err := helm.ActionConfiguration(install.Spec.CertManager.Namespace)
	if err != nil {
		return err
	}

	act := action.NewUninstall(actionConfiguration)
	act.IgnoreNotFound = true
	act.Wait = true
	act.Timeout = 5 * time.Minute

	_, err = act.Run("cert-manager")
	if err != nil {
		return err
	}

	return nil
}
