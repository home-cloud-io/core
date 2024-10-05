package web

import (
	"strings"
	"sync"

	v1 "github.com/home-cloud-io/core/api/platform/server/v1"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
)

type (
	Eventer interface {
		AddStream(stream *connect.ServerStream[v1.ServerEvent]) string
		CloseStream(id string)
	}

	eventer struct {
		mutex   sync.Mutex
		streams map[string]*connect.ServerStream[v1.ServerEvent]
	}
)

var (
	events = &eventer{
		mutex: sync.Mutex{},
		streams: map[string]*connect.ServerStream[v1.ServerEvent]{},
	}
)

func (c *eventer) AddStream(stream *connect.ServerStream[v1.ServerEvent]) string {
	id := uuid.New().String()
	c.mutex.Lock()
	c.streams[id] = stream
	c.mutex.Unlock()
	return id
}

func (c *eventer) CloseStream(id string) {
	c.mutex.Lock()
	delete(c.streams, id)
	c.mutex.Unlock()
}

func (c *eventer) Send(event *v1.ServerEvent) error {
	g := errgroup.Group{}
	c.mutex.Lock()
	defer c.mutex.Unlock()
	for id, stream := range c.streams {
		g.Go(func() error {
			err := stream.Send(event)
			if err != nil {
				if err.Error() == "canceled: http2: stream closed" || strings.Contains(err.Error(), "write: broken pipe") {
					// close stream in background since we have to wait on mutex unlocks
					go c.CloseStream(id)
					return nil
				}
			}
			return err
		})
	}
	return g.Wait()
}
