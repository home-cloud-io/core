package resources

import (
	"fmt"

	v1 "github.com/home-cloud-io/core/services/platform/operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"
)

var (
	DefaultInstall = &v1.Install{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "install",
			Namespace: "home-cloud-system",
		},
		Spec: v1.InstallSpec{
			Istio: v1.IstioSpec{
				Namespace:          "istio-system",
				IngressGatewayName: "ingress-gateway",
			},
			Settings: v1.SettingsSpec{
				Hostname:         "home-cloud.local",
				AutoUpdateApps:   true,
				AutoUpdateSystem: true,
			},
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
					Infrastructure: &gwv1.GatewayInfrastructure{
						ParametersRef: &gwv1.LocalParametersReference{
							Kind: gwv1.Kind("ConfigMap"),
							Name: fmt.Sprintf("%s-options", install.Spec.Istio.IngressGatewayName),
						},
					},
				},
			},
			&corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("%s-options", install.Spec.Istio.IngressGatewayName),
					Namespace: install.Spec.Istio.Namespace,
				},
				Data: map[string]string{
					"service": `spec:
  type: NodePort
  ports:
  - port: 80
    nodePort: 80
`,
				},
			},
		}
	}
)
