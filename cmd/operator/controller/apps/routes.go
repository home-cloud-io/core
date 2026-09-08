package apps

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	v1 "github.com/home-cloud-io/core/api/crds/v1"
	"github.com/home-cloud-io/core/cmd/operator/controller/shared"
	"github.com/home-cloud-io/core/pkg/install/resources"
)

func (r *AppReconciler) createRoute(ctx context.Context, namespace string, route AppRoute) error {
	install, err := shared.GetInstall(ctx, r.Client)
	if err != nil {
		return err
	}

	// create httproute
	port := gwv1.PortNumber(int32(route.Service.Port))
	err = r.Client.Create(ctx, &gwv1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      route.Name,
			Namespace: namespace,
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
			Hostnames: resources.GenerateGatewayHostnames(install, route.Name),
			Rules: []gwv1.HTTPRouteRule{
				{
					BackendRefs: []gwv1.HTTPBackendRef{
						{
							BackendRef: gwv1.BackendRef{
								BackendObjectReference: gwv1.BackendObjectReference{
									Name: gwv1.ObjectName(route.Service.Name),
									Port: &port,
								},
							},
						},
					},
				},
			},
		},
	})
	if client.IgnoreAlreadyExists(err) != nil {
		return err
	}

	// annotate service for dns
	service := &corev1.Service{}
	err = r.Client.Get(ctx, types.NamespacedName{
		Name:      route.Service.Name,
		Namespace: namespace,
	}, service)
	if err != nil {
		return err
	}
	service.Annotations[v1.AnnotationDNSHostnames] = resources.GenerateDNSValue(install, route.Name)
	err = r.Client.Update(ctx, service)
	if err != nil {
		return err
	}

	return nil
}

func (r *AppReconciler) deleteRoute(ctx context.Context, namespace string, route string) error {

	// delete httproute
	err := r.Client.Delete(ctx, &gwv1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      route,
			Namespace: namespace,
		},
	})
	if !errors.IsNotFound(err) {
		return err
	}

	return nil
}
