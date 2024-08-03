package daemon

import (
	"fmt"

	"connectrpc.com/connect"
	v1 "github.com/home-cloud-io/core/api/platform/daemon/v1"
)

type (
	Commander interface {
		SetStream(stream *connect.BidiStream[v1.DaemonMessage, v1.ServerMessage]) error
		CloseStream()

		ShutdownHost() error
		RestartHost() error
	}

	commander struct {
		stream *connect.BidiStream[v1.DaemonMessage, v1.ServerMessage]
	}
)

var (
	commanderSingleton Commander
)

func NewCommander() Commander {
	commanderSingleton = &commander{}
	return commanderSingleton
}

func GetCommander() Commander {
	return commanderSingleton
}

func (c *commander) SetStream(stream *connect.BidiStream[v1.DaemonMessage, v1.ServerMessage]) error {
	if c.stream != nil {
		return fmt.Errorf("stream already set")
	}
	c.stream = stream
	return nil
}

func (c *commander) CloseStream() {
	c.stream = nil
}

func (c *commander) ShutdownHost() error {
	return c.stream.Send(&v1.ServerMessage{
		Message: &v1.ServerMessage_Shutdown{},
	})
}

func (c *commander) RestartHost() error {
	return c.stream.Send(&v1.ServerMessage{
		Message: &v1.ServerMessage_Restart{},
	})
}
