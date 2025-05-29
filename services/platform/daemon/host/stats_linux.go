package host

import (
	"time"

	v1 "github.com/home-cloud-io/core/api/platform/daemon/v1"

	"github.com/mackerelio/go-osstat/cpu"
	"github.com/mackerelio/go-osstat/memory"
)

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
		TotalBytes:     memory.Total,
		UsedBytes:      memory.Used,
		CachedBytes:    memory.Cached,
		FreeBytes:      memory.Free,
		AvailableBytes: memory.Available,
	}
	return nil
}
