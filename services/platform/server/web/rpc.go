package web

import (
	"context"
	"errors"
	"fmt"
	"time"

	dv1 "github.com/home-cloud-io/core/api/platform/daemon/v1"
	v1 "github.com/home-cloud-io/core/api/platform/server/v1"
	sdConnect "github.com/home-cloud-io/core/api/platform/server/v1/v1connect"
	"github.com/home-cloud-io/core/services/platform/server/apps"
	"github.com/home-cloud-io/core/services/platform/server/system"

	"connectrpc.com/connect"
	"github.com/steady-bytes/draft/pkg/chassis"
)

type (
	Rpc interface {
		chassis.RPCRegistrar
		sdConnect.WebServiceHandler
	}

	rpc struct {
		logger chassis.Logger
		actl   apps.Controller
		sctl   system.Controller
	}
)

const (
	ErrFailedToInitDevice = "failed to initialize device"
	ErrInvalidInputValues = "invalid input values"
	ErrFailedToLogin      = "failed to login"
)

func New(logger chassis.Logger, actl apps.Controller, sctl system.Controller) Rpc {
	return &rpc{
		logger,
		actl,
		sctl,
	}
}

// Implement the `RPCRegistrar` interface of draft so the `grpc` handlers are enabled
func (h *rpc) RegisterRPC(server chassis.Rpcer) {
	pattern, handler := sdConnect.NewWebServiceHandler(h)
	server.AddHandler(pattern, handler, true)
}

// APPS

func (h *rpc) InstallApp(ctx context.Context, request *connect.Request[v1.InstallAppRequest]) (*connect.Response[v1.InstallAppResponse], error) {
	h.logger.WithField("request", request.Msg).Info("install request")
	go func(){
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

func (h *rpc) DeleteApp(ctx context.Context, request *connect.Request[v1.DeleteAppRequest]) (*connect.Response[v1.DeleteAppResponse], error) {
	h.logger.WithField("request", request.Msg).Info("delete request")
	err := h.actl.Delete(ctx, h.logger, request.Msg)
	if err != nil {
		h.logger.WithError(err).Error("failed to delete app")
		return nil, err
	}
	h.logger.Info("finished request")
	return connect.NewResponse(&v1.DeleteAppResponse{}), nil
}

func (h *rpc) UpdateApp(ctx context.Context, request *connect.Request[v1.UpdateAppRequest]) (*connect.Response[v1.UpdateAppResponse], error) {
	h.logger.WithField("request", request.Msg).Info("update request")
	err := h.actl.Update(ctx, h.logger, request.Msg)
	if err != nil {
		h.logger.WithError(err).Error("failed to update app")
		return nil, err
	}
	h.logger.Info("finished request")
	return connect.NewResponse(&v1.UpdateAppResponse{}), nil
}

func (h *rpc) AppsHealthCheck(ctx context.Context, request *connect.Request[v1.AppsHealthCheckRequest]) (*connect.Response[v1.AppsHealthCheckResponse], error) {
	checks, err := h.actl.Healthcheck(ctx, h.logger)
	if err != nil {
		h.logger.WithError(err).Error("failed to check apps health")
		return nil, err
	}
	return connect.NewResponse(&v1.AppsHealthCheckResponse{
		Checks: checks,
	}), nil
}

func (h *rpc) GetAppsInStore(ctx context.Context, request *connect.Request[v1.GetAppsInStoreRequest]) (*connect.Response[v1.GetAppsInStoreResponse], error) {
	h.logger.Info("getting apps in store")

	list, err := h.actl.Store(ctx, h.logger)
	if err != nil {
		h.logger.WithError(err).Error(apps.ErrFailedToGetApps)
		return nil, fmt.Errorf(apps.ErrFailedToGetApps)
	}

	return connect.NewResponse(&v1.GetAppsInStoreResponse{Apps: list}), nil
}

// SYSTEM

func (h *rpc) ShutdownHost(ctx context.Context, request *connect.Request[v1.ShutdownHostRequest]) (*connect.Response[v1.ShutdownHostResponse], error) {
	err := h.sctl.ShutdownHost()
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&v1.ShutdownHostResponse{}), nil
}

func (h *rpc) RestartHost(ctx context.Context, request *connect.Request[v1.RestartHostRequest]) (*connect.Response[v1.RestartHostResponse], error) {
	err := h.sctl.RestartHost()
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&v1.RestartHostResponse{}), nil
}

func (h *rpc) CheckForSystemUpdates(ctx context.Context, request *connect.Request[v1.CheckForSystemUpdatesRequest]) (*connect.Response[v1.CheckForSystemUpdatesResponse], error) {
	h.logger.Info("check for system updates request")
	response, err := h.sctl.CheckForOSUpdates(ctx, h.logger)
	if err != nil {
		h.logger.WithError(err).Error("failed to check for system updates")
		return nil, err
	}
	return connect.NewResponse(response), nil
}

func (h *rpc) CheckForContainerUpdates(ctx context.Context, request *connect.Request[v1.CheckForContainerUpdatesRequest]) (*connect.Response[v1.CheckForContainerUpdatesResponse], error) {
	images, err := h.sctl.CheckForContainerUpdates(ctx, h.logger)
	if err != nil {
		h.logger.WithError(err).Error("failed to check for system container updates")
		return nil, err
	}
	return connect.NewResponse(&v1.CheckForContainerUpdatesResponse{
		ImageVersions: images,
	}), err
}

func (h *rpc) ChangeDaemonVersion(ctx context.Context, request *connect.Request[v1.ChangeDaemonVersionRequest]) (*connect.Response[v1.ChangeDaemonVersionResponse], error) {
	err := h.sctl.ChangeDaemonVersion(&dv1.ChangeDaemonVersionCommand{
		Version:    request.Msg.Version,
		SrcHash:    request.Msg.SrcHash,
		VendorHash: request.Msg.VendorHash,
	})
	if err != nil {
		h.logger.WithError(err).Error("failed to change daemon version")
		return nil, err
	}
	return connect.NewResponse(&v1.ChangeDaemonVersionResponse{}), nil
}

func (h *rpc) InstallOSUpdate(ctx context.Context, request *connect.Request[v1.InstallOSUpdateRequest]) (*connect.Response[v1.InstallOSUpdateResponse], error) {
	err := h.sctl.InstallOSUpdate()
	if err != nil {
		h.logger.WithError(err).Error("failed to change install os update")
		return nil, err
	}
	return connect.NewResponse(&v1.InstallOSUpdateResponse{}), nil
}

func (h *rpc) SetSystemImage(ctx context.Context, request *connect.Request[v1.SetSystemImageRequest]) (*connect.Response[v1.SetSystemImageResponse], error) {
	err := h.sctl.SetSystemImage(&dv1.SetSystemImageCommand{
		CurrentImage:   request.Msg.CurrentImage,
		RequestedImage: request.Msg.RequestedImage,
	})
	if err != nil {
		h.logger.WithError(err).Error("failed to set system image")
		return nil, err
	}
	return connect.NewResponse(&v1.SetSystemImageResponse{}), nil
}

func (h *rpc) GetSystemStats(ctx context.Context, request *connect.Request[v1.GetSystemStatsRequest]) (*connect.Response[v1.GetSystemStatsResponse], error) {
	// grab the in-memory cache of current system stats
	stats := system.CurrentStats
	if stats == nil {
		h.logger.Error("failed to get system stats")
		return nil, errors.New("failed to get system stats")
	}
	return connect.NewResponse(&v1.GetSystemStatsResponse{
		Stats: stats,
	}), nil
}

// IsDeviceSetup checks if the device is setup. It's also the first request made by the FE when loading. If the device is not setup, the FE will redirect to the setup page.
// A device is considered setup (or will return true) if the device has a username, password, and the `Settings` object is not empty in `blueprint`.
func (h *rpc) IsDeviceSetup(ctx context.Context, request *connect.Request[v1.IsDeviceSetupRequest]) (*connect.Response[v1.IsDeviceSetupResponse], error) {
	h.logger.Info("checking if device is setup")
	isSetup, err := h.sctl.IsDeviceSetup(ctx)
	if err != nil {
		return nil, fmt.Errorf(system.ErrFailedToGetSettings)
	}

	return connect.NewResponse(&v1.IsDeviceSetupResponse{Setup: isSetup}), nil
}

func (h *rpc) InitializeDevice(ctx context.Context, request *connect.Request[v1.InitializeDeviceRequest]) (*connect.Response[v1.InitializeDeviceResponse], error) {
	h.logger.Info("requested to set up device for the first time")

	var msg = request.Msg
	if err := msg.ValidateAll(); err != nil {
		return nil, err
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

	err := h.sctl.InitializeDevice(ctx, h.logger, deviceSettings)
	if err != nil {
		if err.Error() == system.ErrDeviceAlreadySetup {
			return connect.NewResponse(&v1.InitializeDeviceResponse{Setup: true}), nil
		}

		h.logger.Error(err.Error())
		return nil, err
	}

	return connect.NewResponse(&v1.InitializeDeviceResponse{Setup: true}), nil
}

func (h *rpc) Login(ctx context.Context, request *connect.Request[v1.LoginRequest]) (*connect.Response[v1.LoginResponse], error) {
	h.logger.Info("login request")

	msg := request.Msg
	if err := msg.Validate(); err != nil {
		h.logger.WithError(err).Error(ErrInvalidInputValues)
		return nil, fmt.Errorf(ErrInvalidInputValues)
	}

	res, err := h.sctl.Login(ctx, msg.GetUsername(), msg.GetPassword())
	if err != nil {
		h.logger.WithError(err).Error(ErrFailedToLogin)
		return nil, fmt.Errorf(ErrFailedToLogin)
	}

	return connect.NewResponse(&v1.LoginResponse{Token: res}), nil
}

func (h *rpc) GetDeviceSettings(ctx context.Context, request *connect.Request[v1.GetDeviceSettingsRequest]) (*connect.Response[v1.GetDeviceSettingsResponse], error) {
	h.logger.Info("getting device settings")

	settings, err := h.sctl.GetServerSettings(ctx)
	if err != nil {
		h.logger.WithError(err).Error(system.ErrFailedToGetSettings)
		return nil, errors.New(system.ErrFailedToGetSettings)
	}

	return connect.NewResponse(&v1.GetDeviceSettingsResponse{Settings: settings}), nil
}

func (h *rpc) Subscribe(ctx context.Context, request *connect.Request[v1.SubscribeRequest], stream *connect.ServerStream[v1.ServerEvent]) error {
	err := events.SetStream(stream)
	if err != nil {
		return err
	}
	h.logger.Info("establishing client stream")
	for {
		err := stream.Send(&v1.ServerEvent{
			Event: &v1.ServerEvent_Heartbeat{},
		})
		if err != nil {
			if err.Error() == "canceled: http2: stream closed" {
				h.logger.Info("stream closed by client")
				return nil
			}
			h.logger.WithError(err).Error("failed to send client heartbeat")
			return err
		}
		time.Sleep(5 * time.Second)
	}
}
