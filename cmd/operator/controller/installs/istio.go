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

func reconcileIstio(ctx context.Context, r *InstallReconciler, install *v1.Install) error {
	l := log.FromContext(ctx)

	if !install.Spec.Istio.Disable {
		// NOTE: we can't simply skip an install if the version hasn't changed since the values
		// might have changed with no version bump

		l.Info("reconciling ingress gateway")
		err := kube.InstallResources(ctx, r.Client, resources.GatewayObjects(install))
		if err != nil {
			return err
		}

		l.Info("reconciling istio install")
		err = helmIstio(ctx, install)
		if err != nil {
			return err
		}

		install.Status.Istio = &v1.IstioStatus{
			Source:  install.Spec.Istio.Source,
			Version: install.Spec.Istio.Version,
		}
	} else {
		// only try and uninstall if currently installesd
		if install.Status.Istio != nil {
			l.Info("istio is disabled: removing ingress gateway")
			err := kube.UninstallResources(ctx, r.Client, resources.GatewayObjects(install))
			if err != nil {
				return err
			}

			l.Info("istio is disabled: removing previous installation")
			err = uninstallIstio(ctx, install)
			if err != nil {
				return err
			}
		}
		install.Status.Istio = nil
	}

	return nil
}

func helmIstio(ctx context.Context, install *v1.Install) error {

	cfg, err := helm.ActionConfiguration(install.Spec.Istio.Namespace)
	if err != nil {
		return err
	}
	iAct := action.NewInstall(cfg)
	iAct.Version = install.Spec.Istio.Version
	iAct.Namespace = install.Spec.Istio.Namespace
	iAct.RepoURL = install.Spec.Istio.Source
	iAct.Wait = true
	iAct.Timeout = 5 * time.Minute

	uAct := action.NewUpgrade(cfg)
	uAct.Version = install.Spec.Istio.Version
	uAct.Namespace = install.Spec.Istio.Namespace
	uAct.RepoURL = install.Spec.Istio.Source
	uAct.Wait = true
	uAct.Timeout = 5 * time.Minute

	// istio base
	iAct.ReleaseName = "base"
	values, err := helm.UnmarshalValues(install.Spec.Istio.Base.Values)
	if err != nil {
		return err
	}
	values["profile"] = "ambient"
	err = helm.InstallOrUpgrade(ctx, cfg, iAct, uAct, values)
	if err != nil {
		return err
	}

	// istio istiod
	iAct.ReleaseName = "istiod"
	values, err = helm.UnmarshalValues(install.Spec.Istio.Istiod.Values)
	if err != nil {
		return err
	}
	values["profile"] = "ambient"
	err = helm.InstallOrUpgrade(ctx, cfg, iAct, uAct, values)
	if err != nil {
		return err
	}

	// istio cni
	iAct.ReleaseName = "cni"
	values, err = helm.UnmarshalValues(install.Spec.Istio.CNI.Values)
	if err != nil {
		return err
	}
	values["profile"] = "ambient"
	err = helm.InstallOrUpgrade(ctx, cfg, iAct, uAct, values)
	if err != nil {
		return err
	}

	// istio ztunnel
	iAct.ReleaseName = "ztunnel"
	values, err = helm.UnmarshalValues(install.Spec.Istio.Ztunnel.Values)
	if err != nil {
		return err
	}
	// ztunnel is an ambient-only component so it doesn't need the profile set
	err = helm.InstallOrUpgrade(ctx, cfg, iAct, uAct, values)
	if err != nil {
		return err
	}

	return nil
}

func uninstallIstio(_ context.Context, install *v1.Install) error {
	actionConfiguration, err := helm.ActionConfiguration(install.Spec.Istio.Namespace)
	if err != nil {
		return err
	}

	act := action.NewUninstall(actionConfiguration)
	act.IgnoreNotFound = true
	act.Wait = true
	act.Timeout = 5 * time.Minute

	releases := []string{"ztunnel", "cni", "istiod", "base"}
	for _, release := range releases {
		_, err = act.Run(release)
		if err != nil {
			return err
		}
	}

	return nil
}
