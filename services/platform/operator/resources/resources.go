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
			GatewayAPI: &v1.GatewayAPISpec{},
			Istio: &v1.IstioSpec{
				Namespace:          "istio-system",
				IngressGatewayName: "ingress-gateway",
				Base:               &v1.BaseSpec{},
				Istiod: &v1.IstiodSpec{
					// The default istiod resources are cpu=500m and memory=2048Mi which is wayyyy
					// oversized for the typical Home Cloud installation.
					Values: `
resources:
  requests:
    cpu: 100m
    memory: 100Mi
`,
				},
				CNI: &v1.CNISpec{},
				Ztunnel: &v1.ZtunnelSpec{
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
			Server:   &v1.ServerSpec{},
			MDNS:     &v1.MDNSSpec{},
			Tunnel:   &v1.TunnelSpec{},
			Operator: &v1.OperatorSpec{},
			Daemon: &v1.DaemonSpec{
				System:     &v1.SystemSpec{},
				Kubernetes: &v1.KubernetesSpec{},
			},
			Settings: &v1.SettingsSpec{
				Hostname: "home-cloud.local",
			},
		},
		Status: v1.InstallStatus{
			GatewayAPI: &v1.GatewayAPIStatus{},
			Istio: &v1.IstioStatus{},
			Server: &v1.ServerStatus{},
			MDNS: &v1.MDNSStatus{},
			Tunnel: &v1.TunnelStatus{},
			Operator: &v1.OperatorStatus{},
			Daemon: &v1.DaemonStatus{},
		},
	}
)
