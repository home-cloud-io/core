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
	f, err := os.ReadFile(TimeConfigFile())
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
	err = os.WriteFile(TimeConfigFile(), b, 0777)
	if err != nil {
		return err
	}
	return nil
}

func setSSH(enableSSH bool, trustedSSHKeys []string) error {
	// update ssh config

	bytes, err := os.ReadFile(ServicesConfigFile())
	if err != nil {
		return err
	}
	sshConfig := ServicesConfig{}
	err = json.Unmarshal(bytes, &sshConfig)
	if err != nil {
		return err
	}

	sshConfig.OpenSSH.Enable = enableSSH

	err = WriteJsonFile(ServicesConfigFile(), sshConfig, DefaultFileMode)
	if err != nil {
		return err
	}

	// update users config

	bytes, err = os.ReadFile(UsersConfigFile())
	if err != nil {
		return err
	}
	usersConfig := UsersConfig{}
	err = json.Unmarshal(bytes, &sshConfig)
	if err != nil {
		return err
	}

	if trustedSSHKeys == nil {
		trustedSSHKeys = []string{}
	}
	user, ok := usersConfig.Users["admin"]
	if !ok {
		return fmt.Errorf("admin user did not exist in config")
	}
	user.OpenSSH.AuthorizedKeys.Keys = trustedSSHKeys
	usersConfig.Users["admin"] = user

	err = WriteJsonFile(UsersConfigFile(), usersConfig, DefaultFileMode)
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
