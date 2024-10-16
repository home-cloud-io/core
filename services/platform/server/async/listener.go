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
		// Listen will start consuming events and send them to the configured Callback function.
		Listen(ctx context.Context) error
		// Stop will deregister the Listener.
		Stop()
	}
	listener[T proto.Message] struct {
		events        Broadcaster
		channelID     string
		eventsChannel chan any
		options       *ListenerOptions[T]
	}
	ListenerOptions[T proto.Message] struct {
		// Timeout specifies the maximum allowed time between received events.
		//
		// Default: 15 seconds
		Timeout  time.Duration
		// Callback is the function that will be called when an event matching the given type is broadcast.
		// The Broadcaster will continue sending events to this Listener until either the Callback function
		// returns don=true or returns a non-nil error.
		Callback func(event T) (done bool, err error)
	}
)

func EventGroup() (*errgroup.Group, Broadcaster) {
	return new(errgroup.Group), NewBroadcaster()
}

// RegisterListener will create a new Listener which is registered to receive events of the given type from the provided Broadcaster.
// A subsequent call to Listener.Listen() is required to process events and trigger the provided Callback function.
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
