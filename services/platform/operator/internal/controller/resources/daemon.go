package resources

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/home-cloud-io/core/services/platform/operator/api/v1"
	"github.com/home-cloud-io/core/services/platform/operator/internal/controller/talos"
)

// TODO: change this to use a Helm chart so users can simply point at a custom Helm chart
// and install everything in one go

var (
	DaemonObjects = func(install *v1.Install) []client.Object {
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
			&corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "daemon",
					Namespace: install.Namespace,
				},
				Data: map[string]string{
					"config.yaml": `
service:
  name: daemon
  domain: home-cloud
  env: prod
`},
			},
			&appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "daemon",
					Namespace: install.Namespace,
				},
				Spec: appsv1.DeploymentSpec{
					Replicas: ptr.To[int32](1),
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app": "daemon",
						},
					},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"app": "daemon",
							},
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "daemon",
									Image: fmt.Sprintf("%s:%s", install.Spec.Daemon.Image, install.Spec.Daemon.Tag),
									Ports: []corev1.ContainerPort{
										{
											Name:          "http",
											Protocol:      corev1.ProtocolTCP,
											ContainerPort: 8090,
										},
									},
									// TODO: add default live/readiness checks into the chassis
									// LivenessProbe: &corev1.Probe{
									// 	ProbeHandler: corev1.ProbeHandler{
									// 		HTTPGet: &corev1.HTTPGetAction{
									// 			Path: "/",
									// 			Port: intstr.FromString("http"),
									// 		},
									// 	},
									// },
									// ReadinessProbe: &corev1.Probe{
									// 	ProbeHandler: corev1.ProbeHandler{
									// 		HTTPGet: &corev1.HTTPGetAction{
									// 			Path: "/",
									// 			Port: intstr.FromString("http"),
									// 		},
									// 	},
									// },
									VolumeMounts: []corev1.VolumeMount{
										{
											Name:      "config",
											MountPath: "/etc/config.yaml",
											SubPath:   "config.yaml",
										},
										{
											Name:      "talos-secrets",
											MountPath: "/var/run/secrets/talos.dev",
										},
									},
								},
							},
							Volumes: []corev1.Volume{
								{
									Name: "config",
									VolumeSource: corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{
										LocalObjectReference: corev1.LocalObjectReference{Name: "daemon"},
									}},
								},
								{
									Name: "talos-secrets",
									VolumeSource: corev1.VolumeSource{Secret: &corev1.SecretVolumeSource{
										SecretName: "talos-api-access",
									}},
								},
							},
						},
					},
				},
			},
			&corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "daemon",
					Namespace: install.Namespace,
					Labels: map[string]string{
						"app": "daemon",
					},
				},
				Spec: corev1.ServiceSpec{
					Type: corev1.ServiceTypeClusterIP,
					Ports: []corev1.ServicePort{
						{
							Name:       "http",
							Port:       80,
							TargetPort: intstr.FromInt(8090),
						},
					},
					Selector: map[string]string{
						"app": "daemon",
					},
				},
			},
		}
	}
)
