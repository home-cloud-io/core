package host

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"

	v1 "github.com/home-cloud-io/core/api/platform/daemon/v1"
	"github.com/home-cloud-io/core/services/platform/daemon/execute"

	"github.com/steady-bytes/draft/pkg/chassis"
)

type (
	TimeConfig struct {
		TimeZone string `json:"timeZone"`
	}
	ServicesConfig struct {
		Resolved struct {
			Enable  bool     `json:"enable"`
			Domains []string `json:"domains"`
		} `json:"resolved"`
		K3s struct {
			Enable     bool   `json:"enable"`
			Role       string `json:"role"`
			ExtraFlags string `json:"extraFlags"`
		} `json:"k3s"`
		OpenSSH struct {
			Enable         bool `json:"enable"`
			AuthorizedKeys struct {
				Keys []string `json:"keys"`
			} `json:"authorizedKeys"`
		} `json:"openssh"`
		Avahi struct {
			Enable   bool `json:"enable"`
			IPv4     bool `json:"ipv4"`
			IPv6     bool `json:"ipv6"`
			NSSmDNS4 bool `json:"nssmdns4"`
			Publish  struct {
				Enable       bool `json:"enable"`
				Domain       bool `json:"domain"`
				Addresses    bool `json:"addresses"`
				UserServices bool `json:"userServices"`
			}
		} `json:"avahi"`
	}
)

func SaveSettings(ctx context.Context, logger chassis.Logger, def *v1.SaveSettingsCommand) error {

	err := setUserPassword(ctx, logger, def.AdminPassword)
	if err != nil {
		return err
	}

	err = setTimeZone(def.TimeZone)
	if err != nil {
		return err
	}

	err = setSSH(def.EnableSsh, def.TrustedSshKeys)
	if err != nil {
		return err
	}

	err = RebuildAndSwitchOS(ctx, logger)
	if err != nil {
		return err
	}

	return nil
}

func setTimeZone(timeZone string) error {
	// read
	f, err := os.ReadFile(TimeConfigFile)
	if err != nil {
		return err
	}
	c := TimeConfig{}
	err = json.Unmarshal(f, &c)
	if err != nil {
		return err
	}

	// update
	c.TimeZone = timeZone

	// write
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	err = os.WriteFile(TimeConfigFile, b, 0777)
	if err != nil {
		return err
	}
	return nil
}

func setSSH(enableSSH bool, trustedSSHKeys []string) error {
	// read
	f, err := os.ReadFile(ServicesConfigFile)
	if err != nil {
		return err
	}
	c := ServicesConfig{}
	err = json.Unmarshal(f, &c)
	if err != nil {
		return err
	}

	// update
	c.OpenSSH.Enable = enableSSH
	if trustedSSHKeys == nil {
		trustedSSHKeys = []string{}
	}
	c.OpenSSH.AuthorizedKeys.Keys = trustedSSHKeys

	// write
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	err = os.WriteFile(ServicesConfigFile, b, 0777)
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
