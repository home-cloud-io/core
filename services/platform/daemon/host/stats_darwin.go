//go:build darwin
// +build darwin

package host

import (
	v1 "github.com/home-cloud-io/core/api/platform/daemon/v1"

	"github.com/mackerelio/go-osstat/memory"
)

func getComputeStats(stats *v1.SystemStats) error {
	stats.Compute = &v1.ComputeStats{
		UserPercent:   0,
		SystemPercent: 0,
		IdlePercent:   0,
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
		// memory.Available not included on Darwin builds
	}
	return nil
}
