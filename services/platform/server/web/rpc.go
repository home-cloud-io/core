package web

import (
	"context"
	"errors"
	"fmt"

	dv1 "github.com/home-cloud-io/core/api/platform/daemon/v1"
	v1 "github.com/home-cloud-io/core/api/platform/server/v1"
	sdConnect "github.com/home-cloud-io/core/api/platform/server/v1/v1connect"
	"github.com/home-cloud-io/core/services/platform/server/daemon"
	"github.com/home-cloud-io/core/services/platform/server/system"
	"github.com/home-cloud-io/core/services/platform/server/versioning"

	"connectrpc.com/connect"
	"github.com/steady-bytes/draft/pkg/chassis"
)

type (
	Rpc interface {
		chassis.RPCRegistrar
		sdConnect.WebServiceHandler
	}

	rpc struct {
		logger     chassis.Logger
		controller system.Controller
		updater    versioning.Updater
	}
)

func New(logger chassis.Logger, controller system.Controller, updater versioning.Updater) Rpc {
	return &rpc{
		logger:     logger,
		controller: controller,
		updater:    updater,
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
	err := h.controller.InstallApp(ctx, h.logger, request.Msg)
	if err != nil {
		h.logger.WithError(err).Error("failed to install app")
		return nil, err
	}
	h.logger.Info("finished request")
	return connect.NewResponse(&v1.InstallAppResponse{}), nil
}

func (h *rpc) DeleteApp(ctx context.Context, request *connect.Request[v1.DeleteAppRequest]) (*connect.Response[v1.DeleteAppResponse], error) {
	h.logger.WithField("request", request.Msg).Info("delete request")
	err := h.controller.DeleteApp(ctx, h.logger, request.Msg)
	if err != nil {
		h.logger.WithError(err).Error("failed to delete app")
		return nil, err
	}
	h.logger.Info("finished request")
	return connect.NewResponse(&v1.DeleteAppResponse{}), nil
}

func (h *rpc) UpdateApp(ctx context.Context, request *connect.Request[v1.UpdateAppRequest]) (*connect.Response[v1.UpdateAppResponse], error) {
	h.logger.WithField("request", request.Msg).Info("update request")
	err := h.controller.UpdateApp(ctx, h.logger, request.Msg)
	if err != nil {
		h.logger.WithError(err).Error("failed to update app")
		return nil, err
	}
	h.logger.Info("finished request")
	return connect.NewResponse(&v1.UpdateAppResponse{}), nil
}

func (h *rpc) CheckForSystemUpdates(ctx context.Context, request *connect.Request[v1.CheckForSystemUpdatesRequest]) (*connect.Response[v1.CheckForSystemUpdatesResponse], error) {
	h.logger.Info("check for system updates request")
	response, err := h.updater.CheckForSystemUpdates(ctx, h.logger)
	if err != nil {
		h.logger.WithError(err).Error("failed to check for system updates")
		return nil, err
	}
	return connect.NewResponse(response), nil
}

func (h *rpc) CheckForContainerUpdates(ctx context.Context, request *connect.Request[v1.CheckForContainerUpdatesRequest]) (*connect.Response[v1.CheckForContainerUpdatesResponse], error) {
	images, err := h.updater.CheckForContainerUpdates(ctx, h.logger)
	if err != nil {
		h.logger.WithError(err).Error("failed to check for system container updates")
		return nil, err
	}
	return connect.NewResponse(&v1.CheckForContainerUpdatesResponse{
		ImageVersions: images,
	}), err
}

func (h *rpc) ChangeDaemonVersion(ctx context.Context, request *connect.Request[v1.ChangeDaemonVersionRequest]) (*connect.Response[v1.ChangeDaemonVersionResponse], error) {
	commander := daemon.GetCommander()
	err := commander.ChangeDaemonVersion(&dv1.ChangeDaemonVersionCommand{
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
	commander := daemon.GetCommander()
	err := commander.InstallOSUpdate()
	if err != nil {
		h.logger.WithError(err).Error("failed to change install os update")
		return nil, err
	}
	return connect.NewResponse(&v1.InstallOSUpdateResponse{}), nil
}

func (h *rpc) SetSystemImage(ctx context.Context, request *connect.Request[v1.SetSystemImageRequest]) (*connect.Response[v1.SetSystemImageResponse], error) {
	commander := daemon.GetCommander()
	err := commander.SetSystemImage(&dv1.SetSystemImageCommand{
		CurrentImage:   request.Msg.CurrentImage,
		RequestedImage: request.Msg.RequestedImage,
	})
	if err != nil {
		h.logger.WithError(err).Error("failed to set system image")
		return nil, err
	}
	return connect.NewResponse(&v1.SetSystemImageResponse{}), nil
}

func (h *rpc) AppsHealthCheck(ctx context.Context, request *connect.Request[v1.AppsHealthCheckRequest]) (*connect.Response[v1.AppsHealthCheckResponse], error) {
	checks, err := h.controller.CheckAppsHealth(ctx)
	if err != nil {
		h.logger.WithError(err).Error("failed to check apps health")
		return nil, err
	}
	return connect.NewResponse(&v1.AppsHealthCheckResponse{
		Checks: checks,
	}), nil
}

func (h *rpc) GetSystemStats(ctx context.Context, request *connect.Request[v1.GetSystemStatsRequest]) (*connect.Response[v1.GetSystemStatsResponse], error) {
	// grab the in-memory cache of current system stats
	stats := daemon.CurrentSystemStats
	if stats == nil {
		return nil, fmt.Errorf("no system stats available")
	}
	return connect.NewResponse(&v1.GetSystemStatsResponse{
		Stats: stats,
	}), nil
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

	isSetup, err := h.controller.IsDeviceSetup(ctx)
	if err != nil {
		return nil, errors.New(system.ErrFailedToGetSettings)
	}

	return connect.NewResponse(&v1.IsDeviceSetupResponse{Setup: isSetup}), nil
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

	var msg = request.Msg

	if err := msg.Validate(); err != nil {
		return nil, errors.New(ErrInvalidInputValues)
	}

	res, err := h.controller.Login(ctx, msg.GetUsername(), msg.GetPassword())
	if err != nil {
		return nil, errors.New("failed to login")
	}

	return connect.NewResponse(&v1.LoginResponse{Token: res}), nil
}

func (h *rpc) GetAppsInStore(ctx context.Context, request *connect.Request[v1.GetAppsInStoreRequest]) (*connect.Response[v1.GetAppsInStoreResponse], error) {
	h.logger.Info("getting apps in store")

	apps, err := h.controller.GetAppsInStore(ctx)
	if err != nil {
		return nil, errors.New(system.ErrFailedToGetApps)
	}

	return connect.NewResponse(&v1.GetAppsInStoreResponse{Apps: apps}), nil
}

func (h *rpc) GetDeviceSettings(ctx context.Context, request *connect.Request[v1.GetDeviceSettingsRequest]) (*connect.Response[v1.GetDeviceSettingsResponse], error) {
	return nil, errors.New("not implemented")
}
