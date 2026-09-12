package resources

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/home-cloud-io/core/api/crds/v1"
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
				Namespace: "istio-system",
				// TODO: do this through a separate Gateway resources so that people can run without Istio
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
			GenericDevicePlugin: &v1.GenericDevicePluginSpec{},
			MDNS:                &v1.MDNSSpec{},
			Tunnel:              &v1.TunnelSpec{},
			Operator:            &v1.OperatorSpec{},
			Daemon: &v1.DaemonSpec{
				System:     &v1.SystemSpec{},
				Kubernetes: &v1.KubernetesSpec{},
			},
			Settings: &v1.SettingsSpec{
				Domains: []string{"local"},
				StorageApps: []string{
					"filebrowser",
					"nextexplorer",
				},
			},
		},
		Status: v1.InstallStatus{
			GatewayAPI:          &v1.GatewayAPIStatus{},
			Istio:               &v1.IstioStatus{},
			GenericDevicePlugin: &v1.GenericDevicePluginStatus{},
			MDNS:                &v1.MDNSStatus{},
			Tunnel:              &v1.TunnelStatus{},
			Operator:            &v1.OperatorStatus{},
			Daemon:              &v1.DaemonStatus{},
		},
	}

	GenericDevicePluginObjects = func(install *v1.Install) []client.Object {
		return []client.Object{
			&appsv1.DaemonSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "generic-device-plugin",
					Namespace: "kube-system",
					Labels: map[string]string{
						"app.kubernetes.io/name": "generic-device-plugin",
					},
				},
				Spec: appsv1.DaemonSetSpec{
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app.kubernetes.io/name": "generic-device-plugin",
						},
					},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"app.kubernetes.io/name": "generic-device-plugin",
							},
						},
						Spec: corev1.PodSpec{
							PriorityClassName: "system-node-critical",
							Tolerations: []corev1.Toleration{
								{
									Operator: corev1.TolerationOpExists,
									Effect:   corev1.TaintEffectNoExecute,
								},
								{
									Operator: corev1.TolerationOpExists,
									Effect:   corev1.TaintEffectNoSchedule,
								},
							},
							Containers: []corev1.Container{
								{
									Name:  "generic-device-plugin",
									Image: fmt.Sprintf("%s:%s", install.Spec.GenericDevicePlugin.Image, install.Spec.GenericDevicePlugin.Tag),
									SecurityContext: &corev1.SecurityContext{
										Privileged: ptr.To(true),
									},
									Args: []string{
										"--device",
										`name: serial
groups:
  - paths:
      - path: /dev/ttyUSB*`,
									},
									Resources: corev1.ResourceRequirements{
										Requests: corev1.ResourceList{
											corev1.ResourceCPU:    resource.MustParse("50m"),
											corev1.ResourceMemory: resource.MustParse("10Mi"),
										},
										Limits: corev1.ResourceList{
											corev1.ResourceCPU:    resource.MustParse("50m"),
											corev1.ResourceMemory: resource.MustParse("20Mi"),
										},
									},
									Ports: []corev1.ContainerPort{
										{
											Name:          "http",
											ContainerPort: 8080,
										},
									},
									VolumeMounts: []corev1.VolumeMount{
										{
											Name:      "device-plugin",
											MountPath: "/var/lib/kubelet/device-plugins",
										},
										{
											Name:      "dev",
											MountPath: "/dev",
										},
									},
								},
							},
							Volumes: []corev1.Volume{
								{
									Name: "device-plugin",
									VolumeSource: corev1.VolumeSource{
										HostPath: &corev1.HostPathVolumeSource{
											Path: "/var/lib/kubelet/device-plugins",
										},
									},
								},
								{
									Name: "dev",
									VolumeSource: corev1.VolumeSource{
										HostPath: &corev1.HostPathVolumeSource{
											Path: "/dev",
										},
									},
								},
							},
						},
					},
					UpdateStrategy: appsv1.DaemonSetUpdateStrategy{
						Type: appsv1.RollingUpdateDaemonSetStrategyType,
					},
				},
			},
		}
	}
)
