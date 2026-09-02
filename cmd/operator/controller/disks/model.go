package disks

import (
	"errors"
	"slices"
	"strings"

	dv1 "github.com/home-cloud-io/core/api/platform/daemon/v1"
)

// GetDiskIdentifier returns the "most stable" identifier for the disk
func GetDiskIdentifier(d *dv1.Disk) (string, error) {
	if d.Uuid != "" {
		return d.Uuid, nil
	}
	if d.Wwid != "" {
		return d.Uuid, nil
	}
	if d.Serial != "" {
		return d.Serial, nil
	}

	// this sort may be unnecessary: are symlinks always in the same order?
	s := slices.Clone(d.Symlinks)
	slices.Sort(s)
	for _, symlink := range s {
		if strings.HasPrefix(symlink, "/dev/disk/by-id") {
			return symlink, nil
		}
	}

	return "", errors.New("failed to find stable disk identifier")
}