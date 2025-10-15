package host

import (
	"path/filepath"

	"github.com/steady-bytes/draft/pkg/chassis"
)

const (
	homeCloudRoot = "/etc/home-cloud/"
)

func WireguardKeyPath() string {
	return FilePath(homeCloudRoot, "wireguard-keys/")
}

// FilePath cleans the given path and makes it a local path by prefixing a "./tmp/" if
// the draft env is "test".
func FilePath(paths ...string) string {
	path := filepath.Join(paths...)
	switch chassis.GetConfig().Env() {
	case "test":
		path = filepath.Join(".", "tmp", path)
	case "prod":
		path = filepath.Join("/mnt", "host", path)

	}
	return filepath.Clean(path)
}
