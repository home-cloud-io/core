package async

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/proto"
)

const defaultTimeout = 15 * time.Second

type (
	Listener interface {
		Listen(ctx context.Context) error
		Stop()
	}
	listener[T proto.Message] struct {
		events        Broadcaster
		channelID     string
		eventsChannel chan any
		options       *ListenerOptions[T]
	}
	ListenerOptions[T proto.Message] struct {
		// Default: 15 seconds
		Timeout  time.Duration
		Callback func(event T) (done bool, err error)
	}
)

func EventGroup() (*errgroup.Group, Broadcaster) {
	return new(errgroup.Group), NewBroadcaster()
}

func RegisterListener[T proto.Message](ctx context.Context, events Broadcaster, options *ListenerOptions[T]) Listener {
	// set options
	if options == nil {
		options = &ListenerOptions[T]{}
	}
	if options.Timeout == 0 {
		options.Timeout = defaultTimeout
	}

	// create id for broadcaster channels
	id := uuid.New().String()

	// create channel for events
	eventsChan := make(chan any)
	events.Register(eventsChan, id)

	return &listener[T]{
		events:        events,
		channelID:     id,
		eventsChannel: eventsChan,
		options:       options,
	}
}

func (a *listener[T]) Listen(ctx context.Context) error {
	// deregister event listener at end
	defer a.events.Deregister(a.channelID)

	// wait for event
	for {
		select {
		case <-time.After(a.options.Timeout):
			return fmt.Errorf("timed out waiting on event")
		case e := <-a.eventsChannel:
			if e != nil {
				if e, ok := e.(T); ok {
					done, err := a.options.Callback(e)
					if err != nil {
						return err
					}
					if done {
						return nil
					}
				}
			}
		}
	}
}

func (a *listener[T]) Stop() {
	a.events.Deregister(a.channelID)
}