package resources

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"
)

const (
	DefaultIstioVersion       = "1.27.1"
	DefaultIstioNamespace     = "istio-system"
	DefaultIstioRepoURL       = "https://istio-release.storage.googleapis.com/charts"
	DefaultIngressGatewayName = "ingress-gateway"
)

var (
	IngressGateway = &gwv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{
			Name:      DefaultIngressGatewayName,
			Namespace: DefaultIstioNamespace,
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
	}
)
