package internal

import (
	"context"

	v1 "github.com/home-cloud-io/core/api/platform/server/v1"
	sdConnect "github.com/home-cloud-io/core/api/platform/server/v1/v1connect"
	"github.com/home-cloud-io/core/services/platform/server/system"

	"connectrpc.com/connect"
	"github.com/steady-bytes/draft/pkg/chassis"
)

type (
	Rpc interface {
		chassis.RPCRegistrar
		sdConnect.InternalServiceHandler
	}

	rpc struct {
		logger chassis.Logger
		// actl   apps.Controller
		sctl   system.Controller
	}
)

const (
	ErrFailedToInitDevice = "failed to initialize device"
	ErrInvalidInputValues = "invalid input values"
	ErrFailedToLogin      = "failed to login"
)

func New(logger chassis.Logger, sctl system.Controller) Rpc {
	return &rpc{
		logger,
		sctl,
	}
}

// Implement the `RPCRegistrar` interface of draft so the `grpc` handlers are enabled
func (h *rpc) RegisterRPC(server chassis.Rpcer) {
	pattern, handler := sdConnect.NewInternalServiceHandler(h)
	server.AddHandler(pattern, handler, true)
}

func (h *rpc) AddMdnsHost(ctx context.Context, request *connect.Request[v1.AddMdnsHostRequest]) (*connect.Response[v1.AddMdnsHostResponse], error) {
	err := h.sctl.AddMdnsHost(request.Msg.Hostname)
	if err != nil {
		h.logger.WithError(err).Error("failed to add mDNS host")
		return nil, err
	}
	return connect.NewResponse(&v1.AddMdnsHostResponse{}), nil
}

func (h *rpc) RemoveMdnsHost(ctx context.Context, request *connect.Request[v1.RemoveMdnsHostRequest]) (*connect.Response[v1.RemoveMdnsHostResponse], error) {
	err := h.sctl.RemoveMdnsHost(request.Msg.Hostname)
	if err != nil {
		h.logger.WithError(err).Error("failed to remove mDNS host")
		return nil, err
	}
	return connect.NewResponse(&v1.RemoveMdnsHostResponse{}), nil
}
