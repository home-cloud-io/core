package host

import (
	"context"

	v1 "github.com/home-cloud-io/core/api/platform/daemon/v1"
	sdConnect "github.com/home-cloud-io/core/api/platform/daemon/v1/v1connect"
	"github.com/home-cloud-io/core/services/platform/daemon/communicate"

	"connectrpc.com/connect"
	"github.com/steady-bytes/draft/pkg/chassis"
)

type (
	Server interface {
		chassis.RPCRegistrar
		sdConnect.HostServiceHandler
	}

	server struct {
		logger chassis.Logger
	}
)

func NewServer(logger chassis.Logger) Server {
	return &server{
		logger: logger,
	}
}

// Implement the `RPCRegistrar` interface of draft so the `grpc` handlers are enabled
func (h *server) RegisterRPC(server chassis.Rpcer) {
	pattern, handler := sdConnect.NewHostServiceHandler(h)
	server.AddHandler(pattern, handler, true)
}

func (h *server) ShutdownAlert(ctx context.Context, _ *connect.Request[v1.ShutdownAlertRequest]) (*connect.Response[v1.ShutdownAlertResponse], error) {
	h.logger.Info("shutdown alert")
	err := communicate.GetClient().Send(&v1.DaemonMessage{
		Message: &v1.DaemonMessage_ShutdownAlert{},
	})
	if err != nil {
		h.logger.WithError(err).Error("failed to send shutdown alert")
		return nil, err
	}
	return &connect.Response[v1.ShutdownAlertResponse]{}, nil
}