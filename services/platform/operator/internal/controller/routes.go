package controller

import (
	"context"
	"fmt"
	"net/http"

	sv1 "github.com/home-cloud-io/core/api/platform/server/v1"
	sv1Connect "github.com/home-cloud-io/core/api/platform/server/v1/v1connect"

	"connectrpc.com/connect"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"
)

const (
	// TODO: build this from install crd?
	HomeCloudServerAddress = "http://server.home-cloud-system:8090"
	// HomeCloudServerAddress = "http://localhost:8090" // for local dev

	GatewayName = "ingress-gateway"
)

var (
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
						Name:      gwv1.ObjectName(GatewayName),
						Namespace: &GatewayNamespace,
					},
				},
			},
			// TODO: change this to subdomain? (*.home-cloud.local)
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

	// add route (mDNS hostname) to server
	_, err = sv1Connect.NewInternalServiceClient(http.DefaultClient, HomeCloudServerAddress).
		AddMdnsHost(ctx, connect.NewRequest(&sv1.AddMdnsHostRequest{
			Hostname: route.Name,
		}))
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

	// remove route (mDNS hostname) from server
	_, err = sv1Connect.NewInternalServiceClient(http.DefaultClient, HomeCloudServerAddress).
		RemoveMdnsHost(ctx, connect.NewRequest(&sv1.RemoveMdnsHostRequest{
			Hostname: route,
		}))
	if err != nil {
		return err
	}

	return nil
}
