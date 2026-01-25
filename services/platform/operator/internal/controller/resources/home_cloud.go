package resources

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	v1 "github.com/home-cloud-io/core/services/platform/operator/api/v1"
	"github.com/home-cloud-io/core/services/platform/operator/internal/controller/talos"
)

var (
	HomeCloudObjects = func(install *v1.Install) []client.Object {
		var (
			objects       = []client.Object{}
			serverObjects = []client.Object{
				&corev1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "server",
						Namespace: install.Namespace,
					},
				},
				&rbacv1.Role{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "manage-home-cloud-apps",
						Namespace: install.Namespace,
					},
					Rules: []rbacv1.PolicyRule{
						{
							Verbs: []string{
								"*",
							},
							APIGroups: []string{"home-cloud.io"},
							Resources: []string{"apps", "installs", "wireguards"},
						},
						{
							Verbs: []string{
								"get",
								"create",
								"delete",
							},
							APIGroups: []string{""},
							Resources: []string{"secrets"},
						},
					},
				},
				&rbacv1.ClusterRole{
					ObjectMeta: metav1.ObjectMeta{
						Name: "read-all",
					},
					Rules: []rbacv1.PolicyRule{
						{
							Verbs: []string{
								"get",
								"list",
							},
							// TODO: limit these?
							APIGroups: []string{""},
							Resources: []string{"*"},
						},
					},
				},
				&rbacv1.RoleBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "manage-home-cloud-apps",
						Namespace: install.Namespace,
					},
					Subjects: []rbacv1.Subject{
						{
							Kind:      "ServiceAccount",
							Name:      "server",
							Namespace: install.Namespace,
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: "rbac.authorization.k8s.io",
						Kind:     "Role",
						Name:     "manage-home-cloud-apps",
					},
				},
				&rbacv1.ClusterRoleBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name: "home-cloud-server-read-all",
					},
					Subjects: []rbacv1.Subject{
						{
							Kind:      "ServiceAccount",
							Name:      "server",
							Namespace: install.Namespace,
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: "rbac.authorization.k8s.io",
						Kind:     "ClusterRole",
						Name:     "read-all",
					},
				},
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "server",
						Namespace: install.Namespace,
					},
					Data: map[string]string{
						"config.yaml": `service:
  name: server
  domain: home-cloud
  env: prod
`},
				},
				&appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "server",
						Namespace: install.Namespace,
					},
					Spec: appsv1.DeploymentSpec{
						Replicas: ptr.To[int32](1),
						Selector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								"app": "server",
							},
						},
						Template: corev1.PodTemplateSpec{
							ObjectMeta: metav1.ObjectMeta{
								Labels: map[string]string{
									"app": "server",
								},
							},
							Spec: corev1.PodSpec{
								ServiceAccountName: "server",
								Containers: []corev1.Container{
									{
										Name:  "server",
										Image: fmt.Sprintf("%s:%s", install.Spec.Server.Image, install.Spec.Server.Tag),
										Ports: []corev1.ContainerPort{
											{
												Name:          "http",
												Protocol:      corev1.ProtocolTCP,
												ContainerPort: 8090,
											},
										},
										LivenessProbe: &corev1.Probe{
											ProbeHandler: corev1.ProbeHandler{
												HTTPGet: &corev1.HTTPGetAction{
													Path: "/",
													Port: intstr.FromString("http"),
												},
											},
										},
										ReadinessProbe: &corev1.Probe{
											ProbeHandler: corev1.ProbeHandler{
												HTTPGet: &corev1.HTTPGetAction{
													Path: "/",
													Port: intstr.FromString("http"),
												},
											},
										},
										VolumeMounts: []corev1.VolumeMount{
											{
												Name:      "config",
												MountPath: "/etc/config.yaml",
												SubPath:   "config.yaml",
											},
										},
									},
								},
								Volumes: []corev1.Volume{
									{
										Name: "config",
										VolumeSource: corev1.VolumeSource{
											ConfigMap: &corev1.ConfigMapVolumeSource{
												LocalObjectReference: corev1.LocalObjectReference{Name: "server"},
											},
										},
									},
								},
							},
						},
					},
				},
				&corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "server",
						Namespace: install.Namespace,
						Labels: map[string]string{
							"app": "server",
						},
						Annotations: map[string]string{
							"home-cloud.io/dns": install.Spec.Settings.Hostname,
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
							"app": "server",
						},
					},
				},
				&gwv1.HTTPRoute{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "server",
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
												Name: "server",
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
			mdnsObjects = []client.Object{
				&corev1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "mdns",
						Namespace: install.Namespace,
					},
				},
				&rbacv1.ClusterRole{
					ObjectMeta: metav1.ObjectMeta{
						Name: "home-cloud-mdns",
					},
					Rules: []rbacv1.PolicyRule{
						{
							Verbs: []string{
								"list",
								"watch",
							},
							APIGroups: []string{""},
							Resources: []string{"services"},
						},
					},
				},
				&rbacv1.ClusterRoleBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name: "home-cloud-mdns-reader",
					},
					Subjects: []rbacv1.Subject{
						{
							Kind:      "ServiceAccount",
							Name:      "mdns",
							Namespace: install.Namespace,
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: "rbac.authorization.k8s.io",
						Kind:     "ClusterRole",
						Name:     "home-cloud-mdns",
					},
				},
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "mdns",
						Namespace: install.Namespace,
					},
					Data: map[string]string{
						"config.yaml": `
service:
  name: mdns
  domain: home-cloud
  env: prod
`},
				},
				&appsv1.StatefulSet{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "mdns",
						Namespace: install.Namespace,
					},
					Spec: appsv1.StatefulSetSpec{
						Replicas: ptr.To[int32](1),
						Selector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								"app": "mdns",
							},
						},
						Template: corev1.PodTemplateSpec{
							ObjectMeta: metav1.ObjectMeta{
								Labels: map[string]string{
									"app": "mdns",
								},
							},
							Spec: corev1.PodSpec{
								ServiceAccountName: "mdns",
								// need to be on host for creating the mDNS server on :5353
								// TODO: could potentially get around this with UDPRoutes in GatewayAPI
								HostNetwork: true,
								Containers: []corev1.Container{
									{
										Name:  "mdns",
										Image: fmt.Sprintf("%s:%s", install.Spec.MDNS.Image, install.Spec.MDNS.Tag),
										SecurityContext: &corev1.SecurityContext{
											RunAsUser:                ptr.To(int64(65534)),
											RunAsGroup:               ptr.To(int64(65534)),
											RunAsNonRoot:             ptr.To(true),
											ReadOnlyRootFilesystem:   ptr.To(true),
											AllowPrivilegeEscalation: ptr.To(false),
											SeccompProfile: &corev1.SeccompProfile{
												Type: corev1.SeccompProfileTypeRuntimeDefault,
											},
											Capabilities: &corev1.Capabilities{
												Drop: []corev1.Capability{
													"ALL",
												},
											},
										},
										Env: []corev1.EnvVar{
											{
												Name: "DRAFT_MDNS_HOST_IP",
												ValueFrom: &corev1.EnvVarSource{
													FieldRef: &corev1.ObjectFieldSelector{
														FieldPath: "status.hostIP",
													},
												},
											},
										},
										// no Live/Readiness checks since this lives on the host network
										VolumeMounts: []corev1.VolumeMount{
											{
												Name:      "config",
												MountPath: "/etc/config.yaml",
												SubPath:   "config.yaml",
											},
										},
									},
								},
								Volumes: []corev1.Volume{
									{
										Name: "config",
										VolumeSource: corev1.VolumeSource{
											ConfigMap: &corev1.ConfigMapVolumeSource{
												LocalObjectReference: corev1.LocalObjectReference{Name: "mdns"},
											},
										},
									},
								},
							},
						},
					},
				},
			}
			tunnelObjects = []client.Object{
				&corev1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "tunnel",
						Namespace: install.Namespace,
					},
				},
				&rbacv1.ClusterRole{
					ObjectMeta: metav1.ObjectMeta{
						Name: "home-cloud-tunnel",
					},
					Rules: []rbacv1.PolicyRule{
						{
							Verbs: []string{
								"*",
							},
							APIGroups: []string{"home-cloud.io"},
							Resources: []string{"wireguards"},
						},
						{
							Verbs: []string{
								"*",
							},
							APIGroups: []string{"home-cloud.io"},
							Resources: []string{"wireguards/status"},
						},
						{
							Verbs: []string{
								"get",
								"list",
								"watch",
							},
							APIGroups: []string{""},
							Resources: []string{"secrets"},
						},
					},
				},
				&rbacv1.ClusterRoleBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name: "home-cloud-tunnel",
					},
					Subjects: []rbacv1.Subject{
						{
							Kind:      "ServiceAccount",
							Name:      "tunnel",
							Namespace: install.Namespace,
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: "rbac.authorization.k8s.io",
						Kind:     "ClusterRole",
						Name:     "home-cloud-tunnel",
					},
				},
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "tunnel",
						Namespace: install.Namespace,
					},
					Data: map[string]string{
						"config.yaml": `
service:
  name: tunnel
  domain: home-cloud
  env: prod
  network:
    bind_port: 8091
`},
				},
				&appsv1.StatefulSet{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "tunnel",
						Namespace: install.Namespace,
					},
					Spec: appsv1.StatefulSetSpec{
						Replicas: ptr.To[int32](1),
						Selector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								"app": "tunnel",
							},
						},
						Template: corev1.PodTemplateSpec{
							ObjectMeta: metav1.ObjectMeta{
								Labels: map[string]string{
									"app": "tunnel",
								},
							},
							Spec: corev1.PodSpec{
								ServiceAccountName: "tunnel",
								// need to be on host for Wireguard routing without k8s fudging NAT stuff
								// TODO: could potentially get around this with UDPRoutes in GatewayAPI
								HostNetwork: true,
								Containers: []corev1.Container{
									{
										Name:  "tunnel",
										Image: fmt.Sprintf("%s:%s", install.Spec.Tunnel.Image, install.Spec.Tunnel.Tag),
										SecurityContext: &corev1.SecurityContext{
											Privileged: ptr.To(true),
											Capabilities: &corev1.Capabilities{
												Add: []corev1.Capability{
													"NET_ADMIN",
												},
											},
										},
										// no Live/Readiness checks since this lives on the host network
										VolumeMounts: []corev1.VolumeMount{
											{
												Name:      "config",
												MountPath: "/etc/config.yaml",
												SubPath:   "config.yaml",
											},
										},
									},
								},
								Volumes: []corev1.Volume{
									{
										Name: "config",
										VolumeSource: corev1.VolumeSource{
											ConfigMap: &corev1.ConfigMapVolumeSource{
												LocalObjectReference: corev1.LocalObjectReference{Name: "tunnel"},
											},
										},
									},
								},
							},
						},
					},
				},
			}
			daemonObjects = []client.Object{
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
		)

		if !install.Spec.Server.Disable {
			objects = append(objects, serverObjects...)
		}

		if !install.Spec.MDNS.Disable {
			objects = append(objects, mdnsObjects...)
		}

		if !install.Spec.Tunnel.Disable {
			objects = append(objects, tunnelObjects...)
		}

		if !install.Spec.Daemon.Disable {
			objects = append(objects, daemonObjects...)
		}

		return objects
	}
)
