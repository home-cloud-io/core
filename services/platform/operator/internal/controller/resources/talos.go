package resources

import (
	v1 "github.com/home-cloud-io/core/services/platform/operator/api/v1"
	"github.com/home-cloud-io/core/services/platform/operator/internal/controller/talos"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	TalosObjects = func(install *v1.Install) []client.Object {
		return []client.Object{
			&talos.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "talos-api-access",
					Namespace: install.Spec.HomeCloud.Namespace,
				},
				Spec: talos.ServiceAccountSpec{
					Roles: []string{
						"os:admin",
					},
				},
			},
		}
	}
)
