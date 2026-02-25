package resources

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/home-cloud-io/core/services/platform/operator/api/v1"
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
				Istiod: v1.IstiodSpec{
					// The default istiod resources are cpu=500m and memory=2048Mi which is wayyyy
					// oversized for the typical Home Cloud installation.
					Values: `
resources:
  requests:
    cpu: 100m
    memory: 100Mi
`,
				},
				Ztunnel: v1.ZtunnelSpec{
					// The default ztunnel resources are cpu=200m and memory=512Mi which is wayyyy
					// oversized for the typical Home Cloud installation.
					Values: `
resources:
  requests:
    cpu: 100m
    memory: 100Mi
`,
				},
			},
			Settings: v1.SettingsSpec{
				Hostname: "home-cloud.local",
			},
		},
	}
)
