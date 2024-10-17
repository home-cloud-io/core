package host

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"

	v1 "github.com/home-cloud-io/core/api/platform/daemon/v1"
	"github.com/home-cloud-io/core/services/platform/daemon/execute"

	"github.com/steady-bytes/draft/pkg/chassis"
)

const (
	NixosConfigFile = "/etc/nixos/configuration.nix"
)

func SaveSettings(ctx context.Context, logger chassis.Logger, def *v1.SaveSettingsCommand) error {
	var (
		replacers = []Replacer{}
	)

	err := setUserPassword(ctx, logger, def.AdminPassword)
	if err != nil {
		return err
	}

	if def.TimeZone != "" {
		replacers = append(replacers, func(line string) string {
			if strings.HasPrefix(line, "  time.timeZone =") {
				return fmt.Sprintf("  time.timeZone = \"%s\";", def.TimeZone)
			}
			return line
		})
	}

	if def.EnableSsh {
		replacers = append(replacers, func(line string) string {
			// default config comments out the line
			if line == "  # services.openssh.enable = true;" {
				return "  services.openssh.enable = true;"
			}
			// if it's been disabled after install it will be set to false
			if line == "  services.openssh.enable = false;" {
				return "  services.openssh.enable = true;"
			}
			return line
		})
	} else {
		replacers = append(replacers, func(line string) string {
			if line == "  services.openssh.enable = true;" {
				return "  services.openssh.enable = false;"
			}
			return line
		})
	}

	if len(def.TrustedSshKeys) > 0 {
		newLine := "    openssh.authorizedKeys.keys = ["
		for _, key := range def.TrustedSshKeys {
			newLine += fmt.Sprintf(" \"%s\"", key)
		}
		newLine += " ];"
		replacers = append(replacers, func(line string) string {
			if strings.HasPrefix(line, "    openssh.authorizedKeys.keys =") {
				return newLine
			}
			return line
		})
	} else {
		replacers = append(replacers, func(line string) string {
			if strings.HasPrefix(line, "    openssh.authorizedKeys.keys =") {
				return "    openssh.authorizedKeys.keys = [];"
			}
			return line
		})
	}

	err = LineByLineReplace(NixosConfigFile, replacers)
	if err != nil {
		return err
	}

	err = RebuildAndSwitchOS(ctx, logger)
	if err != nil {
		return err
	}

	return nil
}

func setUserPassword(ctx context.Context, logger chassis.Logger, password string) error {

	if password == "" {
		logger.Info("ignoring empty password change")
		return nil
	}

	config := chassis.GetConfig()
	if config.Env() == "test" {
		logger.Info("mocking password change")
		return nil
	}

	cmd := exec.Command("chpasswd")

	// write the username:password to stdin when the command executes
	stdin, err := cmd.StdinPipe()
	if err != nil {
		logger.WithError(err).Error("failed to get stdin pipe")
		return err
	}
	go func() {
		defer stdin.Close()
		io.WriteString(stdin, fmt.Sprintf("admin:%s", password))
	}()

	// execute command
	err = execute.ExecuteCommand(ctx, cmd)
	if err != nil {
		logger.WithError(err).Error("failed to set user password")
		return err
	}
	logger.Info("user password set successfully")
	return nil
}
