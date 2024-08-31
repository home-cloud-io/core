package kvclient

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"connectrpc.com/connect"
	kvv1 "github.com/steady-bytes/draft/api/core/registry/key_value/v1"
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
	req, err := BuildGetRequest(key, value)
	if err != nil {
		return fmt.Errorf("failed to build get request")
	}

	val, err := clientSingleton.Get(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to get value from key value store")
	}

	if val.Msg.GetValue() == nil {
		return fmt.Errorf("retrieved value is nil")
	}

	if err := val.Msg.GetValue().UnmarshalTo(value); err != nil {
		return fmt.Errorf("failed to unmarshal value")
	}

	return nil
}

// BuildSetRequest is a utility function to create a set request for the key-value store
func BuildSetRequest(key string, value proto.Message) (*connect.Request[kvv1.SetRequest], error) {
	// Create the setting object for the device
	pb, err := anypb.New(value)
	if err != nil {
		return nil, errors.New("failed to build set request")
	}

	set := &kvv1.SetRequest{
		Key:   key,
		Value: pb,
	}

	return connect.NewRequest(set), nil
}

func BuildGetRequest(key string, value proto.Message) (*connect.Request[kvv1.GetRequest], error) {
	pb, err := anypb.New(value)
	if err != nil {
		return nil, errors.New("failed to build get request")
	}

	get := &kvv1.GetRequest{
		Key:   key,
		Value: pb,
	}

	return connect.NewRequest(get), nil
}
