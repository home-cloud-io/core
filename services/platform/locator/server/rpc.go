package server

import (
	"context"
	"errors"
	"sync"

	v1 "github.com/home-cloud-io/core/api/platform/locator/v1"
	sdConnect "github.com/home-cloud-io/core/api/platform/locator/v1/v1connect"

	"connectrpc.com/connect"
	"github.com/steady-bytes/draft/pkg/chassis"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type (
	Rpc interface {
		chassis.RPCRegistrar
		sdConnect.LocatorServiceHandler
	}

	rpcHandler struct {
		logger   chassis.Logger
		registry Registry
	}
)

func New(logger chassis.Logger) Rpc {
	return &rpcHandler{
		logger:   logger,
		registry: NewRegistry(),
	}
}

// Implement the `RPCRegistrar` interface of draft so the `grpc` handlers are enabled
func (h *rpcHandler) RegisterRPC(server chassis.Rpcer) {
	pattern, handler := sdConnect.NewLocatorServiceHandler(h)
	server.AddHandler(pattern, handler, true)
}

func (h *rpcHandler) Locate(ctx context.Context, request *connect.Request[v1.LocateRequest]) (*connect.Response[v1.LocateResponse], error) {

	s, ok := h.registry.Load(request.Msg.ServerId)
	if !ok {
		return nil, status.Error(codes.NotFound, "server not found")
	}

	resp, err := s.Locate(request.Msg.Body)
	if err != nil {
		return nil, err
	}

	if resp.GetAccept() != nil {
		accept := resp.GetAccept()
		return connect.NewResponse(&v1.LocateResponse{
			Body: accept.Body,
		}), nil
	}

	// TODO: log rejections?

	// anything other than an explicit accept returns an error
	return nil, errors.New("failed to get connection info")
}

func (h *rpcHandler) Register(ctx context.Context, request *connect.Request[v1.RegisterRequest]) (*connect.Response[v1.RegisterResponse], error) {
	// TODO: validate the account token

	return connect.NewResponse(&v1.RegisterResponse{
		RefreshToken: "fake_refresh_token",
		AccessToken:  "fake_access_token",
	}), nil
}

func (h *rpcHandler) Connect(ctx context.Context, stream *connect.BidiStream[v1.ServerMessage, v1.LocatorMessage]) error {
	h.logger.Debug("connect request received")
	msg, err := stream.Receive()
	if err != nil {
		h.logger.Info("server disconnected")
		return err
	}
	// TODO: validate the access token
	switch msg.Body.(type) {
	case *v1.ServerMessage_Initialize:
		init := msg.GetInitialize()
		if err := init.ValidateAll(); err != nil {
			return err
		}
		s := &entry{
			id:             init.ServerId,
			stream:         stream,
			mutex:          sync.Mutex{},
			lookupRequests: sync.Map{},
		}
		h.logger.WithField("server_key", s.id).Debug("server connected")
		h.registry.Store(s)
		defer h.registry.Remove(s.id)
		return s.Listen(h.logger)
	default:
		return status.Error(codes.FailedPrecondition, "initialize must be sent before any other message")
	}
}
