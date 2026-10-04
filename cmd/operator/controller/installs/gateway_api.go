package installs

import (
	"context"
	"fmt"
	"net/http"

	"k8s.io/client-go/dynamic"
	"sigs.k8s.io/controller-runtime/pkg/log"

	v1 "github.com/home-cloud-io/core/api/crds/v1"
	"github.com/home-cloud-io/core/pkg/kube"
)

func reconcileGatewayAPI(ctx context.Context, r *InstallReconciler, install *v1.Install) error {
	l := log.FromContext(ctx)

	if !install.Spec.GatewayAPI.Disable {
		if install.Status.GatewayAPI == nil ||
			install.Spec.GatewayAPI.Source != install.Status.GatewayAPI.Source ||
			install.Spec.GatewayAPI.Version != install.Status.GatewayAPI.Version {
			l.Info("reconciling gateway api crds")

			resp, err := http.Get(fmt.Sprintf("%s/%s/standard-install.yaml", install.Spec.GatewayAPI.Source, install.Spec.GatewayAPI.Version))
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

			install.Status.GatewayAPI = &v1.GatewayAPIStatus{
				Source:  install.Spec.GatewayAPI.Source,
				Version: install.Spec.GatewayAPI.Version,
			}
		} else {
			l.V(1).Info("unchanged gateway api install: skipping reconcile")
			return nil
		}

	} else {
		install.Status.GatewayAPI = nil
	}
	return nil
}
