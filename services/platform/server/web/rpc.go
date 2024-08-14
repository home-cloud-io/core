package web

import (
	"context"
	"errors"

	v1 "github.com/home-cloud-io/core/api/platform/server/v1"
	sdConnect "github.com/home-cloud-io/core/api/platform/server/v1/v1connect"
	opv1 "github.com/home-cloud-io/core/services/platform/operator/api/v1"
	"github.com/home-cloud-io/core/services/platform/server/daemon"
	k8sclient "github.com/home-cloud-io/core/services/platform/server/k8s-client"

	"connectrpc.com/connect"
	"github.com/steady-bytes/draft/pkg/chassis"
)

type (
	Rpc interface {
		chassis.RPCRegistrar
		sdConnect.WebServiceHandler
	}

	rpc struct {
		logger    chassis.Logger
		k8sclient k8sclient.Client

		controller Controller
	}
)

func New(logger chassis.Logger) Rpc {
	return &rpc{
		logger:     logger,
		k8sclient:  k8sclient.NewClient(logger),
		controller: NewController(),
	}
}

// Implement the `RPCRegistrar` interface of draft so the `grpc` handlers are enabled
func (h *rpc) RegisterRPC(server chassis.Rpcer) {
	pattern, handler := sdConnect.NewWebServiceHandler(h)
	server.AddHandler(pattern, handler, true)
}

func (h *rpc) ShutdownHost(ctx context.Context, request *connect.Request[v1.ShutdownHostRequest]) (*connect.Response[v1.ShutdownHostResponse], error) {
	commander := daemon.GetCommander()
	err := commander.ShutdownHost()
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&v1.ShutdownHostResponse{}), nil
}

func (h *rpc) RestartHost(ctx context.Context, request *connect.Request[v1.RestartHostRequest]) (*connect.Response[v1.RestartHostResponse], error) {
	commander := daemon.GetCommander()
	err := commander.RestartHost()
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&v1.RestartHostResponse{}), nil
}

func (h *rpc) InstallApp(ctx context.Context, request *connect.Request[v1.InstallAppRequest]) (*connect.Response[v1.InstallAppResponse], error) {
	h.logger.WithField("request", request.Msg).Info("install request")
	err := h.k8sclient.Install(ctx, opv1.AppSpec{
		Chart:   request.Msg.Chart,
		Repo:    request.Msg.Repo,
		Release: request.Msg.Release,
		Values:  request.Msg.Values,
	})
	if err != nil {
		h.logger.WithError(err).Error("failed to install app")
		return nil, err
	}
	h.logger.Info("finished request")
	return connect.NewResponse(&v1.InstallAppResponse{}), nil
}

func (h *rpc) DeleteApp(ctx context.Context, request *connect.Request[v1.DeleteAppRequest]) (*connect.Response[v1.DeleteAppResponse], error) {
	h.logger.WithField("request", request.Msg).Info("delete request")
	err := h.k8sclient.Delete(ctx, opv1.AppSpec{
		Release: request.Msg.Release,
	})
	if err != nil {
		h.logger.WithError(err).Error("failed to delete app")
		return nil, err
	}
	h.logger.Info("finished request")
	return connect.NewResponse(&v1.DeleteAppResponse{}), nil
}

func (h *rpc) UpdateApp(ctx context.Context, request *connect.Request[v1.UpdateAppRequest]) (*connect.Response[v1.UpdateAppResponse], error) {
	h.logger.WithField("request", request.Msg).Info("update request")
	err := h.k8sclient.Update(ctx, opv1.AppSpec{
		Chart:   request.Msg.Chart,
		Repo:    request.Msg.Repo,
		Release: request.Msg.Release,
		Values:  request.Msg.Values,
	})
	if err != nil {
		h.logger.WithError(err).Error("failed to update app")
		return nil, err
	}
	h.logger.Info("finished request")
	return connect.NewResponse(&v1.UpdateAppResponse{}), nil
}

// / RC1 WebApp Api Errors
const (
	ErrFailedToInitDevice = "failed to initialize device"
	ErrInvalidInputValues = "invalid input values"
)

/// RC1 WebApp API

// IsDeviceSetup checks if the device is setup. It's also the first request made by the FE when loading. If the device is not setup, the FE will redirect to the setup page.
// A device is considered setup (or will return true) if the device has a username, password, and the `Settings` object is not empty in `blueprint`.
func (h *rpc) IsDeviceSetup(ctx context.Context, request *connect.Request[v1.IsDeviceSetupRequest]) (*connect.Response[v1.IsDeviceSetupResponse], error) {
	h.logger.Info("checking if device is setup")

	yes, err := h.controller.IsDeviceSetup(ctx)
	if err != nil {
		return nil, errors.New(ErrFailedToGetSettings)
	}

	return connect.NewResponse(&v1.IsDeviceSetupResponse{Setup: yes}), nil
}

func (h *rpc) InitializeDevice(ctx context.Context, request *connect.Request[v1.InitializeDeviceRequest]) (*connect.Response[v1.InitializeDeviceResponse], error) {
	h.logger.Info("setting up device for the first time")

	var msg = request.Msg

	if err := msg.Validate(); err != nil {
		return nil, errors.New(ErrInvalidInputValues)
	}

	// convert the request to the `DeviceSettings` object
	deviceSettings := &v1.DeviceSettings{
		AdminUser: &v1.User{
			Username: msg.GetUsername(),
			Password: msg.GetPassword(),
		},
		Timezone:       msg.GetTimezone(),
		AutoUpdateApps: msg.GetAutoUpdateApps(),
		AutoUpdateOs:   msg.GetAutoUpdateOs(),
	}

	_, err := h.controller.InitializeDevice(ctx, deviceSettings)
	if err != nil {
		return nil, errors.New(ErrFailedToInitDevice)
	}

	return connect.NewResponse(&v1.InitializeDeviceResponse{Setup: true}), nil
}

func (h *rpc) Login(ctx context.Context, request *connect.Request[v1.LoginRequest]) (*connect.Response[v1.LoginResponse], error) {
	return nil, errors.New("not implemented")
}

func (h *rpc) GetDeviceUsageStats(ctx context.Context, request *connect.Request[v1.GetDeviceUsageStatsRequest]) (*connect.Response[v1.GetDeviceUsageStatsResponse], error) {
	return nil, errors.New("not implemented")
}

func (h *rpc) GetInstalledApps(ctx context.Context, request *connect.Request[v1.GetInstalledAppsRequest]) (*connect.Response[v1.GetInstalledAppsResponse], error) {
	return nil, errors.New("not implemented")
}

func (h *rpc) GetAppsInStore(ctx context.Context, request *connect.Request[v1.GetAppsInStoreRequest]) (*connect.Response[v1.GetAppsInStoreResponse], error) {
	return nil, errors.New("not implemented")
}

func (h *rpc) GetDeviceSettings(ctx context.Context, request *connect.Request[v1.GetDeviceSettingsRequest]) (*connect.Response[v1.GetDeviceSettingsResponse], error) {
	return nil, errors.New("not implemented")
}
