package host

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/home-cloud-io/core/services/platform/daemon/execute"
	"github.com/steady-bytes/draft/pkg/chassis"
	"gopkg.in/yaml.v3"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
		{
			Id:       "51af2d46-e8e1-4d6f-a578-ea8d62dda7f5",
			Name:     "Upgrade NixOS to the 25.05 channel",
			Run:      m4,
			Required: true,
		},
		{
			Id:       "deda2d99-d791-4c93-8980-fd460a083f40",
			Name:     "Install Kubernetes Gateway API manifests",
			Run:      m5,
			Required: true,
		},
		{
			Id:       "9b970d3e-fbf8-44df-a21e-793e9b76f438",
			Name:     "Install istio in ambient mode with an ingress gateway and a default route to the home-cloud server",
			Run:      m6,
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

func m4(logger chassis.Logger) error {
	ctx := context.Background()

	err := AddChannel(ctx, logger, "https://nixos.org/channels/nixos-25.05", "nixos")
	if err != nil {
		return err
	}

	err = UpdateChannel(ctx, logger)
	if err != nil {
		return err
	}

	err = RebuildUpgradeBoot(ctx, logger)
	if err != nil {
		return err
	}

	execute.Restart(ctx, logger)

	return nil
}

func m5(logger chassis.Logger) error {
	// get crds
	resp, err := http.Get("https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.3.0/standard-install.yaml")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// write to file
	out, err := os.Create(GatewayAPIManifestFile())
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	// wait for crd to be populated (yes this is hacky, but it works and I'm lazy)
	kube := KubeClient()
	for range 30 {
		response, err := kube.RESTClient().Get().AbsPath("/apis/apiextensions.k8s.io/v1/customresourcedefinitions").DoRaw(context.TODO())
		if err != nil {
			return err
		}
		if strings.Count(string(response), "\"group\":\"gateway.networking.k8s.io\"") == 5 {
			break
		}
		logger.Info("Gateway APIs not yet installed...")
		time.Sleep(2 * time.Second)
	}
	logger.Info("Gateway APIs installed")

	return nil
}

func m6(logger chassis.Logger) error {
	const istioManifest = `apiVersion: v1
kind: Namespace
metadata:
  name: istio-system
---
apiVersion: helm.cattle.io/v1
kind: HelmChart
metadata:
  name: istio-base
  namespace: kube-system
spec:
  repo: https://istio-release.storage.googleapis.com/charts
  chart: base
  targetNamespace: istio-system
  version: 1.27.1
---
apiVersion: helm.cattle.io/v1
kind: HelmChart
metadata:
  name: istio-istiod
  namespace: kube-system
spec:
  repo: https://istio-release.storage.googleapis.com/charts
  chart: istiod
  targetNamespace: istio-system
  version: 1.27.1
  valuesContent: |-
    profile: ambient
---
apiVersion: helm.cattle.io/v1
kind: HelmChart
metadata:
  name: istio-cni
  namespace: kube-system
spec:
  repo: https://istio-release.storage.googleapis.com/charts
  chart: cni
  targetNamespace: istio-system
  version: 1.27.1
  valuesContent: |-
    profile: ambient
    global:
      platform: k3s
---
apiVersion: helm.cattle.io/v1
kind: HelmChart
metadata:
  name: istio-ztunnel
  namespace: kube-system
spec:
  repo: https://istio-release.storage.googleapis.com/charts
  chart: ztunnel
  targetNamespace: istio-system
  version: 1.27.1
# ingress gateway
---
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: ingress-gateway
  namespace: istio-system
spec:
  gatewayClassName: istio
  listeners:
  - name: http
    port: 80
    protocol: HTTP
    allowedRoutes:
      namespaces:
        from: All
# default route to home-cloud server
---
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: home-cloud
  namespace: home-cloud-system
spec:
  parentRefs:
  - name: ingress-gateway
    namespace: istio-system
  hostnames: ["home-cloud.local"]
  rules:
  - backendRefs:
    - name: server
      port: 8090
`

	var (
		draftReplacer = []Replacer{
			func(line string) string {
				if line == "  name: draft-system" {
					line = `  name: draft-system
  labels:
    istio.io/dataplane-mode: ambient`
				}
				return line
			},
		}
		serverReplacer = []Replacer{
			func(line string) string {
				if line == "  name: home-cloud-system" {
					line = `  name: home-cloud-system
  labels:
    istio.io/dataplane-mode: ambient`
				}
				return line
			},
		}
	)

	err := os.WriteFile(IstioManifestFile(), []byte(istioManifest), 0600)
	if err != nil {
		return err
	}

	err = LineByLineReplace(DraftManifestFile(), draftReplacer)
	if err != nil {
		return err
	}

	err = LineByLineReplace(ServerManifestFile(), serverReplacer)
	if err != nil {
		return err
	}

	// wait for the ingress gateway to be available
	kube := KubeClient()
	for range 30 {
		deploy, err := kube.AppsV1().Deployments("istio-system").Get(context.Background(), "ingress-gateway-istio", v1.GetOptions{})
		if err != nil {
			if kerrors.IsNotFound(err) {
				logger.Info("waiting on istio ingress gateway to deploy...")
				time.Sleep(2 * time.Second)
				continue
			}
			return err
		}
		var ready bool
		for _, c := range deploy.Status.Conditions {
			if c.Type == appsv1.DeploymentAvailable && c.Status == corev1.ConditionTrue {
				ready = true
			}
		}
		if !ready {
			logger.Info("waiting on istio ingress gateway to be available...")
			time.Sleep(2 * time.Second)
			continue
		}

		break
	}
	logger.Info("istio ingress gateway available")

	return nil
}
