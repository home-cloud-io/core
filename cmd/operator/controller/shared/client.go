package shared

import (
	"context"
	"reflect"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

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
