package async

import (
	"fmt"
	"reflect"
	"sync"
)

type (
	// Broadcaster is an interface for broadcasting events to multiple listeners (channels).
	// This is a simple wrapper around go channels to avoid the limitation that channels are only
	// one-to-one pipes. The Broadcaster interface allows for multiple listeners to receive the
	// same message from a single broadcasting channel.
	Broadcaster interface {
		// Send broadcasts the provided message to all registered listeners. This method is non-blocking,
		// and will guarantee that all registered listeners receive the message.
		Send(any)
		// Register registers a channel to receive messages from the broadcaster.
		Register(channel chan any, id string)
		// Deregister deregisters a channel from receiving messages from the broadcaster.
		Deregister(id string)
	}
	broadcaster struct {
		mu          sync.RWMutex
		subscribers map[string]chan any
	}
)

func NewBroadcaster() Broadcaster {
	return &broadcaster{
		subscribers: map[string]chan any{},
	}
}

func (b *broadcaster) Register(c chan any, id string) {
	b.mu.Lock()
	b.subscribers[id] = c
	b.mu.Unlock()
}

func (b *broadcaster) Deregister(id string) {
	b.mu.Lock()
	delete(b.subscribers, id)
	b.mu.Unlock()
}

func (b *broadcaster) Send(m any) {
	go func() {
		b.mu.RLock()
		defer b.mu.RUnlock()
		for _, c := range b.subscribers {
			select {
			case c <- m: // attempt to send message and drop if not able
			default:
				// TODO: convert to logger?
				fmt.Printf("dropping message: %s\n", reflect.TypeOf(m).String())
			}
		}
	}()
}
