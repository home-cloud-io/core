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
)

func Client() (*client.Client, error) {
	ctx := context.Background()
	config := chassis.GetConfig()
	config.SetDefault(talosConfigKey, defaultTalosConfig)

	configPath := config.GetString(talosConfigKey)
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
