package system

import (
	"context"

	v1 "github.com/home-cloud-io/core/api/platform/daemon/v1"
	sdConnect "github.com/home-cloud-io/core/api/platform/daemon/v1/v1connect"
	"github.com/home-cloud-io/core/services/platform/server/async"

	"connectrpc.com/connect"
	"github.com/steady-bytes/draft/pkg/chassis"
)

type (
	Server interface {
		chassis.RPCRegistrar
		sdConnect.DaemonStreamServiceHandler
	}

	server struct {
		logger      chassis.Logger
		broadcaster async.Broadcaster
	}
)

var CurrentStats *v1.SystemStats

func New(logger chassis.Logger, broadcaster async.Broadcaster) Server {
	return &server{
		logger:      logger,
		broadcaster: broadcaster,
	}
}

func (h *server) RegisterRPC(server chassis.Rpcer) {
	pattern, handler := sdConnect.NewDaemonStreamServiceHandler(h)
	server.AddHandler(pattern, handler, true)
}

func (h *server) Communicate(ctx context.Context, stream *connect.BidiStream[v1.DaemonMessage, v1.ServerMessage]) error {
	err := com.SetStream(stream)
	if err != nil {
		return err
	}
	h.logger.Info("establishing daemon stream")
	for {
		if ctx.Err() != nil {
			h.logger.WithError(err).Error("stream error")
			com.CloseStream()
			return err
		}

		message, err := stream.Receive()
		if err != nil {
			h.logger.WithError(err).Error("failed to recieve message")
			com.CloseStream()
			return err
		}
		switch message.Message.(type) {
		case *v1.DaemonMessage_ShutdownAlert:
			h.logger.Info("shutdown alert")
		case *v1.DaemonMessage_Heartbeat:
			h.logger.Debug("heartbeat received")
			err = stream.Send(&v1.ServerMessage{
				Message: &v1.ServerMessage_Heartbeat{},
			})
			if err != nil {
				h.logger.WithError(err).Error("failed to send heartbeat")
			}
		case *v1.DaemonMessage_OsUpdateDiff:
			h.broadcaster.Send(message.GetOsUpdateDiff())
		case *v1.DaemonMessage_CurrentDaemonVersion:
			h.broadcaster.Send(message.GetCurrentDaemonVersion())
		case *v1.DaemonMessage_DeviceInitialized:
			h.broadcaster.Send(message.GetDeviceInitialized())
		case *v1.DaemonMessage_UploadFileReady:
			h.broadcaster.Send(message.GetUploadFileReady())
		case *v1.DaemonMessage_UploadFileChunkCompleted:
			h.broadcaster.Send(message.GetUploadFileChunkCompleted())
		case *v1.DaemonMessage_SystemStats:
			CurrentStats = message.GetSystemStats()
		case *v1.DaemonMessage_SettingsSaved:
			h.broadcaster.Send(message.GetSettingsSaved())
		case *v1.DaemonMessage_WireguardInterfaceAdded:
			h.broadcaster.Send(message.GetWireguardInterfaceAdded())
		case *v1.DaemonMessage_WireguardInterfaceRemoved:
			h.broadcaster.Send(message.GetWireguardInterfaceRemoved())
		case *v1.DaemonMessage_StunServerSet:
			h.broadcaster.Send(message.GetStunServerSet())
		case *v1.DaemonMessage_LocatorServerAdded:
			h.broadcaster.Send(message.GetLocatorServerAdded())
		case *v1.DaemonMessage_LocatorServerRemoved:
			h.broadcaster.Send(message.GetLocatorServerRemoved())
		case *v1.DaemonMessage_AllLocatorsDisabled:
			h.broadcaster.Send(message.GetAllLocatorsDisabled())
		case *v1.DaemonMessage_ComponentVersions:
			h.broadcaster.Send(message.GetComponentVersions())
		case *v1.DaemonMessage_Logs:
			h.broadcaster.Send(message.GetLogs())
		default:
			h.logger.WithField("message", message).Warn("unknown message type received")
		}
	}
}
