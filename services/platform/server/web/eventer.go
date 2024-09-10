package web

import (
	"fmt"

	v1 "github.com/home-cloud-io/core/api/platform/server/v1"

	"connectrpc.com/connect"
)

type (
	Eventer interface {
		SetStream(stream *connect.ServerStream[v1.ServerEvent]) error
		CloseStream()
	}

	eventer struct {
		stream *connect.ServerStream[v1.ServerEvent]
	}
)

var (
	events = &eventer{}
)

func (c *eventer) SetStream(stream *connect.ServerStream[v1.ServerEvent]) error {
	if c.stream != nil {
		return fmt.Errorf("stream already set")
	}
	c.stream = stream
	return nil
}

func (c *eventer) CloseStream() {
	c.stream = nil
}

func (c *eventer) Send(event *v1.ServerEvent) error {
	if c.stream == nil {
		return nil
	}
	return c.stream.Send(event)
}
