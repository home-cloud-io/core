package talos

import (
	"bytes"
	"context"
	"fmt"

	"github.com/cosi-project/runtime/pkg/resource"
	"github.com/cosi-project/runtime/pkg/state"
	"github.com/siderolabs/talos/cmd/talosctl/pkg/talos/yamlstrip"
	"github.com/siderolabs/talos/pkg/machinery/api/machine"
	"github.com/siderolabs/talos/pkg/machinery/client"
	clientconfig "github.com/siderolabs/talos/pkg/machinery/client/config"
	"github.com/siderolabs/talos/pkg/machinery/config/configpatcher"
	"github.com/siderolabs/talos/pkg/machinery/config/types/block"
	"github.com/steady-bytes/draft/pkg/chassis"
	"go.yaml.in/yaml/v4"
	"k8s.io/utils/ptr"
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

func CreateUserVolume(ctx context.Context, logger chassis.Logger, uvc *block.UserVolumeConfigV1Alpha1) (id string, err error) {
	// TODO: support multi-node
	node := ""

	c, err := Client(ctx)
	if err != nil {
		logger.WithError(err).Error("failed to create client")
		return
	}

	out, err := yaml.Marshal(uvc)
	if err != nil {
		logger.WithError(err).Error("failed to marshal UserVolumeConfig to yaml")
		return
	}

	patch, err := configpatcher.LoadPatch(out)
	if err != nil {
		logger.WithError(err).Error("failed to load patch")
		return
	}
	patches := []configpatcher.Patch{patch}

	rd, err := c.ResolveResourceKind(ctx, ptr.To(""), "mc")
	if err != nil {
		logger.WithError(err).Error("failed to resolve resource kind")
		return
	}

	mc, err := c.COSI.Get(
		ctx,
		resource.NewMetadata("config", rd.TypedSpec().Type, "v1alpha1", resource.VersionUndefined),
		state.WithGetUnmarshalOptions(state.WithSkipProtobufUnmarshal()),
	)
	if err != nil {
		logger.WithError(err).Error("failed to get MachineConfig")
		return
	}

	body, err := extractMachineConfigBody(mc)
	if err != nil {
		logger.WithError(err).Error("failed to extract MachineConfig body")
		return
	}

	cfg, err := configpatcher.Apply(configpatcher.WithBytes(body), patches)
	if err != nil {
		logger.WithError(err).Error("failed to apply patch")
		return
	}

	patched, err := cfg.Bytes()
	if err != nil {
		logger.WithError(err).Error("failed to get config bytes")
		return
	}

	_, err = c.ApplyConfiguration(ctx, &machine.ApplyConfigurationRequest{
		Data: patched,
		Mode: machine.ApplyConfigurationRequest_AUTO,
		// DryRun:         patchCmdFlags.dryRun,
		// TryModeTimeout: durationpb.New(patchCmdFlags.configTryTimeout),
	})
	if err != nil {
		logger.WithError(err).Error("failed to apply configuration")
		return
	}

	if bytes.Equal(
		bytes.TrimSpace(yamlstrip.Comments(patched)),
		bytes.TrimSpace(yamlstrip.Comments(body)),
	) {
		logger.Info("apply was skipped: no changes detected")
	} else {
		logger.WithFields(chassis.Fields{
			"resource": fmt.Sprintf("%s/%s", mc.Metadata().Type(), mc.Metadata().ID()),
			"node":     node,
		}).Info("patched resource")
	}

	return fmt.Sprintf("u-%s", uvc.MetaName), nil
}

func extractMachineConfigBody(mc resource.Resource) ([]byte, error) {
	if mc.Metadata().Annotations().Empty() {
		return yaml.Marshal(mc.Spec())
	}

	spec, err := yaml.Marshal(mc.Spec())
	if err != nil {
		return nil, err
	}

	var bodyStr string

	if err = yaml.Unmarshal(spec, &bodyStr); err != nil {
		return nil, err
	}

	return []byte(bodyStr), nil
}
