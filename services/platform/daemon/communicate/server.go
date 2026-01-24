package communicate

import (
	"context"

	"connectrpc.com/connect"
	"github.com/siderolabs/talos/pkg/machinery/client"
	"github.com/steady-bytes/draft/pkg/chassis"

	v1 "github.com/home-cloud-io/core/api/platform/daemon/v1"
	sdConnect "github.com/home-cloud-io/core/api/platform/daemon/v1/v1connect"
	"github.com/home-cloud-io/core/services/platform/daemon/talos"
)

type (
	Rpc interface {
		chassis.RPCRegistrar
		sdConnect.DaemonServiceHandler
	}

	rpcHandler struct {
		logger chassis.Logger
		// TODO: probably want to wrap this in a controller for more complex operations eventually
		client *client.Client
	}
)

func New(logger chassis.Logger) Rpc {
	client, err := talos.Client()
	if err != nil {
		logger.WithError(err).Fatal("failed to create talos client")
	}
	return &rpcHandler{
		logger,
		client,
	}
}

// Implement the `RPCRegistrar` interface of draft so the `grpc` handlers are enabled
func (h *rpcHandler) RegisterRPC(server chassis.Rpcer) {
	pattern, handler := sdConnect.NewDaemonServiceHandler(h)
	server.AddHandler(pattern, handler, true)
}

func (h *rpcHandler) ShutdownHost(ctx context.Context, request *connect.Request[v1.ShutdownHostRequest]) (*connect.Response[v1.ShutdownHostResponse], error) {
	h.logger.Info("shutting down host")
	err := h.client.Shutdown(ctx)
	if err != nil {
		h.logger.WithError(err).Error("failed to shutdown host")
		return nil, err
	}
	return connect.NewResponse(&v1.ShutdownHostResponse{}), nil
}

func (h *rpcHandler) RebootHost(ctx context.Context, request *connect.Request[v1.RebootHostRequest]) (*connect.Response[v1.RebootHostResponse], error) {
	h.logger.Info("rebooting host")
	err := h.client.Reboot(ctx)
	if err != nil {
		h.logger.WithError(err).Error("failed to reboot host")
		return nil, err
	}
	return connect.NewResponse(&v1.RebootHostResponse{}), nil
}
