package controller

import (
	"context"
	"net/http"
	"os"

	sv1 "github.com/home-cloud-io/core/api/platform/server/v1"
	sv1Connect "github.com/home-cloud-io/core/api/platform/server/v1/v1connect"

	"connectrpc.com/connect"
	ntv1 "github.com/steady-bytes/draft/api/core/control_plane/networking/v1"
	ntv1Connect "github.com/steady-bytes/draft/api/core/control_plane/networking/v1/v1connect"
	kvv1 "github.com/steady-bytes/draft/api/core/registry/key_value/v1"
	kvv1Connect "github.com/steady-bytes/draft/api/core/registry/key_value/v1/v1connect"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

const (
	FuseAddressBlueprintKey = "fuse_address"
	// TODO: retrieve this from blueprint
	HomeCloudServerAddress = "server.home-cloud-system.svc.cluster.local:8090"
)

func (r *AppReconciler) createRoute(ctx context.Context, route *ntv1.Route) error {
	fuseAddress, err := getFuseAddress(ctx)
	if err != nil {
		return err
	}

	// add route to fuse
	_, err = ntv1Connect.NewNetworkingServiceClient(http.DefaultClient, fuseAddress).
		AddRoute(ctx, connect.NewRequest(&ntv1.AddRouteRequest{
			Route: route,
		}))
	if err != nil {
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

func (r *AppReconciler) deleteRoute(ctx context.Context, route string) error {
	fuseAddress, err := getFuseAddress(ctx)
	if err != nil {
		return err
	}

	// delete route from fuse
	_, err = ntv1Connect.NewNetworkingServiceClient(http.DefaultClient, fuseAddress).
		DeleteRoute(ctx, connect.NewRequest(&ntv1.DeleteRouteRequest{
			Name: route,
		}))
	if err != nil {
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

func getFuseAddress(ctx context.Context) (string, error) {
	val, err := anypb.New(&kvv1.Value{})
	if err != nil {
		return "", err
	}

	// get fuse address from blueprint
	// TODO: replace os.Getenv with making this a draft process?
	response, err := kvv1Connect.NewKeyValueServiceClient(http.DefaultClient, os.Getenv("DRAFT_SERVICE_ENTRYPOINT")).
		Get(ctx, connect.NewRequest(&kvv1.GetRequest{
			Key:   FuseAddressBlueprintKey,
			Value: val,
		}))
	if err != nil {
		return "", err
	}

	// unmarshal value
	value := &kvv1.Value{}
	if err := anypb.UnmarshalTo(response.Msg.GetValue(), value, proto.UnmarshalOptions{}); err != nil {
		return "", err
	}

	return value.Data, nil
}
