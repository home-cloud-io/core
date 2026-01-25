package resources

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/home-cloud-io/core/services/platform/operator/api/v1"
)

var (
	MDNSObjects = func(install *v1.Install) []client.Object {
		return []client.Object{
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
	}
)
