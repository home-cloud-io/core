package server

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"connectrpc.com/connect"
	"github.com/cosi-project/runtime/pkg/resource"
	"github.com/dustin/go-humanize"
	"github.com/siderolabs/go-kubernetes/kubernetes/upgrade"
	"github.com/siderolabs/talos/pkg/cluster"
	k8s "github.com/siderolabs/talos/pkg/cluster/kubernetes"
	"github.com/siderolabs/talos/pkg/machinery/api/machine"
	"github.com/siderolabs/talos/pkg/machinery/api/storage"
	"github.com/siderolabs/talos/pkg/machinery/cel"
	"github.com/siderolabs/talos/pkg/machinery/cel/celenv"
	"github.com/siderolabs/talos/pkg/machinery/client"
	"github.com/siderolabs/talos/pkg/machinery/config/encoder"
	"github.com/siderolabs/talos/pkg/machinery/config/types/block"
	"github.com/siderolabs/talos/pkg/machinery/constants"
	blockpb "github.com/siderolabs/talos/pkg/machinery/resources/block"
	"github.com/steady-bytes/draft/pkg/chassis"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "github.com/home-cloud-io/core/api/platform/daemon/v1"
	sdConnect "github.com/home-cloud-io/core/api/platform/daemon/v1/v1connect"
	"github.com/home-cloud-io/core/pkg/talos"
)

type (
	Rpc interface {
		chassis.RPCRegistrar
		sdConnect.DaemonServiceHandler
	}

	rpcHandler struct {
		logger chassis.Logger
	}
)

func New(logger chassis.Logger) Rpc {
	return &rpcHandler{
		logger,
	}
}

// Implement the `RPCRegistrar` interface of draft so the `grpc` handlers are enabled
func (h *rpcHandler) RegisterRPC(server chassis.Rpcer) {
	pattern, handler := sdConnect.NewDaemonServiceHandler(h)
	server.AddHandler(pattern, handler, true)
}

func (h *rpcHandler) ShutdownHost(ctx context.Context, request *connect.Request[v1.ShutdownHostRequest]) (*connect.Response[v1.ShutdownHostResponse], error) {
	h.logger.Info("shutting down host")

	client, err := talos.Client(ctx)
	if err != nil {
		h.logger.WithError(err).Error(talos.ErrFailedToCreateClient)
		return nil, fmt.Errorf(talos.ErrFailedToCreateClient)
	}

	err = client.Shutdown(ctx)
	if err != nil {
		h.logger.WithError(err).Error("failed to shutdown host")
		return nil, err
	}
	return connect.NewResponse(&v1.ShutdownHostResponse{}), nil
}

func (h *rpcHandler) RebootHost(ctx context.Context, request *connect.Request[v1.RebootHostRequest]) (*connect.Response[v1.RebootHostResponse], error) {
	h.logger.Info("rebooting host")

	client, err := talos.Client(ctx)
	if err != nil {
		h.logger.WithError(err).Error(talos.ErrFailedToCreateClient)
		return nil, fmt.Errorf(talos.ErrFailedToCreateClient)
	}

	err = client.Reboot(ctx)
	if err != nil {
		h.logger.WithError(err).Error("failed to reboot host")
		return nil, err
	}
	return connect.NewResponse(&v1.RebootHostResponse{}), nil
}

func (h *rpcHandler) SystemStats(ctx context.Context, request *connect.Request[v1.SystemStatsRequest]) (*connect.Response[v1.SystemStatsResponse], error) {
	h.logger.Debug("getting system stats")

	client, err := talos.Client(ctx)
	if err != nil {
		h.logger.WithError(err).Error(talos.ErrFailedToCreateClient)
		return nil, fmt.Errorf(talos.ErrFailedToCreateClient)
	}

	stats := &v1.SystemStats{}
	stats.StartTime = timestamppb.Now()

	// TODO: this seems to always be roughly 0%
	computeResp, err := client.MachineClient.SystemStat(ctx, &emptypb.Empty{})
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
	memoryResp, err := client.MachineClient.Memory(ctx, &emptypb.Empty{})
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
	mountsResp, err := client.MachineClient.Mounts(ctx, &emptypb.Empty{})
	if err != nil {
		h.logger.WithError(err).Error("failed to get memory stats")
	}
	stats.Drives = []*v1.DriveStats{}
	for _, mount := range mountsResp.Messages[0].Stats {
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

func (h *rpcHandler) Version(ctx context.Context, request *connect.Request[v1.VersionRequest]) (*connect.Response[v1.VersionResponse], error) {
	h.logger.Debug("getting version")

	client, err := talos.Client(ctx)
	if err != nil {
		h.logger.WithError(err).Error(talos.ErrFailedToCreateClient)
		return nil, fmt.Errorf(talos.ErrFailedToCreateClient)
	}

	resp, err := client.MachineClient.Version(ctx, &emptypb.Empty{})
	if err != nil {
		h.logger.WithError(err).Error("failed to get version")
		return nil, err
	}

	return connect.NewResponse(&v1.VersionResponse{
		Name:    "talos",
		Version: resp.Messages[0].Version.Tag,
	}), nil
}

func (h *rpcHandler) Upgrade(ctx context.Context, request *connect.Request[v1.UpgradeRequest]) (*connect.Response[v1.UpgradeResponse], error) {
	h.logger.Info("upgrading host")

	client, err := talos.Client(ctx)
	if err != nil {
		h.logger.WithError(err).Error(talos.ErrFailedToCreateClient)
		return nil, fmt.Errorf(talos.ErrFailedToCreateClient)
	}

	_, err = client.MachineClient.Upgrade(ctx, &machine.UpgradeRequest{
		Image: fmt.Sprintf("%s:%s", request.Msg.Source, request.Msg.Version),
	})
	if err != nil {
		h.logger.WithError(err).Error("failed to upgrade")
		return nil, err
	}
	return connect.NewResponse(&v1.UpgradeResponse{}), nil
}

func (h *rpcHandler) UpgradeKubernetes(ctx context.Context, request *connect.Request[v1.UpgradeKubernetesRequest]) (*connect.Response[v1.UpgradeKubernetesResponse], error) {
	h.logger.Info("upgrading kubernetes")

	client, err := talos.Client(ctx)
	if err != nil {
		h.logger.WithError(err).Error(talos.ErrFailedToCreateClient)
		return nil, fmt.Errorf(talos.ErrFailedToCreateClient)
	}

	err = upgradeKubernetes(ctx, client, request.Msg.Version)
	if err != nil {
		h.logger.WithError(err).Error("failed to upgrade kubernetes")
		return nil, fmt.Errorf("failed to upgrade kubernetes")
	}
	return connect.NewResponse(&v1.UpgradeKubernetesResponse{}), nil
}

func (h *rpcHandler) CreateVolume(ctx context.Context, request *connect.Request[v1.CreateVolumeRequest]) (*connect.Response[v1.CreateVolumeResponse], error) {
	h.logger.Info("creating volume")

	var minSize block.ByteSize
	err := minSize.UnmarshalText([]byte(request.Msg.MinSize))
	if err != nil {
		h.logger.WithError(err).Warn("invalid min_size")
		return nil, status.Error(codes.InvalidArgument, "invalid min_size")
	}

	var maxSize block.Size
	err = maxSize.UnmarshalText([]byte(request.Msg.MaxSize))
	if err != nil {
		h.logger.WithError(err).Warn("invalid max_size")
		return nil, status.Error(codes.InvalidArgument, "invalid max_size")
	}

	uvc := block.NewUserVolumeConfigV1Alpha1()
	uvc.MetaName = request.Msg.Name
	uvc.ProvisioningSpec = block.ProvisioningSpec{
		DiskSelectorSpec: block.DiskSelector{
			// TODO: will probably want to expose this expression on the API
			Match: cel.MustExpression(cel.ParseBooleanExpression("!system_disk", celenv.DiskLocator())),
		},
		ProvisioningMinSize: minSize,
		ProvisioningMaxSize: maxSize,
	}

	_, err = uvc.Validate(talos.ValidationMode{})
	if err != nil {
		h.logger.WithError(err).Warn("failed UserVolumeConfig validation")
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	id, err := talos.CreateUserVolume(ctx, h.logger, uvc)
	if err != nil {
		h.logger.WithError(err).Error("failed to create volume")
		return nil, err
	}

	return connect.NewResponse(&v1.CreateVolumeResponse{
		Id:   id,
		Path: fmt.Sprintf("/var/mnt/%s", request.Msg.Name),
	}), nil
}

func (h *rpcHandler) DeleteVolume(ctx context.Context, request *connect.Request[v1.DeleteVolumeRequest]) (*connect.Response[v1.DeleteVolumeResponse], error) {
	h.logger.Error("unimplemented")
	return nil, status.Error(codes.Unimplemented, "unimplemented")
}

func (h *rpcHandler) GetDisks(ctx context.Context, request *connect.Request[v1.GetDisksRequest]) (*connect.Response[v1.GetDisksResponse], error) {
	h.logger.Debug("getting disks")

	client, err := talos.Client(ctx)
	if err != nil {
		h.logger.WithError(err).Error(talos.ErrFailedToCreateClient)
		return nil, fmt.Errorf(talos.ErrFailedToCreateClient)
	}

	list, err := client.COSI.List(ctx, resource.NewMetadata("runtime", blockpb.DiskType, "", resource.VersionUndefined))
	if err != nil {
		h.logger.WithError(err).Error("failed to list disks")
	}
	for _, item := range list.Items {
		disk := item.Spec().(*blockpb.DiskSpec)
		fmt.Println(disk.Symlinks)
	}


	disksStatsResp, err := client.MachineClient.DiskStats(ctx, &emptypb.Empty{})
	if err != nil {
		h.logger.WithError(err).Error("failed to get disk stats")
		return nil, err
	}

	for _, msg := range disksStatsResp.Messages {
		for _, device := range msg.Devices {
			if strings.HasPrefix(device.Name, "sd") && len(device.Name) == 3 {
				fmt.Printf("Disk: %s\n", device.Name)
				continue
			}
			if strings.HasPrefix(device.Name, "nvme") && len(device.Name) == 7 {
				fmt.Printf("Disk: %s\n", device.Name)
				continue
			}
		}
	}

	disksResp, err := client.StorageClient.Disks(ctx, &emptypb.Empty{})
	if err != nil {
		h.logger.WithError(err).Error("failed to get disks")
		return nil, err
	}

	disks := []*v1.Disk{}
	for _, msg := range disksResp.Messages {
		for _, disk := range msg.Disks {
			// we only consider NVME, SSD, and HDD usable drives
			if disk.Type == storage.Disk_NVME ||
				disk.Type == storage.Disk_SSD ||
				disk.Type == storage.Disk_HDD {
				fmt.Printf("%s - %s - %s\n", disk.DeviceName, humanize.BigBytes(big.NewInt(int64(disk.Size))), disk.BusPath)
				disks = append(disks, &v1.Disk{
					DeviceName: disk.DeviceName,
					Model:      disk.Model,
					Serial:     disk.Serial,
					Wwid:       disk.Wwid,
					Uuid:       disk.Uuid,
					Type:       convertDiskType(disk.Type),
					SystemDisk: disk.SystemDisk,
					Size:       disk.Size,
				})
			}
		}
	}

	// stream, err := client.MachineClient.DiskUsage(ctx, &machine.DiskUsageRequest{
	// 	Paths: []string{"/", "/mnt/apps1"},
	// })
	// if err != nil {
	// 	h.logger.WithError(err).Error("failed to get disk usage")
	// 	return nil, err
	// }

	// err = helpers.ReadGRPCStream(stream, func(info *machine.DiskUsageInfo, node string, multipleNodes bool) error {
	// 	fmt.Printf("%s - %s: %d\n", node, info.Name, info.GetSize())
	// 	return nil
	// })
	// if err != nil {
	// 	h.logger.WithError(err).Error("failed to stream disk usage info")
	// 	return nil, err
	// }

	return connect.NewResponse(&v1.GetDisksResponse{
		Disks: disks,
	}), nil
}

// TODO: basically the same as CreateVolume but uses the VolumeTypeDisk which loads the entire disk as a UserVolume
func (h *rpcHandler) LoadDisk(ctx context.Context, request *connect.Request[v1.LoadDiskRequest]) (*connect.Response[v1.LoadDiskResponse], error) {
	h.logger.Error("unimplemented")
	return nil, status.Error(codes.Unimplemented, "unimplemented")
}

// helpers

func convertDiskType(t storage.Disk_DiskType) v1.DiskType {
	switch t {
	case storage.Disk_SSD:
		return v1.DiskType_DEVICE_TYPE_SSD
	case storage.Disk_HDD:
		return v1.DiskType_DEVICE_TYPE_HDD
	case storage.Disk_NVME:
		return v1.DiskType_DEVICE_TYPE_NVME
	default:
		return v1.DiskType_DEVICE_TYPE_UNSPECIFIED
	}
}

func upgradeKubernetes(ctx context.Context, c *client.Client, toVersion string) error {

	upgradeOptions := k8s.UpgradeOptions{
		PrePullImages:          true,
		UpgradeKubelet:         true,
		KubeletImage:           constants.KubeletImage,
		APIServerImage:         constants.KubernetesAPIServerImage,
		ControllerManagerImage: constants.KubernetesControllerManagerImage,
		SchedulerImage:         constants.KubernetesSchedulerImage,
		ProxyImage:             constants.KubeProxyImage,
	}

	clientProvider := &cluster.ConfigClientProvider{
		DefaultClient: c,
	}
	defer clientProvider.Close() //nolint:errcheck

	state := struct {
		cluster.ClientProvider
		cluster.K8sProvider
	}{
		ClientProvider: clientProvider,
		K8sProvider: &cluster.KubernetesClient{
			ClientProvider: clientProvider,
			ForceEndpoint:  upgradeOptions.ControlPlaneEndpoint,
		},
	}

	fromVersion, err := k8s.DetectLowestVersion(ctx, &state, upgradeOptions)
	if err != nil {
		return err
	}

	upgradeOptions.Path, err = upgrade.NewPath(fromVersion, toVersion)
	if err != nil {
		return err
	}

	upgradeOptions.EncoderOpt = encoder.WithComments(encoder.CommentsAll)

	return k8s.Upgrade(ctx, &state, upgradeOptions)
}
