package resources

import (
	v1 "github.com/home-cloud-io/core/services/platform/operator/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"
)

var (
	DefaultInstall = &v1.Install{
		Spec: v1.InstallSpec{
			GatewayAPI: v1.GatewayAPISpec{
				Version: "v1.3.0",
			},
			Istio: v1.IstioSpec{
				Version:            "1.27.1",
				Namespace:          "istio-system",
				Repo:               "https://istio-release.storage.googleapis.com/charts",
				IngressGatewayName: "ingress-gateway",
			},
			Draft: v1.DraftSpec{
				Namespace: "draft-system",
				Blueprint: v1.BlueprintSpec{
					Image: "ghcr.io/steady-bytes/draft-core-blueprint",
					Tag:   "v0.0.6",
				},
			},
			HomeCloud: v1.HomeCloudSpec{
				Namespace: "home-cloud-system",
				Hostname:  "home-cloud.local",
				Server: v1.ServerSpec{
					Image: "ghcr.io/home-cloud-io/core-platform-server",
					Tag:   "v0.0.52",
				},
			},
			VolumeMountHostPath: "/mnt/k8s-pvs/",
		},
	}
)

var (
	CommonObjects = func(install *v1.Install) []client.Object {
		return []client.Object{
			&gwv1.Gateway{
				ObjectMeta: metav1.ObjectMeta{
					Name:      install.Spec.Istio.IngressGatewayName,
					Namespace: install.Spec.Istio.Namespace,
				},
				Spec: gwv1.GatewaySpec{
					GatewayClassName: "istio",
					Listeners: []gwv1.Listener{
						{
							Name:     "http",
							Port:     80,
							Protocol: gwv1.HTTPProtocolType,
							AllowedRoutes: &gwv1.AllowedRoutes{
								Namespaces: &gwv1.RouteNamespaces{
									From: ptr.To[gwv1.FromNamespaces]("All"),
								},
							},
						},
					},
				},
			},
		}
	}
)
