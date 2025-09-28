package resources

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	DraftObjects = []client.Object{
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "draft-system",
			},
		},
		&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "blueprint",
				Namespace: "draft-system",
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
		&corev1.PersistentVolume{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "blueprint",
				Labels: map[string]string{"type": "local"},
			},
			Spec: corev1.PersistentVolumeSpec{
				Capacity: corev1.ResourceList{corev1.ResourceName("storage"): resource.MustParse("5G")},
				// TODO: change this
				PersistentVolumeSource: corev1.PersistentVolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: "/mnt"}},
				AccessModes:            []corev1.PersistentVolumeAccessMode{corev1.PersistentVolumeAccessMode("ReadWriteMany")},
				ClaimRef: &corev1.ObjectReference{
					Namespace: "draft-system",
					Name:      "blueprint",
				},
				PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimPolicy("Retain"),
				StorageClassName:              "manual",
				NodeAffinity: &corev1.VolumeNodeAffinity{Required: &corev1.NodeSelector{NodeSelectorTerms: []corev1.NodeSelectorTerm{
					{
						MatchExpressions: []corev1.NodeSelectorRequirement{
							{
								Key:      "kubernetes.io/hostname",
								Operator: corev1.NodeSelectorOperator("In"),
								Values:   []string{"home-cloud"},
							},
						},
					},
				}},
				},
			},
		},
		&corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "blueprint",
				Namespace: "draft-system",
			},
			Spec: corev1.PersistentVolumeClaimSpec{
				AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.PersistentVolumeAccessMode("ReadWriteMany")},
				StorageClassName: ptr.To[string]("manual"),
				Resources: corev1.VolumeResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceStorage: resource.MustParse("5G"),
					},
				},
			},
		},
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "blueprint",
				Namespace: "draft-system",
			},
			Data: map[string]string{"config.yaml": `
service:
  name: blueprint
  domain: core
  env: prod
badger:
  path: /etc/badger/data
raft:
  node-id: node_1
  address: blueprint.draft-system.svc.cluster.local
  port: 1111
  bootstrap: true
`},
		},
		&appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "blueprint",
				Namespace: "draft-system",
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
							{
								Name:         "badger",
								VolumeSource: corev1.VolumeSource{PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: "blueprint"}},
							},
						},
						Containers: []corev1.Container{
							{
								Name:  "blueprint",
								Image: "ghcr.io/steady-bytes/draft-core-blueprint:v0.0.6",
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
									{
										Name:      "badger",
										MountPath: "/etc/badger/data",
									},
								},
							},
						},
					},
				},
				ServiceName: "blueprint",
			},
		},
	}
)
