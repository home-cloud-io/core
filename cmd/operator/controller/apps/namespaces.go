package apps

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/home-cloud-io/core/api/crds/v1"
	"github.com/home-cloud-io/core/pkg/kube"
)

func reconcileNamespaces(ctx context.Context, r *AppReconciler, _ *v1.App, config *AppConfig) error {
	return kube.CreateOrUpdate(ctx, r.Client, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: config.Namespace,
			Labels: map[string]string{
				"istio.io/dataplane-mode": "ambient",
			},
		},
	})
}
