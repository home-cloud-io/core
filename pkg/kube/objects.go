package kube

import (
	"context"
	"errors"
	"io"
	"reflect"
	"slices"

	"go.yaml.in/yaml/v2"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func InstallResources(ctx context.Context, kube client.Client, objects []client.Object) error {
	for _, o := range objects {
		err := CreateOrUpdate(ctx, kube, o)
		if err != nil {
			return err
		}
	}
	return nil
}

func UninstallResources(ctx context.Context, kube client.Client, objects []client.Object) error {
	for _, o := range slices.Backward(objects) {
		err := kube.Delete(ctx, o)
		if client.IgnoreNotFound(err) != nil {
			return err
		}
	}
	return nil
}

func CreateOrUpdate(ctx context.Context, kube client.Client, obj client.Object) error {
	// this is a bit of a mess and might not be totally necessary but it creates a new instance
	// of the same underlying type in obj (which must be a pointer) so that we don't overwrite all
	// fields when we really only want the ResourceVersion
	c := reflect.New(reflect.TypeOf(obj).Elem()).Interface().(client.Object)

	err := kube.Get(ctx, client.ObjectKeyFromObject(obj), c)
	if err != nil {
		if kerrors.IsNotFound(err) {
			return kube.Create(ctx, obj)
		}
		return err
	}
	obj.SetResourceVersion(c.GetResourceVersion())
	return kube.Update(ctx, obj)
}

// Apply applies the given YAML manifests to kubernetes
func Apply(ctx context.Context, kube client.Client, dynamicClient *dynamic.DynamicClient, reader io.Reader) error {
	l := log.FromContext(ctx)
	dec := yaml.NewDecoder(reader)
	for {
		// parse the YAML doc
		obj := &unstructured.Unstructured{Object: map[string]interface{}{}}
		err := dec.Decode(obj.Object)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
		if obj.Object == nil {
			l.Info("skipping empty document")
			continue
		}
		// get GroupVersionResource to invoke the dynamic client
		gvk := obj.GroupVersionKind()
		restMapping, err := kube.RESTMapper().RESTMapping(gvk.GroupKind(), gvk.Version)
		if err != nil {
			return err
		}
		gvr := restMapping.Resource
		// apply the YAML doc
		namespace := obj.GetNamespace()
		if len(namespace) == 0 {
			namespace = "default"
		}

		applyOpts := metav1.ApplyOptions{FieldManager: "home-cloud-operator"}
		_, err = dynamicClient.Resource(gvr).Apply(context.TODO(), obj.GetName(), obj, applyOpts)
		if err != nil {
			l.Error(err, "failed to apply object, attempting forced apply", "kind", obj.GetKind(), "name", obj.GetName())
			applyOpts.Force = true
			_, err = dynamicClient.Resource(gvr).Apply(context.TODO(), obj.GetName(), obj, applyOpts)
			if err != nil {
				l.Error(err, "failed to apply object with force, aborting", "kind", obj.GetKind(), "name", obj.GetName())
				return err
			}
		}
		l.V(1).Info("applied YAML for object", "kind", obj.GetKind(), "name", obj.GetName())
	}
	return nil
}
