package controller

import (
	"context"
	"net/http"
	"os"

	"connectrpc.com/connect"
	ntv1 "github.com/steady-bytes/draft/api/core/control_plane/networking/v1"
	ntv1Connect "github.com/steady-bytes/draft/api/core/control_plane/networking/v1/v1connect"
	kvv1 "github.com/steady-bytes/draft/api/core/registry/key_value/v1"
	kvv1Connect "github.com/steady-bytes/draft/api/core/registry/key_value/v1/v1connect"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	FuseAddressBlueprintKey = "fuse_address"
)

func (r *AppReconciler) createRoute(ctx context.Context, route *ntv1.Route) error {
	val, err := anypb.New(&kvv1.Value{})
	if err != nil {
		return err
	}

	// get fuse address from blueprint
	// TODO: replace os.Getenv with making this a draft process?
	response, err := kvv1Connect.NewKeyValueServiceClient(http.DefaultClient, os.Getenv("DRAFT_SERVICE_ENTRYPOINT")).
		Get(ctx, connect.NewRequest(&kvv1.GetRequest{
			Key:   FuseAddressBlueprintKey,
			Value: val,
		}))
	if err != nil {
		return err
	}

	// unmarshal value
	value := &kvv1.Value{}
	if err := anypb.UnmarshalTo(response.Msg.GetValue(), value, proto.UnmarshalOptions{}); err != nil {
		return err
	}

	// add route to fuse
	_, err = ntv1Connect.NewNetworkingServiceClient(http.DefaultClient, value.Data).
		AddRoute(ctx, connect.NewRequest(&ntv1.AddRouteRequest{
			Route: route,
		}))
	if err != nil {
		return err
	}

	// create service for mdns
	err = r.Create(ctx, &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      route.Name,
			Namespace: "home-cloud",
		},
		Spec: corev1.ServiceSpec{
			Type:         corev1.ServiceTypeExternalName,
			// TODO: this needs to take into account the node the app is deployed on
			ExternalName: os.Getenv("HOST_IP"),
		},
	})
	if client.IgnoreAlreadyExists(err) != nil {
		return err
	}

	return nil
}
