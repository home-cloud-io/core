package web

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/steady-bytes/draft/pkg/chassis"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	v1 "github.com/home-cloud-io/core/api/platform/server/v1"
	sdConnect "github.com/home-cloud-io/core/api/platform/server/v1/v1connect"
	"github.com/home-cloud-io/core/services/platform/server/apps"
	"github.com/home-cloud-io/core/services/platform/server/system"
)

type (
	Rpc interface {
		chassis.RPCRegistrar
		sdConnect.WebServiceHandler
	}

	rpcHandler struct {
		logger chassis.Logger
		actl   apps.Controller
		sctl   system.Controller
	}
)

const (
	ErrFailedToInitDevice       = "failed to initialize device"
	ErrInvalidInputValues       = "invalid input values"
	ErrFailedToLogin            = "failed to login"
	ErrFailedPeerRegistration   = "failed to register peer"
	ErrFailedPeerDeregistration = "failed to deregister peer"
)

func New(logger chassis.Logger, actl apps.Controller, sctl system.Controller) Rpc {
	return &rpcHandler{
		logger,
		actl,
		sctl,
	}
}

// Implement the `RPCRegistrar` interface of draft so the `grpc` handlers are enabled
func (h *rpcHandler) RegisterRPC(server chassis.Rpcer) {
	pattern, handler := sdConnect.NewWebServiceHandler(h)
	server.AddHandler(pattern, handler, true)
}

// APPS

func (h *rpcHandler) InstallApp(ctx context.Context, request *connect.Request[v1.InstallAppRequest]) (*connect.Response[v1.InstallAppResponse], error) {
	h.logger.WithField("request", request.Msg).Info("install request")
	go func() {
		c := context.WithoutCancel(ctx)
		err := h.actl.Install(c, h.logger, request.Msg)
		if err != nil {
			h.logger.WithError(err).Error("failed to install app")
			err := events.Send(&v1.ServerEvent{
				Event: &v1.ServerEvent_Error{
					Error: &v1.ErrorEvent{
						Error: err.Error(),
					},
				},
			})
			if err != nil {
				h.logger.WithError(err).Error("failed to send error event to client")
			}
			return
		}
		h.logger.Info("app finished installing")
		err = events.Send(&v1.ServerEvent{
			Event: &v1.ServerEvent_AppInstalled{
				AppInstalled: &v1.AppInstalledEvent{
					Name: request.Msg.Release,
				},
			},
		})
		if err != nil {
			h.logger.WithError(err).Error("failed to send app installed event to client")
		}
	}()
	h.logger.Info("finished request")
	return connect.NewResponse(&v1.InstallAppResponse{}), nil
}

func (h *rpcHandler) DeleteApp(ctx context.Context, request *connect.Request[v1.DeleteAppRequest]) (*connect.Response[v1.DeleteAppResponse], error) {
	h.logger.WithField("request", request.Msg).Info("delete request")
	err := h.actl.Delete(ctx, h.logger, request.Msg)
	if err != nil {
		h.logger.WithError(err).Error("failed to delete app")
		return nil, err
	}
	h.logger.Info("finished request")
	return connect.NewResponse(&v1.DeleteAppResponse{}), nil
}

func (h *rpcHandler) UpdateApp(ctx context.Context, request *connect.Request[v1.UpdateAppRequest]) (*connect.Response[v1.UpdateAppResponse], error) {
	h.logger.WithField("request", request.Msg).Info("update request")
	err := h.actl.Update(ctx, h.logger, request.Msg)
	if err != nil {
		h.logger.WithError(err).Error("failed to update app")
		return nil, err
	}
	h.logger.Info("finished request")
	return connect.NewResponse(&v1.UpdateAppResponse{}), nil
}

func (h *rpcHandler) AppsHealthCheck(ctx context.Context, request *connect.Request[v1.AppsHealthCheckRequest]) (*connect.Response[v1.AppsHealthCheckResponse], error) {
	checks, err := h.actl.PrettyHealthcheck(ctx, h.logger)
	if err != nil {
		h.logger.WithError(err).Error("failed to check apps health")
		return nil, err
	}
	return connect.NewResponse(&v1.AppsHealthCheckResponse{
		Checks: checks,
	}), nil
}

func (h *rpcHandler) GetAppsInStore(ctx context.Context, request *connect.Request[v1.GetAppsInStoreRequest]) (*connect.Response[v1.GetAppsInStoreResponse], error) {
	h.logger.Info("getting apps in store")

	list, err := h.actl.Store(ctx, h.logger)
	if err != nil {
		h.logger.WithError(err).Error(apps.ErrFailedToGetApps)
		return nil, fmt.Errorf(apps.ErrFailedToGetApps)
	}

	return connect.NewResponse(&v1.GetAppsInStoreResponse{Apps: list}), nil
}

// SYSTEM

func (h *rpcHandler) ShutdownHost(ctx context.Context, request *connect.Request[v1.ShutdownHostRequest]) (*connect.Response[v1.ShutdownHostResponse], error) {
	h.logger.Info("shutdown host request")
	err := h.sctl.ShutdownHost(ctx)
	if err != nil {
		h.logger.WithError(err).Error("failed to shutdown host")
		return nil, err
	}
	return connect.NewResponse(&v1.ShutdownHostResponse{}), nil
}

func (h *rpcHandler) RestartHost(ctx context.Context, request *connect.Request[v1.RestartHostRequest]) (*connect.Response[v1.RestartHostResponse], error) {
	h.logger.Info("restart host request")
	err := h.sctl.RestartHost(ctx)
	if err != nil {
		h.logger.WithError(err).Error("failed to restart host")
		return nil, err
	}
	return connect.NewResponse(&v1.RestartHostResponse{}), nil
}

func (h *rpcHandler) GetSystemStats(ctx context.Context, request *connect.Request[v1.GetSystemStatsRequest]) (*connect.Response[v1.GetSystemStatsResponse], error) {
	stats, err := h.sctl.SystemStats(ctx, h.logger)
	if err != nil {
		h.logger.WithError(err).Error("failed to get system stats")
		return nil, errors.New("failed to get system stats")
	}
	return connect.NewResponse(&v1.GetSystemStatsResponse{
		Stats: stats,
	}), nil
}

func (h *rpcHandler) GetDeviceSettings(ctx context.Context, request *connect.Request[v1.GetDeviceSettingsRequest]) (*connect.Response[v1.GetDeviceSettingsResponse], error) {
	h.logger.Info("getting device settings")

	settings, err := h.sctl.GetServerSettings(ctx, h.logger)
	if err != nil {
		h.logger.WithError(err).Error(system.ErrFailedToGetSettings)
		return nil, errors.New(system.ErrFailedToGetSettings)
	}

	return connect.NewResponse(&v1.GetDeviceSettingsResponse{Settings: settings}), nil
}

func (h *rpcHandler) SetDeviceSettings(ctx context.Context, request *connect.Request[v1.SetDeviceSettingsRequest]) (*connect.Response[v1.SetDeviceSettingsResponse], error) {
	h.logger.Info("setting device settings")

	err := request.Msg.ValidateAll()
	if err != nil {
		return nil, err
	}

	err = h.sctl.SetServerSettings(ctx, h.logger, request.Msg.Settings)
	if err != nil {
		h.logger.WithError(err).Error(system.ErrFailedToSetSettings)
		return nil, fmt.Errorf("%s: %s", system.ErrFailedToSetSettings, err.Error())
	}

	return connect.NewResponse(&v1.SetDeviceSettingsResponse{}), nil
}

func (h *rpcHandler) GetAppStorage(ctx context.Context, request *connect.Request[v1.GetAppStorageRequest]) (*connect.Response[v1.GetAppStorageResponse], error) {
	h.logger.Info("getting app storage")

	appsStorage, err := h.actl.GetAppStorage(ctx, h.logger)
	if err != nil {
		h.logger.WithError(err).Error(apps.ErrFailedToGetAppStorage)
		return nil, errors.New(apps.ErrFailedToGetAppStorage)
	}

	return connect.NewResponse(&v1.GetAppStorageResponse{Apps: appsStorage}), nil
}

func (h *rpcHandler) EnableSecureTunnelling(ctx context.Context, request *connect.Request[v1.EnableSecureTunnellingRequest]) (*connect.Response[v1.EnableSecureTunnellingResponse], error) {
	h.logger.Info("enabling secure tunnelling")

	err := h.sctl.EnableWireguard(ctx, h.logger)
	if err != nil {
		h.logger.WithError(err).Error("failed to enable secure tunnelling")
		return nil, errors.New("failed to enable secure tunnelling")
	}

	return connect.NewResponse(&v1.EnableSecureTunnellingResponse{}), nil
}

func (h *rpcHandler) DisableSecureTunnelling(ctx context.Context, request *connect.Request[v1.DisableSecureTunnellingRequest]) (*connect.Response[v1.DisableSecureTunnellingResponse], error) {
	h.logger.Info("disabling secure tunnelling")

	err := h.sctl.DisableWireguard(ctx, h.logger)
	if err != nil {
		h.logger.WithError(err).Error("failed to disable secure tunnelling")
		return nil, errors.New("failed to disable secure tunnelling")
	}

	return connect.NewResponse(&v1.DisableSecureTunnellingResponse{}), nil
}

func (h *rpcHandler) RegisterPeer(ctx context.Context, request *connect.Request[v1.RegisterPeerRequest]) (*connect.Response[v1.RegisterPeerResponse], error) {
	h.logger.Info("register a peer")

	resp, err := h.sctl.RegisterPeer(ctx, h.logger)
	if err != nil {
		h.logger.WithError(err).Error(ErrFailedPeerRegistration)
		return nil, errors.New(ErrFailedPeerRegistration)
	}

	return connect.NewResponse(resp), nil
}

func (h *rpcHandler) DeregisterPeer(ctx context.Context, request *connect.Request[v1.DeregisterPeerRequest]) (*connect.Response[v1.DeregisterPeerResponse], error) {
	h.logger.Info("deregister a peer")

	err := h.sctl.DeregisterPeer(ctx, h.logger, request.Msg)
	if err != nil {
		h.logger.WithError(err).Error(ErrFailedPeerDeregistration)
		return nil, errors.New(ErrFailedPeerDeregistration)
	}

	return connect.NewResponse(&v1.DeregisterPeerResponse{}), nil
}

func (h *rpcHandler) RegisterToLocator(ctx context.Context, request *connect.Request[v1.RegisterToLocatorRequest]) (*connect.Response[v1.RegisterToLocatorResponse], error) {
	h.logger.WithField("msg", request.Msg).Info("registering to locator")

	if err := request.Msg.ValidateAll(); err != nil {
		h.logger.WithError(err).Error("invalid request")
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	err := h.sctl.AddLocator(ctx, request.Msg.WireguardInterface, request.Msg.LocatorAddress)
	if err != nil {
		h.logger.WithError(err).Error("failed to add locator")
		return nil, fmt.Errorf("failed to add locator")
	}

	return connect.NewResponse(&v1.RegisterToLocatorResponse{}), nil
}

func (h *rpcHandler) DeregisterFromLocator(ctx context.Context, request *connect.Request[v1.DeregisterFromLocatorRequest]) (*connect.Response[v1.DeregisterFromLocatorResponse], error) {
	h.logger.Info("deregistering from locator")

	err := h.sctl.RemoveLocator(ctx, request.Msg.WireguardInterface, request.Msg.LocatorAddress)
	if err != nil {
		h.logger.WithError(err).Error("failed to remove locator")
		return nil, fmt.Errorf("failed to remove locator")
	}

	return connect.NewResponse(&v1.DeregisterFromLocatorResponse{}), nil
}

func (h *rpcHandler) GetComponentVersions(ctx context.Context, request *connect.Request[v1.GetComponentVersionsRequest]) (*connect.Response[v1.GetComponentVersionsResponse], error) {
	h.logger.Info("getting app storage")

	response, err := h.sctl.GetComponentVersions(ctx, h.logger)
	if err != nil {
		h.logger.WithError(err).Error(apps.ErrFailedToGetComponentVersions)
		return nil, errors.New(apps.ErrFailedToGetComponentVersions)
	}

	return connect.NewResponse(response), nil
}

func (h *rpcHandler) GetSystemLogs(ctx context.Context, request *connect.Request[v1.GetSystemLogsRequest]) (*connect.Response[v1.GetSystemLogsResponse], error) {
	h.logger.Info("getting system logs")

	logs, err := h.sctl.GetContainerLogs(ctx, h.logger, int64(request.Msg.SinceSeconds))
	if err != nil {
		h.logger.WithError(err).Error(apps.ErrFailedToGetLogs)
		return nil, errors.New(apps.ErrFailedToGetLogs)
	}

	domainsMap := make(map[string]struct{})
	namespacesMap := make(map[string]struct{})
	sourcesMap := make(map[string]struct{})
	domains := []string{}
	namespaces := []string{}
	sources := []string{}
	for _, log := range logs {
		if _, ok := domainsMap[log.Domain]; !ok {
			domainsMap[log.Domain] = struct{}{}
			domains = append(domains, log.Domain)
		}
		if _, ok := namespacesMap[log.Namespace]; !ok {
			namespacesMap[log.Namespace] = struct{}{}
			namespaces = append(namespaces, log.Namespace)
		}
		if _, ok := sourcesMap[log.Source]; !ok {
			sourcesMap[log.Source] = struct{}{}
			sources = append(sources, log.Source)
		}
	}

	return connect.NewResponse(&v1.GetSystemLogsResponse{
		Logs:       logs,
		Domains:    domains,
		Namespaces: namespaces,
		Sources:    sources,
	}), nil
}

func (h *rpcHandler) Subscribe(ctx context.Context, request *connect.Request[v1.SubscribeRequest], stream *connect.ServerStream[v1.ServerEvent]) error {
	h.logger.Info("establishing client stream")
	id := events.AddStream(stream)
	for {
		err := stream.Send(&v1.ServerEvent{
			Event: &v1.ServerEvent_Heartbeat{},
		})
		if err != nil {
			events.CloseStream(id)
			if err.Error() == "canceled: http2: stream closed" || strings.Contains(err.Error(), "write: broken pipe") {
				h.logger.Info("stream closed by client")
				return nil
			}
			h.logger.WithError(err).Warn("failed to send client heartbeat")
			return err
		}
		time.Sleep(5 * time.Second)
	}
}
