package communicate

import (
	"context"

	"connectrpc.com/connect"
	"github.com/siderolabs/talos/pkg/machinery/client"
	"github.com/steady-bytes/draft/pkg/chassis"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "github.com/home-cloud-io/core/api/platform/daemon/v1"
	sdConnect "github.com/home-cloud-io/core/api/platform/daemon/v1/v1connect"
	"github.com/home-cloud-io/core/services/platform/daemon/talos"
)

type (
	Rpc interface {
		chassis.RPCRegistrar
		sdConnect.DaemonServiceHandler
	}

	rpcHandler struct {
		logger chassis.Logger
		// TODO: probably want to wrap this in a controller for more complex operations eventually
		client *client.Client
	}
)

func New(logger chassis.Logger) Rpc {
	client, err := talos.Client()
	if err != nil {
		logger.WithError(err).Fatal("failed to create talos client")
	}
	return &rpcHandler{
		logger,
		client,
	}
}

// Implement the `RPCRegistrar` interface of draft so the `grpc` handlers are enabled
func (h *rpcHandler) RegisterRPC(server chassis.Rpcer) {
	pattern, handler := sdConnect.NewDaemonServiceHandler(h)
	server.AddHandler(pattern, handler, true)
}

func (h *rpcHandler) ShutdownHost(ctx context.Context, request *connect.Request[v1.ShutdownHostRequest]) (*connect.Response[v1.ShutdownHostResponse], error) {
	h.logger.Info("shutting down host")
	err := h.client.Shutdown(ctx)
	if err != nil {
		h.logger.WithError(err).Error("failed to shutdown host")
		return nil, err
	}
	return connect.NewResponse(&v1.ShutdownHostResponse{}), nil
}

func (h *rpcHandler) RebootHost(ctx context.Context, request *connect.Request[v1.RebootHostRequest]) (*connect.Response[v1.RebootHostResponse], error) {
	h.logger.Info("rebooting host")
	err := h.client.Reboot(ctx)
	if err != nil {
		h.logger.WithError(err).Error("failed to reboot host")
		return nil, err
	}
	return connect.NewResponse(&v1.RebootHostResponse{}), nil
}

func (h *rpcHandler) SystemStats(ctx context.Context, request *connect.Request[v1.SystemStatsRequest]) (*connect.Response[v1.SystemStatsResponse], error) {
	stats := &v1.SystemStats{}
	stats.StartTime = timestamppb.Now()

	// TODO: this seems to always be roughly 0%
	computeResp, err := h.client.MachineClient.SystemStat(ctx, &emptypb.Empty{})
	if err != nil {
		h.logger.WithError(err).Error("failed to get load average stats")
	}
	stat := computeResp.Messages[0].CpuTotal
	idle := stat.Idle + stat.Iowait
	nonIdle := stat.User + stat.Nice + stat.System + stat.Irq + stat.Steal + stat.SoftIrq
	total := idle + nonIdle
	stats.Compute = &v1.ComputeStats{
		UserPercent:   float32(computeResp.Messages[0].CpuTotal.User / total),
		SystemPercent: float32(computeResp.Messages[0].CpuTotal.System / total),
		IdlePercent:   float32(computeResp.Messages[0].CpuTotal.Idle / total),
	}

	// TODO: returns 42% when talosctl dashboard shows 32%
	memoryResp, err := h.client.MachineClient.Memory(ctx, &emptypb.Empty{})
	if err != nil {
		h.logger.WithError(err).Error("failed to get memory stats")
	}
	stats.Memory = &v1.MemoryStats{
		TotalBytes:     memoryResp.Messages[0].Meminfo.Memtotal,
		FreeBytes:      memoryResp.Messages[0].Meminfo.Memfree,
		AvailableBytes: memoryResp.Messages[0].Meminfo.Memavailable,
		UsedBytes:      memoryResp.Messages[0].Meminfo.Memtotal - memoryResp.Messages[0].Meminfo.Memavailable,
		CachedBytes:    memoryResp.Messages[0].Meminfo.Cached,
	}

	// TODO: get disk total amounts, then subtract UserVolume usage?
	drivesResp, err := h.client.MachineClient.Mounts(ctx, &emptypb.Empty{})
	if err != nil {
		h.logger.WithError(err).Error("failed to get memory stats")
	}
	stats.Drives = []*v1.DriveStats{}
	for _, mount := range drivesResp.Messages[0].Stats {
		if mount.MountedOn == "/" {
			stats.Drives = []*v1.DriveStats{
				{
					MountPoint: mount.MountedOn,
					TotalBytes: mount.Size,
					FreeBytes:  mount.Available,
				},
			}
		}
	}

	stats.EndTime = timestamppb.Now()
	return connect.NewResponse(&v1.SystemStatsResponse{
		Stats: stats,
	}), nil
}
