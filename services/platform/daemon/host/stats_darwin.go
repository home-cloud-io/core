//go:build darwin
// +build darwin

package host

// NOTE: the daemon is not intended to be run on darwin systems except for development
// purposes. The below stats are limited on darwin builds only to enable build for local
// development.

import (
	v1 "github.com/home-cloud-io/core/api/platform/daemon/v1"

	"github.com/mackerelio/go-osstat/memory"
)

// NOTE: darwin builds do not support compute stats and so this will only ever
// return zeros as values.
func getComputeStats(stats *v1.SystemStats) error {
	stats.Compute = &v1.ComputeStats{
		UserPercent:   0,
		SystemPercent: 0,
		IdlePercent:   0,
	}
	return nil
}

// NOTE: darwin builds do not support Available memory calls so this value
// will only ever be zero.
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
