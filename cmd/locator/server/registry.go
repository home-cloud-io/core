package server

import (
	"sync"

	"github.com/google/uuid"
	v1 "github.com/home-cloud-io/core/api/platform/locator/v1"
	"github.com/steady-bytes/draft/pkg/chassis"

	"connectrpc.com/connect"
)

type (
	Registry interface {
		Store(server *entry)
		Load(id string) (server *entry, ok bool)
		Remove(id string)
	}
	registry struct {
		mutex   sync.RWMutex
		servers map[string]*entry
	}
	entry struct {
		id             string
		stream         *connect.BidiStream[v1.ServerMessage, v1.LocatorMessage]
		mutex          sync.Mutex
		lookupRequests sync.Map
	}
)

func NewRegistry() Registry {
	return &registry{
		servers: make(map[string]*entry),
	}
}

func (r *registry) Store(server *entry) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.servers[server.id] = server
}

func (r *registry) Load(id string) (*entry, bool) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	s, ok := r.servers[id]
	if !ok {
		return nil, false
	}
	return s, true
}

func (r *registry) Remove(id string) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	delete(r.servers, id)
}

func (s *entry) Send(msg *v1.LocatorMessage) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.stream.Send(msg)
}

func (s *entry) Locate(body *v1.EncryptedMessage) (*v1.ServerMessage, error) {
	var (
		requestId = uuid.New().String()
		ch        = make(chan *v1.ServerMessage)
	)

	s.lookupRequests.Store(requestId, ch)

	err := s.Send(&v1.LocatorMessage{
		Body: &v1.LocatorMessage_Locate{
			Locate: &v1.Locate{
				RequestId: requestId,
				Body:      body,
			},
		},
	})
	if err != nil {
		return nil, err
	}

	return <-ch, nil
}

func (s *entry) Listen(logger chassis.Logger) error {
	for {
		msg, err := s.stream.Receive()
		if err != nil {
			logger.Info("server disconnected")
			return err
		}
		// TODO: validate the access token
		switch msg.Body.(type) {
		case *v1.ServerMessage_Accept:
			v, ok := s.lookupRequests.LoadAndDelete(msg.GetAccept().RequestId)
			if !ok {
				// TODO
			}
			ch, ok := v.(chan *v1.ServerMessage)
			if !ok {
				// TODO
			}
			ch <- msg
		case *v1.ServerMessage_Reject:
			v, ok := s.lookupRequests.LoadAndDelete(msg.GetReject().RequestId)
			if !ok {
				// TODO
			}
			ch, ok := v.(chan *v1.ServerMessage)
			if !ok {
				// TODO
			}
			ch <- msg
		default:
			// TODO
		}
	}
}
