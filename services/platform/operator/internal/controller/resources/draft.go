package resources

import (
	"fmt"

	v1 "github.com/home-cloud-io/core/services/platform/operator/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	DraftObjects = func(install *v1.Install) []client.Object {
		return []client.Object{
			&corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "blueprint",
					Namespace: install.Spec.Draft.Namespace,
				},
				Spec: corev1.ServiceSpec{
					Ports: []corev1.ServicePort{
						{
							Name: "http",
							Port: 8090,
						},
					},
					Selector: map[string]string{"app": "blueprint"},
				},
			},
			// &corev1.PersistentVolume{
			// 	ObjectMeta: metav1.ObjectMeta{
			// 		Name:   "blueprint",
			// 		Labels: map[string]string{"type": "local"},
			// 	},
			// 	// TODO: make this more configurable for things like NFS, etc.
			// 	Spec: corev1.PersistentVolumeSpec{
			// 		Capacity:               corev1.ResourceList{corev1.ResourceName("storage"): resource.MustParse("5G")},
			// 		PersistentVolumeSource: corev1.PersistentVolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: filepath.Join(install.Spec.VolumeMountHostPath, "blueprint")}},
			// 		AccessModes:            []corev1.PersistentVolumeAccessMode{corev1.PersistentVolumeAccessMode("ReadWriteMany")},
			// 		ClaimRef: &corev1.ObjectReference{
			// 			Namespace: install.Spec.Draft.Namespace,
			// 			Name:      "blueprint",
			// 		},
			// 		PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimPolicy("Retain"),
			// 		StorageClassName:              "manual",
			// 		NodeAffinity: &corev1.VolumeNodeAffinity{Required: &corev1.NodeSelector{NodeSelectorTerms: []corev1.NodeSelectorTerm{
			// 			{
			// 				MatchExpressions: []corev1.NodeSelectorRequirement{
			// 					{
			// 						Key:      "kubernetes.io/hostname",
			// 						Operator: corev1.NodeSelectorOperator("In"),
			// 						// TODO: change back to home-cloud
			// 						Values:   []string{"talos-dzc-j08"},
			// 					},
			// 				},
			// 			},
			// 		}},
			// 		},
			// 	},
			// },
			// &corev1.PersistentVolumeClaim{
			// 	ObjectMeta: metav1.ObjectMeta{
			// 		Name:      "blueprint",
			// 		Namespace: install.Spec.Draft.Namespace,
			// 	},
			// 	Spec: corev1.PersistentVolumeClaimSpec{
			// 		AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.PersistentVolumeAccessMode("ReadWriteMany")},
			// 		StorageClassName: ptr.To[string]("manual"),
			// 		Resources: corev1.VolumeResourceRequirements{
			// 			Requests: corev1.ResourceList{
			// 				corev1.ResourceStorage: resource.MustParse("5G"),
			// 			},
			// 		},
			// 	},
			// },
			&corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "blueprint",
					Namespace: install.Spec.Draft.Namespace,
				},
				Data: map[string]string{"config.yaml": fmt.Sprintf(`
service:
  name: blueprint
  domain: core
  env: prod
badger:
  path: /etc/badger
raft:
  node-id: node_1
  address: blueprint.%s
  port: 1111
  bootstrap: true
`, install.Spec.Draft.Namespace)},
			},
			&appsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "blueprint",
					Namespace: install.Spec.Draft.Namespace,
				},
				Spec: appsv1.StatefulSetSpec{
					Replicas: ptr.To[int32](1),
					Selector: &metav1.LabelSelector{MatchLabels: map[string]string{
						"app": "blueprint",
					},
					},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "blueprint"}},
						Spec: corev1.PodSpec{
							Volumes: []corev1.Volume{
								{
									Name: "config",
									VolumeSource: corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{
										LocalObjectReference: corev1.LocalObjectReference{Name: "blueprint"},
									},
									},
								},
								// {
								// 	Name:         "badger",
								// 	VolumeSource: corev1.VolumeSource{PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: "blueprint"}},
								// },
							},
							Containers: []corev1.Container{
								{
									Name:  "blueprint",
									Image: fmt.Sprintf("%s:%s", install.Spec.Draft.Blueprint.Image, install.Spec.Draft.Blueprint.Tag),
									Ports: []corev1.ContainerPort{
										{
											Name:          "grpc",
											ContainerPort: 8090,
											Protocol:      corev1.Protocol("TCP"),
										},
										{
											Name:          "raft",
											ContainerPort: 1111,
											Protocol:      corev1.Protocol("TCP"),
										},
									},
									VolumeMounts: []corev1.VolumeMount{
										{
											Name:      "config",
											MountPath: "/etc/config.yaml",
											SubPath:   "config.yaml",
										},
										// {
										// 	Name:      "badger",
										// 	MountPath: "/etc/badger/data",
										// },
									},
								},
							},
						},
					},
					ServiceName: "blueprint",
				},
			},
		}
	}
)
