package communicate

import (
	"context"

	"connectrpc.com/connect"
	"github.com/steady-bytes/draft/pkg/chassis"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	v1 "github.com/home-cloud-io/core/api/platform/daemon/v1"
	sdConnect "github.com/home-cloud-io/core/api/platform/daemon/v1/v1connect"
	"github.com/home-cloud-io/core/services/platform/daemon/execute"
	"github.com/home-cloud-io/core/services/platform/daemon/host"
)

type (
	Rpc interface {
		chassis.RPCRegistrar
		sdConnect.DaemonServiceHandler
	}

	rpcHandler struct {
		logger          chassis.Logger
		secureTunneling host.SecureTunnelingController
		// actl   apps.Controller
		// sctl   system.Controller
	}
)

func New(logger chassis.Logger, secureTunneling host.SecureTunnelingController) Rpc {
	return &rpcHandler{
		logger,
		secureTunneling,
	}
}

// Implement the `RPCRegistrar` interface of draft so the `grpc` handlers are enabled
func (h *rpcHandler) RegisterRPC(server chassis.Rpcer) {
	pattern, handler := sdConnect.NewDaemonServiceHandler(h)
	server.AddHandler(pattern, handler, true)
}

func (h *rpcHandler) ShutdownHost(ctx context.Context, request *connect.Request[v1.ShutdownHostRequest]) (*connect.Response[v1.ShutdownHostResponse], error) {
	err := execute.Shutdown(ctx, h.logger)
	if err != nil {
		h.logger.WithError(err).Error("failed to shutdown host")
		return nil, err
	}
	return connect.NewResponse(&v1.ShutdownHostResponse{}), nil
}

func (h *rpcHandler) RebootHost(ctx context.Context, request *connect.Request[v1.RebootHostRequest]) (*connect.Response[v1.RebootHostResponse], error) {
	err := execute.Reboot(ctx, h.logger)
	if err != nil {
		h.logger.WithError(err).Error("failed to reboot host")
		return nil, err
	}
	return connect.NewResponse(&v1.RebootHostResponse{}), nil
}

func (h *rpcHandler) InitializeHost(ctx context.Context, request *connect.Request[v1.InitializeHostRequest]) (*connect.Response[v1.InitializeHostResponse], error) {
	// TODO: I think this needs to rotate all certs since I think we'll need to ship an ISO with preloaded certs?
	return connect.NewResponse(&v1.InitializeHostResponse{}), nil
}

func (h *rpcHandler) AddWireguardInterface(ctx context.Context, request *connect.Request[v1.AddWireguardInterfaceRequest]) (*connect.Response[v1.AddWireguardInterfaceResponse], error) {
	publicKey, err := h.secureTunneling.AddInterface(ctx, request.Msg.Interface)
	if err != nil {
		h.logger.WithFields(chassis.Fields{
			"wireguard_interface": request.Msg.Interface.Name,
		}).WithError(err).Error("failed to add wireguard interface")
		return nil, err
	}
	return connect.NewResponse(&v1.AddWireguardInterfaceResponse{
		PublicKey: publicKey,
	}), nil
}

func (h *rpcHandler) RemoveWireguardInterface(ctx context.Context, request *connect.Request[v1.RemoveWireguardInterfaceRequest]) (*connect.Response[v1.RemoveWireguardInterfaceResponse], error) {
	err := h.secureTunneling.RemoveInterface(ctx, request.Msg.Name)
	if err != nil {
		h.logger.WithFields(chassis.Fields{
			"wireguard_interface": request.Msg.Name,
		}).WithError(err).Error("failed to remove wireguard interface")
		return nil, err
	}
	return connect.NewResponse(&v1.RemoveWireguardInterfaceResponse{}), nil
}

func (h *rpcHandler) AddWireguardPeer(ctx context.Context, request *connect.Request[v1.AddWireguardPeerRequest]) (*connect.Response[v1.AddWireguardPeerResponse], error) {
	addresses, dnsServers, err := h.secureTunneling.AddPeer(ctx, request.Msg.WireguardInterface, request.Msg.Peer)
	if err != nil {
		h.logger.WithFields(chassis.Fields{
			"wireguard_interface": request.Msg.WireguardInterface,
			"wireguard_peer":      request.Msg.Peer.PublicKey,
		}).WithError(err).Error("failed to add wireguard peer")
		return nil, err
	}
	return connect.NewResponse(&v1.AddWireguardPeerResponse{
		Addresses:  addresses,
		DnsServers: dnsServers,
	}), nil
}

func (h *rpcHandler) RemoveWireguardPeer(ctx context.Context, request *connect.Request[v1.RemoveWireguardPeerRequest]) (*connect.Response[v1.RemoveWireguardPeerResponse], error) {
	// TODO
	return nil, status.Error(codes.Unimplemented, "unimplemented")
}

func (h *rpcHandler) SetSTUNServer(ctx context.Context, request *connect.Request[v1.SetSTUNServerRequest]) (*connect.Response[v1.SetSTUNServerResponse], error) {
	err := h.secureTunneling.BindSTUNServer(ctx, request.Msg.WireguardInterface, request.Msg.ServerAddress)
	if err != nil {
		h.logger.WithFields(chassis.Fields{
			"wireguard_interface": request.Msg.WireguardInterface,
			"server_address":      request.Msg.ServerAddress,
		}).WithError(err).Error("failed to bind to new stun server")
		return nil, err
	}
	return connect.NewResponse(&v1.SetSTUNServerResponse{}), nil
}

func (h *rpcHandler) AddLocatorServer(ctx context.Context, request *connect.Request[v1.AddLocatorServerRequest]) (*connect.Response[v1.AddLocatorServerResponse], error) {
	err := h.secureTunneling.AddLocator(ctx, request.Msg.WireguardInterface, request.Msg.LocatorAddress)
	if err != nil {
		h.logger.WithFields(chassis.Fields{
			"wireguard_interface": request.Msg.WireguardInterface,
			"server_address":      request.Msg.LocatorAddress,
		}).WithError(err).Error("failed to add locator server")
		return nil, err
	}

	return connect.NewResponse(&v1.AddLocatorServerResponse{}), nil
}

func (h *rpcHandler) RemoveLocatorServer(ctx context.Context, request *connect.Request[v1.RemoveLocatorServerRequest]) (*connect.Response[v1.RemoveLocatorServerResponse], error) {
	err := h.secureTunneling.RemoveLocator(ctx, request.Msg.WireguardInterface, request.Msg.LocatorAddress)
	if err != nil {
		h.logger.WithFields(chassis.Fields{
			"wireguard_interface": request.Msg.WireguardInterface,
			"server_address":      request.Msg.LocatorAddress,
		}).WithError(err).Error("failed to remove locator server")
		return nil, err
	}
	return connect.NewResponse(&v1.RemoveLocatorServerResponse{}), nil
}
