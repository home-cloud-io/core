package host

import (
	"syscall"
	"time"

	v1 "github.com/home-cloud-io/core/api/platform/daemon/v1"

	"github.com/mackerelio/go-osstat/cpu"
	"github.com/mackerelio/go-osstat/memory"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
)

const (
	ComputeMeasurementDuration = 1 * time.Second
)

func SystemStats(mounts []string) (*v1.SystemStats, error) {
	var (
		stats = &v1.SystemStats{
			StartTime: timestamppb.Now(),
			Drives:    make([]*v1.DriveStats, len(mounts)),
		}
		err error
	)

	err = getComputeStats(stats)
	if err != nil {
		return nil, err
	}

	err = getMemoryStats(stats)
	if err != nil {
		return nil, err
	}

	err = getDriveStats(stats, mounts)
	if err != nil {
		return nil, err
	}

	stats.EndTime = timestamppb.Now()
	return stats, nil
}

func getComputeStats(stats *v1.SystemStats) error {
	before, err := cpu.Get()
	if err != nil {
		return err
	}
	time.Sleep(ComputeMeasurementDuration)
	after, err := cpu.Get()
	if err != nil {
		return err
	}
	total := float32(after.Total - before.Total)
	stats.Compute = &v1.ComputeStats{
		UserPercent:   float32(after.User-before.User) / total * 100,
		SystemPercent: float32(after.System-before.System) / total * 100,
		IdlePercent:   float32(after.Idle-before.Idle) / total * 100,
	}
	return nil
}

func getMemoryStats(stats *v1.SystemStats) error {
	memory, err := memory.Get()
	if err != nil {
		return err
	}
	stats.Memory = &v1.MemoryStats{
		TotalBytes:     uint32(memory.Total),
		UsedBytes:      uint32(memory.Used),
		CachedBytes:    uint32(memory.Cached),
		FreeBytes:      uint32(memory.Free),
		AvailableBytes: uint32(memory.Available),
	}
	return nil
}

func getDriveStats(stats *v1.SystemStats, mounts []string) error {
	for index, mountPoint := range mounts {
		var stat syscall.Statfs_t
		err := syscall.Statfs(mountPoint, &stat)
		if err != nil {
			return err
		}
		// Available blocks * size per block = available space in bytes
		free := uint32(stat.Bavail * uint64(stat.Bsize))
		total := uint32(stat.Blocks) * uint32(stat.Bsize)
		stats.Drives[index] = &v1.DriveStats{
			MountPoint: mountPoint,
			TotalBytes: total,
			FreeBytes:  free,
		}
	}
	return nil
}
