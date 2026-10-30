package bugreport

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	opv1 "github.com/home-cloud-io/core/api/crds/v1"
)

type typeConfig struct {
	gvk    schema.GroupVersionKind
	filter func(o *unstructured.Unstructured)
}

var typesMap = map[string]typeConfig{
	// CORE
	"configmaps": {
		gvk: schema.GroupVersionKind{
			Group:   corev1.SchemeGroupVersion.Group,
			Version: corev1.SchemeGroupVersion.Version,
			Kind:    "ConfigMap",
		},
		filter: func(o *unstructured.Unstructured) {
			unstructured.RemoveNestedField(o.Object, "data")
		},
	},
	"secrets": {
		gvk: schema.GroupVersionKind{
			Group:   corev1.SchemeGroupVersion.Group,
			Version: corev1.SchemeGroupVersion.Version,
			Kind:    "Secret",
		},
		filter: func(o *unstructured.Unstructured) {
			unstructured.RemoveNestedField(o.Object, "data")
		},
	},
	// NOTE: we don't do pods here because we're already listing pods to get their logs
	// 		 so this would be redundant
	// "pods": {
	// 	gvk: schema.GroupVersionKind{
	// 		Group:   corev1.SchemeGroupVersion.Group,
	// 		Version: corev1.SchemeGroupVersion.Version,
	// 		Kind:    "Pod",
	// 	},
	// },
	// APPS
	"deployments": {
		gvk: schema.GroupVersionKind{
			Group:   appsv1.SchemeGroupVersion.Group,
			Version: appsv1.SchemeGroupVersion.Version,
			Kind:    "Deployment",
		},
	},
	"statefulsets": {
		gvk: schema.GroupVersionKind{
			Group:   appsv1.SchemeGroupVersion.Group,
			Version: appsv1.SchemeGroupVersion.Version,
			Kind:    "StatefulSet",
		},
	},
	"daemonsets": {
		gvk: schema.GroupVersionKind{
			Group:   appsv1.SchemeGroupVersion.Group,
			Version: appsv1.SchemeGroupVersion.Version,
			Kind:    "DaemonSet",
		},
	},
	// GATEWAY API
	"gateways": {
		gvk: schema.GroupVersionKind{
			Group:   gwv1.GroupVersion.Group,
			Version: gwv1.GroupVersion.Version,
			Kind:    "Gateway",
		},
	},
	"httproutes": {
		gvk: schema.GroupVersionKind{
			Group:   gwv1.GroupVersion.Group,
			Version: gwv1.GroupVersion.Version,
			Kind:    "HTTPRoute",
		},
	},
	// HOME CLOUD
	"apps": {
		gvk: schema.GroupVersionKind{
			Group:   opv1.GroupVersion.Group,
			Version: opv1.GroupVersion.Version,
			Kind:    "App",
		},
	},
	"disks": {
		gvk: schema.GroupVersionKind{
			Group:   opv1.GroupVersion.Group,
			Version: opv1.GroupVersion.Version,
			Kind:    "Disk",
		},
	},
	"installs": {
		gvk: schema.GroupVersionKind{
			Group:   opv1.GroupVersion.Group,
			Version: opv1.GroupVersion.Version,
			Kind:    "Install",
		},
	},
	"wireguard": {
		gvk: schema.GroupVersionKind{
			Group:   opv1.GroupVersion.Group,
			Version: opv1.GroupVersion.Version,
			Kind:    "Wireguard",
		},
	},
	// TODO: cert-manager... and more??
}
