package web

import (
	"context"

	v1 "github.com/home-cloud-io/core/api/platform/server/v1"
	sdConnect "github.com/home-cloud-io/core/api/platform/server/v1/v1connect"
	"github.com/home-cloud-io/core/services/platform/server/daemon"

	"connectrpc.com/connect"
	"github.com/steady-bytes/draft/pkg/chassis"
)

type (
	Rpc interface {
		chassis.RPCRegistrar
		sdConnect.WebServiceHandler
	}

	rpc struct {
		logger chassis.Logger
	}
)

func New(logger chassis.Logger) Rpc {
	return &rpc{
		logger: logger,
	}
}

// Implement the `RPCRegistrar` interface of draft so the `grpc` handlers are enabled
func (h *rpc) RegisterRPC(server chassis.Rpcer) {
	pattern, handler := sdConnect.NewWebServiceHandler(h)
	server.AddHandler(pattern, handler, true)
}

func (h *rpc) ShutdownHost(ctx context.Context, request *connect.Request[v1.ShutdownHostRequest]) (*connect.Response[v1.ShutdownHostResponse], error) {
	commander := daemon.GetCommander()
	err := commander.ShutdownHost()
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&v1.ShutdownHostResponse{}), nil
}

func (h *rpc) RestartHost(ctx context.Context, request *connect.Request[v1.RestartHostRequest]) (*connect.Response[v1.RestartHostResponse], error) {
	commander := daemon.GetCommander()
	err := commander.RestartHost()
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&v1.RestartHostResponse{}), nil
}
