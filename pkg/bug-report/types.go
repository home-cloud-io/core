package bugreport

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type typeConfig struct {
	gvk    schema.GroupVersionKind
	filter func(o *unstructured.Unstructured)
}

var clusterTypes = map[string]typeConfig{
	"nodes": {
		gvk: schema.GroupVersionKind{
			Group:   corev1.SchemeGroupVersion.Group,
			Version: corev1.SchemeGroupVersion.Version,
			Kind:    "Node",
		},
	},
	"persistentvolumes": {
		gvk: schema.GroupVersionKind{
			Group:   corev1.SchemeGroupVersion.Group,
			Version: corev1.SchemeGroupVersion.Version,
			Kind:    "PersistentVolume",
		},
	},
}

var namespacedTypes = map[string]typeConfig{
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
	"persistentvolumesclaims": {
		gvk: schema.GroupVersionKind{
			Group:   corev1.SchemeGroupVersion.Group,
			Version: corev1.SchemeGroupVersion.Version,
			Kind:    "PersistentVolumeClaim",
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
	"services": {
		gvk: schema.GroupVersionKind{
			Group:   corev1.SchemeGroupVersion.Group,
			Version: corev1.SchemeGroupVersion.Version,
			Kind:    "Service",
		},
	},
}
