package system

import (
	"fmt"
	"sync"

	dv1 "github.com/home-cloud-io/core/api/platform/daemon/v1"

	"connectrpc.com/connect"
)

type (
	Commander interface {
		SetStream(stream *connect.BidiStream[dv1.DaemonMessage, dv1.ServerMessage]) error
		CloseStream()
	}

	commander struct {
		mutex  sync.Mutex
		stream *connect.BidiStream[dv1.DaemonMessage, dv1.ServerMessage]
	}
)

var (
	com = &commander{
		mutex: sync.Mutex{},
	}
	listeners      = map[string]chan<- *dv1.DaemonMessage{}
	listenersMutex = sync.Mutex{}
	ErrNoStream    = fmt.Errorf("no stream")
)

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

func (c *commander) Send(request *dv1.ServerMessage) error {
	if c.stream == nil {
		return ErrNoStream
	}
	c.mutex.Lock()
	err := c.stream.Send(request)
	c.mutex.Unlock()
	return err
}
