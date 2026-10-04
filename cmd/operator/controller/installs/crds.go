package installs

import (
	"context"
	"fmt"
	"net/http"

	"golang.org/x/mod/semver"
	"k8s.io/client-go/dynamic"
	"sigs.k8s.io/controller-runtime/pkg/log"

	v1 "github.com/home-cloud-io/core/api/crds/v1"
	"github.com/home-cloud-io/core/pkg/kube"
)

func reconcileCRDs(ctx context.Context, r *InstallReconciler, install *v1.Install) error {
	l := log.FromContext(ctx)

	switch semver.Compare(install.Spec.Version, install.Status.Version) {
	case 1:
		l.Info("reconciling home cloud crds")

		resp, err := http.Get(fmt.Sprintf("%s/%s/crds.yaml", ReleasesURL, install.Spec.Version))
		if err != nil {
			return err
		}
		dynamicClient, err := dynamic.NewForConfig(r.Config)
		if err != nil {
			return err
		}
		err = kube.Apply(ctx, r.Client, dynamicClient, resp.Body)
		if err != nil {
			return err
		}

		install.Status.Version = install.Spec.Version
	case 0:
		l.V(1).Info("unchanged home cloud crds: skipping reconcile")
		return nil
	case -1:
		l.Error(fmt.Errorf("invalid CRD version"), "cannot install older CRDs over a newer version",
			"spec_version", install.Spec.Version, "status_version", install.Status.Version)
		// we won't kill the rest of the reconcile, just log and continue
		return nil
	}

	return nil
}
