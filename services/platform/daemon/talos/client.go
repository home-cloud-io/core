package talos

import (
	"context"
	"fmt"

	"github.com/siderolabs/talos/pkg/machinery/client"
	clientconfig "github.com/siderolabs/talos/pkg/machinery/client/config"
	"github.com/steady-bytes/draft/pkg/chassis"
)

const (
	talosConfigKey     = "daemon.talos_config"
	defaultTalosConfig = "/var/run/secrets/talos.dev/config"

	ErrFailedToCreateClient = "failed to create talos client"
)

func init() {
	chassis.GetConfig().SetDefault(talosConfigKey, defaultTalosConfig)
}

// Client configures and returns a Talos client. This client will not update certificates or configuration
// if they are rotated on disk. Therefore it is best to create new clients on use instead of using a single
// long-lived client.
func Client(ctx context.Context) (*client.Client, error) {
	configPath := chassis.GetConfig().GetString(talosConfigKey)
	cfg, err := clientconfig.Open(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file %q: %w", configPath, err)
	}

	opts := []client.OptionFunc{
		client.WithConfig(cfg),
		client.WithDefaultGRPCDialOptions(),
	}

	return client.New(ctx, opts...)
}
