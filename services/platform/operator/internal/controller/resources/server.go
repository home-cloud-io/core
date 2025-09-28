package resources

import (
	"fmt"

	v1 "github.com/home-cloud-io/core/services/platform/operator/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"
)

var (
	HomeCloudServerObjects = func(install *v1.Install) []client.Object {
		return []client.Object{
			&corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: install.Spec.HomeCloud.Namespace,
					Labels: map[string]string{
						"istio.io/dataplane-mode": "ambient",
					},
				},
			},
			&rbacv1.Role{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "manage-home-cloud-apps",
					Namespace: install.Spec.HomeCloud.Namespace,
				},
				Rules: []rbacv1.PolicyRule{
					{
						Verbs: []string{
							"*",
						},
						APIGroups: []string{"home-cloud.io"},
						Resources: []string{"apps", "installs"},
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
					Namespace: install.Spec.HomeCloud.Namespace,
				},
				Subjects: []rbacv1.Subject{
					{
						Kind:      "ServiceAccount",
						Name:      "default",
						Namespace: install.Spec.HomeCloud.Namespace,
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
						Name:      "default",
						Namespace: install.Spec.HomeCloud.Namespace,
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
					Namespace: install.Spec.HomeCloud.Namespace,
				},
				Data: map[string]string{
					"config.yaml": fmt.Sprintf(`
service:
  name: server
  domain: home-cloud
  env: prod
  entrypoint: http://blueprint.%s:8090
`, install.Spec.Draft.Namespace)},
			},
			&appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "server",
					Namespace: install.Spec.HomeCloud.Namespace,
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
							Containers: []corev1.Container{
								{
									Name:  "server",
									Image: fmt.Sprintf("%s:%s", install.Spec.HomeCloud.Server.Image, install.Spec.HomeCloud.Server.Tag),
									Ports: []corev1.ContainerPort{
										{
											Name:          "http",
											Protocol:      corev1.ProtocolTCP,
											ContainerPort: 8090,
										},
									},
									VolumeMounts: []corev1.VolumeMount{
										{
											Name:      "config",
											MountPath: "/etc/config.yaml",
											SubPath:   "config.yaml",
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
								},
							},
							Volumes: []corev1.Volume{
								{
									Name: "config",
									VolumeSource: corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{
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
					Namespace: install.Spec.HomeCloud.Namespace,
					Labels: map[string]string{
						"app": "server",
					},
				},
				Spec: corev1.ServiceSpec{
					Type: corev1.ServiceTypeClusterIP,
					Ports: []corev1.ServicePort{
						{
							Name: "http",
							Port: 8090,
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
					Namespace: install.Spec.HomeCloud.Namespace,
				},
				Spec: gwv1.HTTPRouteSpec{
					CommonRouteSpec: gwv1.CommonRouteSpec{
						ParentRefs: []gwv1.ParentReference{
							{
								Name:      gwv1.ObjectName(install.Spec.Istio.IngressGatewayName),
								Namespace: ptr.To[gwv1.Namespace](gwv1.Namespace(install.Spec.Istio.Namespace)),
							},
						},
					},
					Hostnames: []gwv1.Hostname{
						gwv1.Hostname(install.Spec.HomeCloud.Hostname),
					},
					Rules: []gwv1.HTTPRouteRule{
						{
							BackendRefs: []gwv1.HTTPBackendRef{
								{
									BackendRef: gwv1.BackendRef{
										BackendObjectReference: gwv1.BackendObjectReference{
											Name: "server",
											Port: ptr.To[gwv1.PortNumber](8090),
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
