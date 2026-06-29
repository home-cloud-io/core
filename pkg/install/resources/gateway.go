package resources

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	v1 "github.com/home-cloud-io/core/api/crds/v1"
)

var (
	GatewayObjects = func(install *v1.Install) []client.Object {
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
			&gwv1.HTTPRoute{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "operator",
					Namespace: install.Namespace,
				},
				Spec: gwv1.HTTPRouteSpec{
					CommonRouteSpec: gwv1.CommonRouteSpec{
						ParentRefs: []gwv1.ParentReference{
							{
								Name:      gwv1.ObjectName(install.Spec.Istio.IngressGatewayName),
								Namespace: ptr.To(gwv1.Namespace(install.Spec.Istio.Namespace)),
							},
						},
					},
					Hostnames: []gwv1.Hostname{
						gwv1.Hostname(install.Spec.Settings.Hostname),
					},
					Rules: []gwv1.HTTPRouteRule{
						{
							BackendRefs: []gwv1.HTTPBackendRef{
								{
									BackendRef: gwv1.BackendRef{
										BackendObjectReference: gwv1.BackendObjectReference{
											Name: "operator",
											Port: ptr.To[gwv1.PortNumber](80),
										},
									},
								},
							},
						},
					},
				},
			},
		}
	}
)
