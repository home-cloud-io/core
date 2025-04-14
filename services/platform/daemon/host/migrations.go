package host

import (
	"context"
	"errors"
	"os"
	"strings"
	"time"

	"github.com/steady-bytes/draft/pkg/chassis"
	"gopkg.in/yaml.v3"
)

type (
	Migrator interface {
		Migrate()
	}
	migrator struct {
		logger chassis.Logger
	}
	migrationsHistory struct {
		Migrations []migrationRun
	}
	migrationRun struct {
		Id        string
		Name      string
		Timestamp time.Time
		Error     string
	}
	migrationConfig struct {
		Id       string
		Name     string
		Required bool
		Run      func(logger chassis.Logger) error
	}
)

var (
	migrationsList = []migrationConfig{
		{
			Id:       "60fbc7c6-9388-4c33-9a02-4da87de5ba6d",
			Name:     "Grant server read permissions on all cluster resources",
			Run:      m1,
			Required: true,
		},
		{
			Id:       "386e797e-4bf8-4a58-bb34-45edba27e8e5",
			Name:     "Convert .nix configuration to mostly .json config files",
			Run:      m2,
			Required: true,
		},
		{
			Id:       "b5a63e29-4b35-48e9-b78f-8f3522225f6f",
			Name:     "Add a nix.json config file which enables automatic, weekly garbage collection",
			Run:      m3,
			Required: true,
		},
	}
)

func NewMigrator(logger chassis.Logger) Migrator {
	return migrator{
		logger: logger,
	}
}

func (m migrator) Migrate() {
	m.logger.Info("running migrations")

	history := migrationsHistory{}
	f, err := os.ReadFile(MigrationsFile())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			m.logger.Info("no migrations history file")
			history.Migrations = []migrationRun{}
		} else {
			m.logger.WithError(err).Panic("failed to open migrations history file")
		}
	} else {
		err = yaml.Unmarshal(f, &history)
		if err != nil {
			m.logger.WithError(err).Panic("failed to unmarshal migrations history file")
		}
	}

	for _, l := range migrationsList {
		complete := false
		for _, h := range history.Migrations {
			if l.Id == h.Id {
				complete = true
				break
			}
		}
		if !complete {
			log := m.logger.WithFields(
				chassis.Fields{
					"id":   l.Id,
					"name": l.Name,
				},
			)
			r := migrationRun{
				Id:        l.Id,
				Name:      l.Name,
				Timestamp: time.Now(),
			}
			log.Info("running migration")
			err := l.Run(m.logger)
			if err != nil {
				// TODO: send error to server
				log.WithError(err).Error("failed to run migration")
				if l.Required {
					m.logger.Panic("failed to run required migration")
				}
				r.Error = err.Error()
			}
			history.Migrations = append(history.Migrations, r)
		}
	}

	data, err := yaml.Marshal(history)
	if err != nil {
		m.logger.WithError(err).Panic("failed to marshal migrations history")
	}

	err = os.WriteFile(MigrationsFile(), data, 0666)
	if err != nil {
		m.logger.WithError(err).Panic("failed to write migrations history file")
	}

	m.logger.Info("migrations completed")
}

func m1(logger chassis.Logger) error {
	var (
		replacers = []Replacer{}
		fileName  = ServerManifestFile()
	)

	replacers = append(replacers, func(line string) string {
		if line == "  - pods" {
			line = "  - \"*\""
		}
		return line
	})
	replacers = append(replacers, func(line string) string {
		if strings.Contains(line, "read-pods") {
			line = strings.ReplaceAll(line, "read-pods", "read-all")
		}
		return line
	})

	err := LineByLineReplace(fileName, replacers)
	if err != nil {
		return err
	}

	return nil
}

func m2(logger chassis.Logger) error {
	var (
		ctx     = context.Background()
		configs = map[string]any{
			BootConfigFile(): BootConfig{
				Loader: BootConfigLoader{
					SystemdBoot: BootConfigLoaderSystemdBoot{
						Enable: true,
					},
				},
				BCache: BootConfigBCache{
					Enable: false,
				},
			},
			NetworkingConfigFile(): NetworkingConfig{
				Hostname: "home-cloud",
				Domain:   "local",
				NetworkManager: NetworkingConfigNetworkManager{
					Enable: true,
				},
				Wireless: NetworkingConfigWireless{
					Enable: false,
				},
				Firewall: NetworkingConfigFirewall{
					Enable: false,
				},
			},
			SecurityConfigFile(): SecurityConfig{
				Sudo: SecurityConfigSudo{
					WheelNeedsPassword: false,
				},
			},
			ServicesConfigFile(): ServicesConfig{
				Resolved: ServicesConfigResolved{
					Enable:  true,
					Domains: []string{"local"},
				},
				K3s: ServicesConfigK3s{
					Enable:     true,
					Role:       "server",
					ExtraFlags: "--tls-san home-cloud.local --disable traefik --service-node-port-range 80-32767",
				},
				OpenSSH: ServicesConfigOpenSSH{
					Enable: false,
				},
				Avahi: ServicesConfigAvahi{
					Enable:   true,
					IPv4:     true,
					IPv6:     true,
					NSSmDNS4: true,
					Publish: ServicesConfigAvahiPublish{
						Enable:       true,
						Domain:       true,
						Addresses:    true,
						UserServices: true,
					},
				},
			},
			TimeConfigFile(): TimeConfig{
				TimeZone: "Etc/UTC",
			},
			UsersConfigFile(): UsersConfig{
				Users: map[string]User{
					"admin": {
						IsNormalUser: true,
						ExtraGroups:  []string{"wheel"},
						OpenSSH: UserOpenSSH{
							AuthorizedKeys: UserOpenSSHAuthorizedKeys{
								Keys: []string{},
							},
						},
					},
				},
			},
		}
	)
	const (
		nixVarsContents = `
{ lib, ... }:
with lib;
{
  options.vars = {
    root = mkOption {
      type = types.str;
      default = "/etc/nixos";
      description = "";
    };
  };
}
`
		nixConfigurationContents = `
{ config, lib, pkgs, ... }:
let
  home-cloud-daemon = import ./home-cloud/daemon/default.nix;
in
{
  imports = [
    <nixpkgs/nixos/modules/profiles/all-hardware.nix>
    <nixpkgs/nixos/modules/profiles/base.nix>
    ./vars.nix
    ./hardware-configuration.nix
  ];

  boot = lib.importJSON (lib.concatStrings [ config.vars.root "/config/boot.json" ]);
  networking = lib.importJSON (lib.concatStrings [ config.vars.root "/config/networking.json" ]);
  services = lib.importJSON (lib.concatStrings [ config.vars.root "/config/services.json" ]);
  time = lib.importJSON (lib.concatStrings [ config.vars.root "/config/time.json" ]);
  users = lib.importJSON (lib.concatStrings [ config.vars.root "/config/users.json" ]);
  security = lib.importJSON (lib.concatStrings [ config.vars.root "/config/security.json" ]);

  # This service runs the Home Cloud Daemon at boot.
  systemd.services.daemon = {
    enable = true;
    description = "Home Cloud Daemon";
    after = [ "network.target" ];
    serviceConfig = {
      Environment = [
        "DRAFT_CONFIG=/etc/home-cloud/config.yaml"
        "NIX_PATH=/root/.nix-defexpr/channels:nixpkgs=/nix/var/nix/profiles/per-user/root/channels/nixos:nixos-config=/etc/nixos/configuration.nix:/nix/var/nix/profiles/per-user/root/channels"
        "PATH=/run/current-system/sw/bin"
      ];
      Restart = "always";
      RestartSec = 3;
      WorkingDirectory = "/root";
      ExecStart = ''
        ${home-cloud-daemon}/bin/daemon
      '';
    };
    wantedBy = [ "multi-user.target" ];
  };

  environment.systemPackages =
    [
      home-cloud-daemon
      pkgs.avahi
      pkgs.coreutils
      pkgs.curl
      pkgs.nano
      pkgs.openssl
      pkgs.nvd
    ];

  # This option defines the first version of NixOS you have installed on this particular machine,
  # and is used to maintain compatibility with application data (e.g. databases) created on older NixOS versions.
  #
  # Most users should NEVER change this value after the initial install, for any reason,
  # even if you've upgraded your system to a new NixOS release.
  #
  # This value does NOT affect the Nixpkgs version your packages and OS are pulled from,
  # so changing it will NOT upgrade your system - see https://nixos.org/manual/nixos/stable/#sec-upgrading for how
  # to actually do that.
  #
  # This value being lower than the current NixOS release does NOT mean your system is
  # out of date, out of support, or vulnerable.
  #
  # Do NOT change this value unless you have manually inspected all the changes it would make to your configuration,
  # and migrated your data accordingly.
  #
  # For more information, see https://nixos.org/manual/nixos/stable/options#opt-system.stateVersion .
  system.stateVersion = "24.05"; # Did you read the comment?
}
`
	)

	err := os.MkdirAll(NixosConfigsPath(), 0755)
	if err != nil {
		return err
	}

	for path, config := range configs {
		err = WriteJsonFile(path, config, 0600)
		if err != nil {
			return err
		}
	}

	err = os.WriteFile(NixosVarsFile(), []byte(nixVarsContents), 0600)
	if err != nil {
		return err
	}

	err = os.WriteFile(NixosConfigFile(), []byte(nixConfigurationContents), 0600)
	if err != nil {
		return err
	}

	err = RebuildAndSwitchOS(ctx, logger)
	if err != nil {
		return err
	}

	return nil
}

func m3(logger chassis.Logger) error {
	var (
		ctx           = context.Background()
		nixConfigFile = NixConfig{
			GC: NixConfigGC{
				Automatic: true,
				Dates:     "weekly",
				Options:   "--delete-older-than 30d",
			},
		}
		replacers = []Replacer{
			func(line string) string {
				if strings.Contains(line, "boot = lib.importJSON") {
					line = `  boot = lib.importJSON (lib.concatStrings [ config.vars.root "/config/boot.json" ]);
  nix = lib.importJSON (lib.concatStrings [ config.vars.root "/config/nix.json" ]);`
				}
				return line
			},
		}
	)

	// create new nix.json file
	err := WriteJsonFile(NixConfigFile(), nixConfigFile, 0600)
	if err != nil {
		return err
	}

	// reference nix.json file in configuration.json file
	err = LineByLineReplace(NixosConfigFile(), replacers)
	if err != nil {
		return err
	}

	err = RebuildAndSwitchOS(ctx, logger)
	if err != nil {
		return err
	}

	return nil
}
