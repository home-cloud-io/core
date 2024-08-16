package daemon

import (
	"context"

	v1 "github.com/home-cloud-io/core/api/platform/daemon/v1"
	sdConnect "github.com/home-cloud-io/core/api/platform/daemon/v1/v1connect"

	"connectrpc.com/connect"
	"github.com/steady-bytes/draft/pkg/chassis"
)

type (
	Rpc interface {
		chassis.RPCRegistrar
		sdConnect.DaemonStreamServiceHandler
	}

	rpc struct {
		logger    chassis.Logger
		commander Commander
		messages  chan *v1.DaemonMessage
	}
)

func New(logger chassis.Logger, messages chan *v1.DaemonMessage) Rpc {
	return &rpc{
		logger:    logger,
		commander: NewCommander(),
		messages:  messages,
	}
}

func (h *rpc) RegisterRPC(server chassis.Rpcer) {
	pattern, handler := sdConnect.NewDaemonStreamServiceHandler(h)
	server.AddHandler(pattern, handler, true)
}

func (h *rpc) Communicate(ctx context.Context, stream *connect.BidiStream[v1.DaemonMessage, v1.ServerMessage]) error {
	err := h.commander.SetStream(stream)
	if err != nil {
		return err
	}
	h.logger.Info("establishing stream")
	for {
		message, err := stream.Receive()
		if err != nil {
			h.logger.WithError(err).Error("failed to recieve message")
			h.commander.CloseStream()
			return err
		}
		switch message.Message.(type) {
		case *v1.DaemonMessage_ShutdownAlert:
			h.logger.Info("shutdown alert")
		case *v1.DaemonMessage_Heartbeat:
			h.logger.Debug("heartbeat received")
		case *v1.DaemonMessage_OsUpdateDiff:
			h.messages <- message
		case *v1.DaemonMessage_CurrentDaemonVersion:
			h.messages <- message
		default:
			h.logger.WithField("message", message).Warn("unknown message type received")
		}
		stream.Send(&v1.ServerMessage{
			Message: &v1.ServerMessage_Heartbeat{},
		})
	}
}
