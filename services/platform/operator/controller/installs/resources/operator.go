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
)

var (
	// NOTE: For these resources specifically, they have the TypeMeta manually configured.
	//       This is not necessary for most usecases but it allows `tools/releaser` to use
	//       these definitions to create the releasable `operator.yaml` consistent with what
	//       the operator will reconcile itself.
	OperatorObjects = func(install *v1.Install) []client.Object {
		return []client.Object{
			&corev1.ServiceAccount{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "ServiceAccount",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "operator",
					Namespace: install.Namespace,
				},
			},
			&rbacv1.ClusterRoleBinding{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "rbac.authorization.k8s.io/v1",
					Kind:       "ClusterRoleBinding",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "operator-rolebinding",
				},
				RoleRef: rbacv1.RoleRef{
					APIGroup: "rbac.authorization.k8s.io",
					Kind:     "ClusterRole",
					Name:     "cluster-admin",
				},
				Subjects: []rbacv1.Subject{
					{
						Kind:      "ServiceAccount",
						Name:      "operator",
						Namespace: install.Namespace,
					},
				},
			},
			&corev1.ConfigMap{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "ConfigMap",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "operator",
					Namespace: install.Namespace,
				},
				Data: map[string]string{
					"config.yaml": `service:
  name: operator
  domain: home-cloud
  env: prod
`},
			},
			&appsv1.Deployment{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "operator",
					Namespace: install.Namespace,
				},
				Spec: appsv1.DeploymentSpec{
					Replicas: ptr.To[int32](1),
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app": "operator",
						},
					},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"app": "operator",
							},
						},
						Spec: corev1.PodSpec{
							TerminationGracePeriodSeconds: ptr.To(int64(10)),
							ServiceAccountName:            "operator",
							Containers: []corev1.Container{
								{
									Name:  "operator",
									Image: fmt.Sprintf("%s:%s", install.Spec.Operator.Image, install.Spec.Operator.Tag),
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
											LocalObjectReference: corev1.LocalObjectReference{Name: "operator"},
										},
									},
								},
							},
						},
					},
				},
			},
			&corev1.Service{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "Service",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "operator",
					Namespace: install.Namespace,
					Labels: map[string]string{
						"app": "operator",
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
						"app": "operator",
					},
				},
			},
			&gwv1.HTTPRoute{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "gateway.networking.k8s.io/v1",
					Kind:       "HTTPRoute",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "operator",
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
											Name: "operator",
											Port: ptr.To[gwv1.PortNumber](80),
										},
									},
								},
							},
						},
					},
				},
				// TODO: is this necessary for releaser? Otherwise we get a status.parents=null
				Status: gwv1.HTTPRouteStatus{
					RouteStatus: gwv1.RouteStatus{
						Parents: []gwv1.RouteParentStatus{},
					},
				},
			},
		}
	}
)
