package controller

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"
)

var (
	// TODO: get this value from the install
	GatewayName      = "ingress-gateway"
	GatewayNamespace = gwv1.Namespace("istio-system")
)

func (r *AppReconciler) createRoute(ctx context.Context, namespace string, route AppRoute) error {

	// create httproute
	port := gwv1.PortNumber(int32(route.Service.Port))
	err := r.Client.Create(ctx, &gwv1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      route.Name,
			Namespace: namespace,
		},
		Spec: gwv1.HTTPRouteSpec{
			CommonRouteSpec: gwv1.CommonRouteSpec{
				ParentRefs: []gwv1.ParentReference{
					{
						// TODO: derive these from Install CRD
						Name:      GatewayName,
						Namespace: &GatewayNamespace,
					},
				},
			},
			// TODO: change this to subdomain? (*.home-cloud.local)
			// subdomains don't work on Windows with mDNS so this would require running our
			// own DNS server (which we want to do anyway)
			Hostnames: []gwv1.Hostname{gwv1.Hostname(fmt.Sprintf("%s.local", route.Name))},
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
