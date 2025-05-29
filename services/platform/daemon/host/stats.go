package host

import (
	"syscall"
	"time"

	v1 "github.com/home-cloud-io/core/api/platform/daemon/v1"

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

func getDriveStats(stats *v1.SystemStats, mounts []string) error {
	for index, mountPoint := range mounts {
		var stat syscall.Statfs_t
		err := syscall.Statfs(mountPoint, &stat)
		if err != nil {
			return err
		}
		// Available blocks * size per block = available space in bytes
		free := stat.Bavail * uint64(stat.Bsize)
		total := stat.Blocks * uint64(stat.Bsize)
		stats.Drives[index] = &v1.DriveStats{
			MountPoint: mountPoint,
			TotalBytes: total,
			FreeBytes:  free,
		}
	}
	return nil
}
