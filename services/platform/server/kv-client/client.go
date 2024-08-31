package kvclient

import (
	"context"
	"fmt"
	"net/http"

	"connectrpc.com/connect"
	v1 "github.com/steady-bytes/draft/api/core/registry/key_value/v1"
	kvv1Connect "github.com/steady-bytes/draft/api/core/registry/key_value/v1/v1connect"
	"github.com/steady-bytes/draft/pkg/chassis"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

var (
	clientSingleton kvv1Connect.KeyValueServiceClient
)

const (
	APP_STORE_ENTRIES_KEY       = "app_store_entries"
	DEFAULT_DEVICE_SETTINGS_KEY = "device"
	SEED_KEY                    = "secret_seed"
)

func Init() {
	clientSingleton = kvv1Connect.NewKeyValueServiceClient(http.DefaultClient, chassis.GetConfig().Entrypoint())
}

func Get(ctx context.Context, key string, value proto.Message) error {
	// convert value to request
	pb, err := anypb.New(value)
	if err != nil {
		return err
	}
	get := &v1.GetRequest{
		Key:   key,
		Value: pb,
	}

	// issue request
	val, err := clientSingleton.Get(ctx, connect.NewRequest(get))
	if err != nil {
		return err
	}

	// convert response to value
	if val.Msg.GetValue() == nil {
		return fmt.Errorf("retrieved value is nil")
	}
	if err := val.Msg.GetValue().UnmarshalTo(value); err != nil {
		return err
	}

	return nil
}

func Set(ctx context.Context, key string, value proto.Message) (string, error) {
	// convert value to request
	pb, err := anypb.New(value)
	if err != nil {
		return "", err
	}
	set := &v1.SetRequest{
		Key:   key,
		Value: pb,
	}

	// issue request
	resp, err := clientSingleton.Set(ctx, connect.NewRequest(set))
	if err != nil {
		return "", err
	}

	return resp.Msg.Key, nil
}

func List(ctx context.Context, t proto.Message) (map[string]*anypb.Any, error) {
	// convert value to request
	pb, err := anypb.New(t)
	if err != nil {
		return nil, err
	}
	list := &v1.ListRequest{
		Value: pb,
	}
	resp, err := clientSingleton.List(ctx, connect.NewRequest(list))
	if err != nil {
		return nil, err
	}

	return resp.Msg.Values, nil
}
