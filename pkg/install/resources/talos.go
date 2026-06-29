package resources

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/home-cloud-io/core/api/crds/v1"
	talos "github.com/home-cloud-io/core/pkg/talos/api"
)

var (
	TalosObjects = func(install *v1.Install) []client.Object {
		return []client.Object{
			&talos.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "talos-api-access",
					Namespace: install.Namespace,
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
