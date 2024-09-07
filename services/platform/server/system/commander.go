package system

import (
	"fmt"

	dv1 "github.com/home-cloud-io/core/api/platform/daemon/v1"

	"connectrpc.com/connect"
)

type (
	Commander interface {
		SetStream(stream *connect.BidiStream[dv1.DaemonMessage, dv1.ServerMessage]) error
		CloseStream()
	}

	commander struct {
		stream *connect.BidiStream[dv1.DaemonMessage, dv1.ServerMessage]
	}
)

var (
	com *commander
)

func Init() {
	com = &commander{}
}

func (c *commander) SetStream(stream *connect.BidiStream[dv1.DaemonMessage, dv1.ServerMessage]) error {
	if c.stream != nil {
		return fmt.Errorf("stream already set")
	}
	c.stream = stream
	return nil
}

func (c *commander) CloseStream() {
	c.stream = nil
}
