package system

import (
	"context"
	"fmt"
	"sync"

	dv1 "github.com/home-cloud-io/core/api/platform/daemon/v1"
	"github.com/home-cloud-io/core/services/platform/server/async"

	"connectrpc.com/connect"
	"github.com/google/uuid"
)

type (
	Commander interface {
		SetStream(stream *connect.BidiStream[dv1.DaemonMessage, dv1.ServerMessage]) error
		CloseStream()

		Send(request *dv1.ServerMessage) error
		Request(ctx context.Context, request *dv1.ServerMessage, options *async.ListenerOptions[*dv1.DaemonMessage]) (response *dv1.DaemonMessage, err error)
	}

	commander struct {
		mutex       sync.Mutex
		broadcaster async.Broadcaster
		stream      *connect.BidiStream[dv1.DaemonMessage, dv1.ServerMessage]
	}
)

var (
	com         *commander
	ErrNoStream = fmt.Errorf("no stream")
)

func NewCommander(broadcaster async.Broadcaster) {
	com = &commander{
		mutex:       sync.Mutex{},
		broadcaster: broadcaster,
	}
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

func (c *commander) Send(request *dv1.ServerMessage) error {
	if c.stream == nil {
		return ErrNoStream
	}
	c.mutex.Lock()
	err := c.stream.Send(request)
	c.mutex.Unlock()
	return err
}

func (c *commander) Request(ctx context.Context, request *dv1.ServerMessage, options *async.ListenerOptions[*dv1.DaemonMessage]) (response *dv1.DaemonMessage, err error) {
	// the default usecase uses a unique subject to match the request to the response
	if request.Subject == "" {
		request.Subject = uuid.New().String()
	}

	if options == nil {
		options = &async.ListenerOptions[*dv1.DaemonMessage]{}
	}

	if options.Callback == nil {
		// the default callback simply checks if the subject matches
		options.Callback =
			func(event *dv1.DaemonMessage) (bool, error) {
				if event.Subject == request.Subject {
					response = event
					return true, nil
				}
				return false, nil
			}
	}

	listener := async.RegisterListener(ctx, c.broadcaster, options)

	cancelableCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	err = com.Send(request)
	if err != nil {
		return nil, err
	}

	err = listener.Listen(cancelableCtx)
	if err != nil {
		return nil, err
	}

	return response, nil
}
