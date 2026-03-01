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
	TunnelObjects = func(install *v1.Install) []client.Object {
		return []client.Object{
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
	}
)
