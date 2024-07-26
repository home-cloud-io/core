package daemon

import (
	"context"
	"time"

	v1 "github.com/home-cloud-io/core/api/platform/daemon/v1"
	sdConnect "github.com/home-cloud-io/core/api/platform/daemon/v1/v1connect"

	"connectrpc.com/connect"
	"github.com/steady-bytes/draft/pkg/chassis"
)

type (
	Rpc interface {
		chassis.RPCRegistrar
		sdConnect.DaemonStreamServiceHandler
		Run()
	}

	rpc struct {
		logger chassis.Logger
		stream *connect.BidiStream[v1.DaemonMessage, v1.ServerMessage]
	}
)

func New(logger chassis.Logger) Rpc {
	return &rpc{
		logger: logger,
	}
}

// Implement the `RPCRegistrar` interface of draft so the `grpc` handlers are enabled
func (h *rpc) RegisterRPC(server chassis.Rpcer) {
	pattern, handler := sdConnect.NewDaemonStreamServiceHandler(h)
	server.AddHandler(pattern, handler, true)
}

func (h *rpc) Communicate(ctx context.Context, stream *connect.BidiStream[v1.DaemonMessage, v1.ServerMessage]) error {
	if h.stream == nil {
		h.stream = stream
	}
	h.logger.Info("establishing stream")
	for {
		_, err := stream.Receive()
		if err != nil {
			h.logger.WithError(err).Error("failed to recieve message")
			// "close" public stream
			h.stream = nil
			return err
		}
		h.logger.Debug("heartbeat received")
		stream.Send(&v1.ServerMessage{
			Message: &v1.ServerMessage_Heartbeat{},
		})
	}
}

func (h *rpc) Run() {
	for {
		if h.stream != nil {
			h.stream.Send(&v1.ServerMessage{
				Message: &v1.ServerMessage_Reboot{},
			})
		} else {
			h.logger.Warn("no stream")
		}
		time.Sleep(1 * time.Second)
		if h.stream != nil {
			h.stream.Send(&v1.ServerMessage{
				Message: &v1.ServerMessage_Shutdown{},
			})
		} else {
			h.logger.Warn("no stream")
		}
		time.Sleep(1 * time.Second)
	}
}
