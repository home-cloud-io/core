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
			},
			Settings: v1.SettingsSpec{
				Hostname: "home-cloud.local",
			},
		},
	}
)
