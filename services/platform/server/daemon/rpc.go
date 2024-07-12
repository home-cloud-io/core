package daemon

import (
	"context"

	v1 "github.com/home-cloud-io/core/api/core/daemon/v1"
	sdConnect "github.com/home-cloud-io/core/api/core/daemon/v1/v1connect"
	"github.com/steady-bytes/draft/pkg/chassis"

	"connectrpc.com/connect"
)

type (
	Rpc interface {
		chassis.RPCRegistrar
		sdConnect.DaemonStreamServiceHandler
	}

	rpc struct {
		logger chassis.Logger
	}
)

func New() Rpc {
	return &rpc{}
}

// Implement the `RPCRegistrar` interface of draft so the `grpc` handlers are enabled
func (h *rpc) RegisterRPC(server chassis.Rpcer) {
	server.EnableReflection(sdConnect.DaemonStreamServiceName)
	server.AddHandler(sdConnect.NewDaemonStreamServiceHandler(h))
	h.logger = server.Logger()
}

func (h *rpc) Communicate(ctx context.Context, stream *connect.BidiStream[v1.DaemonMessage, v1.ServerMessage]) error {
	h.logger.Info("establishing stream")
	for {
		_, err := stream.Receive()
		if err != nil {
			h.logger.WithError(err).Error("failed to recieve message")
			return err
		}
		h.logger.Info("heartbeat received")
		stream.Send(&v1.ServerMessage{
			Message: &v1.ServerMessage_Heartbeat{},
		})
	}
}
