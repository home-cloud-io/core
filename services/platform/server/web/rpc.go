package web

import (
	"context"
	"fmt"
	"sync"

	dv1 "github.com/home-cloud-io/core/api/platform/daemon/v1"
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
		logger           chassis.Logger
		k8sclient        k8sclient.Client
		messages         chan *dv1.DaemonMessage
		systemUpdateLock sync.Mutex
	}
)

const (
	daemonTagPath = "refs/tags/services/platform/daemon/"
)

func New(logger chassis.Logger, messages chan *dv1.DaemonMessage) Rpc {
	return &rpc{
		logger:           logger,
		k8sclient:        k8sclient.NewClient(logger),
		messages:         messages,
		systemUpdateLock: sync.Mutex{},
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
		Version: request.Msg.Version,
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
		Version: request.Msg.Version,
	})
	if err != nil {
		h.logger.WithError(err).Error("failed to update app")
		return nil, err
	}
	h.logger.Info("finished request")
	return connect.NewResponse(&v1.UpdateAppResponse{}), nil
}

func (h *rpc) CheckForSystemUpdates(ctx context.Context, request *connect.Request[v1.CheckForSystemUpdatesRequest]) (*connect.Response[v1.CheckForSystemUpdatesResponse], error) {
	h.logger.Info("check for system updates request")
	if !h.systemUpdateLock.TryLock() {
		h.logger.Warn("call to check for system updates while another check is already in progress")
		return nil, fmt.Errorf("system update check already in progress")
	}
	defer h.systemUpdateLock.Unlock()

	var (
		response = &v1.CheckForSystemUpdatesResponse{}
	)

	// get the os update diff from the daemon
	commander := daemon.GetCommander()
	err := commander.RequestOSUpdateDiff()
	if err != nil {
		return nil, err
	}
	for {
		msg := <-h.messages
		switch msg.Message.(type) {
		case *dv1.DaemonMessage_OsUpdateDiff:
			m := msg.Message.(*dv1.DaemonMessage_OsUpdateDiff)
			response.OsDiff = m.OsUpdateDiff.Description
		default:
			h.logger.WithField("message", msg).Warn("unrequested message type received")
		}
		if response.OsDiff != "" {
			break
		}
	}

	// get the current daemon version from the daemon
	err = commander.RequestCurrentDaemonVersion()
	if err != nil {
		return nil, err
	}
	for {
		msg := <-h.messages
		switch msg.Message.(type) {
		case *dv1.DaemonMessage_CurrentDaemonVersion:
			m := msg.Message.(*dv1.DaemonMessage_CurrentDaemonVersion)
			response.DaemonVersions = &v1.DaemonVersions{
				Current: m.CurrentDaemonVersion.Version,
			}
		default:
			h.logger.WithField("message", msg).Warn("unrequested message type received")
		}
		if response.DaemonVersions != nil {
			break
		}
	}

	// get latest available daemon version
	latest, err := getLatestDaemonVersion()
	if err != nil {
		return nil, err
	}
	response.DaemonVersions.Latest = latest

	return connect.NewResponse(response), nil
}

func (h *rpc) CheckForContainerUpdates(ctx context.Context, request *connect.Request[v1.CheckForContainerUpdatesRequest]) (*connect.Response[v1.CheckForContainerUpdatesResponse], error) {
	images, err := h.k8sclient.CurrentContainerVersions(ctx)
	if err != nil {
		h.logger.WithError(err).Error("failed to get current container versions")
		return nil, err
	}

	images, err = getLatestImageTags(ctx, images)
	if err != nil {
		h.logger.WithError(err).Error("failed to get latest image versions")
		return nil, err
	}

	return connect.NewResponse(&v1.CheckForContainerUpdatesResponse{
		ImageVersions: images,
	}), err
}

func (h *rpc) ChangeDaemonVersion(ctx context.Context, request *connect.Request[v1.ChangeDaemonVersionRequest]) (*connect.Response[v1.ChangeDaemonVersionResponse], error) {
	commander := daemon.GetCommander()
	err := commander.ChangeDaemonVersion(request.Msg)
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
	err := commander.SetSystemImage(request.Msg)
	if err != nil {
		h.logger.WithError(err).Error("failed to set system image")
		return nil, err
	}
	return connect.NewResponse(&v1.SetSystemImageResponse{}), nil
}
