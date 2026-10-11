package installs

import (
	"context"

	v1 "github.com/home-cloud-io/core/api/crds/v1"
	"github.com/home-cloud-io/core/pkg/install/resources"
	"github.com/home-cloud-io/core/pkg/kube"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func reconcileNamespaces(ctx context.Context, r *InstallReconciler, install *v1.Install) error {
	l := log.FromContext(ctx)

	l.Info("reconciling namespaces")
	for _, o := range resources.NamespaceObjects(install) {
		err := kube.CreateOrUpdate(ctx, r.Client, o)
		if err != nil {
			return err
		}
	}
	// no status update

	return nil
}

func uninstallNamespaces(ctx context.Context, r *InstallReconciler, install *v1.Install) error {
	return kube.UninstallResources(ctx, r.Client, resources.NamespaceObjects(install))
}
