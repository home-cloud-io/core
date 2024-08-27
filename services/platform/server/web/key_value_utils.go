package web

import (
	"errors"

	"connectrpc.com/connect"
	kvv1 "github.com/steady-bytes/draft/api/core/registry/key_value/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

// buildSetRequest is a utility function to create a set request for the key-value store
func buildSetRequest(key string, value proto.Message) (*connect.Request[kvv1.SetRequest], error) {
	// Create the setting object for the device
	pb, err := anypb.New(value)
	if err != nil {
		return nil, errors.New(ErrFailedToCreateSettings)
	}

	set := &kvv1.SetRequest{
		Key:   key,
		Value: pb,
	}

	return connect.NewRequest(set), nil
}

func buildGetRequest(key string, value proto.Message) (*connect.Request[kvv1.GetRequest], error) {
	pb, err := anypb.New(value)
	if err != nil {
		return nil, errors.New(ErrFailedToCreateSettings)
	}

	get := &kvv1.GetRequest{
		Key:   key,
		Value: pb,
	}

	return connect.NewRequest(get), nil
}
