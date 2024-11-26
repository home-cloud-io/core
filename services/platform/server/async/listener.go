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
	}
	listener[T proto.Message] struct {
		broadcaster Broadcaster
		id          string
		ch          chan any
		options     *ListenerOptions[T]
	}
	ListenerOptions[T proto.Message] struct {
		// Timeout specifies the maximum allowed time between received events.
		//
		// Default: 15 seconds
		Timeout time.Duration
		// Callback is the function that will be called when an event matching the given type is broadcast.
		// The Broadcaster will continue sending events to this Listener until either the Callback function
		// returns don=true or returns a non-nil error.
		Callback func(event T) (done bool, err error)
		// Buffer specifies the buffer capacity of the listener. This is the limit of pending messages the
		// Broadcaster will queue before dropping messages. Your Listener is reponsible for keeping up with
		// the queue to avoid dropped messages.
		//
		// Default: 16
		Buffer int
	}
)

func EventGroup() (*errgroup.Group, Broadcaster) {
	return new(errgroup.Group), NewBroadcaster()
}

// RegisterListener will create a new Listener which is registered to receive events of the given type from the provided Broadcaster.
// A subsequent call to Listener.Listen() is required to process events and trigger the provided Callback function.
func RegisterListener[T proto.Message](_ context.Context, broadcaster Broadcaster, options *ListenerOptions[T]) Listener {
	// set options
	if options == nil {
		options = &ListenerOptions[T]{}
	}
	if options.Timeout == 0 {
		options.Timeout = defaultTimeout
	}
	if options.Buffer == 0 {
		options.Buffer = 16
	}

	// create id for broadcaster channels
	id := uuid.New().String()

	// create channel for events
	ch := make(chan any, options.Buffer)
	broadcaster.Register(ch, id)

	return &listener[T]{
		broadcaster: broadcaster,
		id:          id,
		ch:          ch,
		options:     options,
	}
}

func (a *listener[T]) Listen(ctx context.Context) error {
	// deregister event listener at end
	defer a.broadcaster.Deregister(a.id)

	// wait for event
	for {
		select {
		case <-ctx.Done():
			a.broadcaster.Deregister(a.id)
			return nil
		case <-time.After(a.options.Timeout):
			return fmt.Errorf("timed out waiting on event")
		case e := <-a.ch:
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
