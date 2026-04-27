package cmd

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/steady-bytes/draft/tools/dctl/input"
	"github.com/steady-bytes/draft/tools/dctl/output"
	"gopkg.in/yaml.v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8syaml "sigs.k8s.io/yaml"

	opv1 "github.com/home-cloud-io/core/services/platform/operator/api/v1"
	"github.com/home-cloud-io/core/services/platform/operator/resources"
	"github.com/home-cloud-io/core/services/platform/operator/server/system"
)

var (
	path string
	out  string

	crdFiles = []string{
		"home-cloud.io_apps.yaml",
		"home-cloud.io_installs.yaml",
		"home-cloud.io_wireguards.yaml",
	}
)

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate Home Cloud release files",
	RunE: func(cmd *cobra.Command, args []string) error {

		fmt.Printf("Input path: %s\n", path)
		fmt.Printf("Output path: %s\n", out)

		spec, err := manifestRelease()
		if err != nil {
			return err
		}

		err = installRelease(spec)
		if err != nil {
			return err
		}

		err = crdsRelease()
		if err != nil {
			return err
		}

		err = operatorRelease(spec)
		if err != nil {
			return err
		}

		return nil
	},
}

func manifestRelease() (*opv1.InstallSpec, error) {
	f, err := os.Create(filepath.Join(out, "manifest.yaml"))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// get version manifest from repo``
	resp, err := http.Get(system.LatestReleaseManifestURL)
	if err != nil {
		return nil, err
	}

	// decode body into spec
	dec := yaml.NewDecoder(resp.Body)
	latest := &opv1.InstallSpec{}
	err = dec.Decode(latest)
	if err != nil {
		return nil, err
	}

	output.Print("Version: %s -> ?", latest.Version)
	latest.Version = GetWithDefault(latest.Version)

	output.Print("Gateway API: %s -> ?", latest.GatewayAPI.Version)
	latest.GatewayAPI.Version = GetWithDefault(latest.GatewayAPI.Version)

	output.Print("Istio: %s -> ?", latest.Istio.Version)
	latest.Istio.Version = GetWithDefault(latest.Istio.Version)

	output.Print("mDNS: %s -> ?", latest.MDNS.Tag)
	latest.MDNS.Tag = GetWithDefault(latest.MDNS.Tag)

	output.Print("Tunnel: %s -> ?", latest.Tunnel.Tag)
	latest.Tunnel.Tag = GetWithDefault(latest.Tunnel.Tag)

	output.Print("Operator: %s -> ?", latest.Operator.Tag)
	latest.Operator.Tag = GetWithDefault(latest.Operator.Tag)

	output.Print("Daemon: %s -> ?", latest.Daemon.Tag)
	latest.Daemon.Tag = GetWithDefault(latest.Daemon.Tag)

	output.Print("Talos: %s -> ?", latest.Daemon.System.Version)
	latest.Daemon.System.Version = GetWithDefault(latest.Daemon.System.Version)

	output.Print("Kubernetes: %s -> ?", latest.Daemon.Kubernetes.Version)
	latest.Daemon.Kubernetes.Version = GetWithDefault(latest.Daemon.Kubernetes.Version)

	data, err := k8syaml.Marshal(latest)
	if err != nil {
		return nil, err
	}
	_, err = f.Write(data)
	if err != nil {
		return nil, err
	}

	return latest, nil
}

func installRelease(spec *opv1.InstallSpec) error {
	f, err := os.Create(filepath.Join(out, "install.yaml"))
	if err != nil {
		return err
	}
	defer f.Close()

	install := &opv1.Install{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "home-cloud.io/v1",
			Kind:       "Install",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      "install",
			Namespace: "home-cloud-system",
			Finalizers: []string{
				"install.home-cloud.io/finalizer",
			},
		},
		Spec: opv1.InstallSpec{
			Version: spec.Version,
		},
	}

	data, err := k8syaml.Marshal(install)
	if err != nil {
		return err
	}
	_, err = f.Write(data)
	if err != nil {
		return err
	}

	return nil
}

func crdsRelease() error {
	f, err := os.Create(filepath.Join(out, "crds.yaml"))
	if err != nil {
		return err
	}
	defer f.Close()

	operatorPath := filepath.Join(path, "services", "platform", "operator")
	crdPath := filepath.Join(operatorPath, "config", "crd", "bases")

	for _, file := range crdFiles {
		data, err := os.ReadFile(filepath.Join(crdPath, file))
		if err != nil {
			return err
		}
		_, err = f.Write(data)
		if err != nil {
			return err
		}
	}

	return nil
}

func operatorRelease(spec *opv1.InstallSpec) error {
	f, err := os.Create(filepath.Join(out, "operator.yaml"))
	if err != nil {
		return err
	}
	defer f.Close()

	install := resources.DefaultInstall
	install.Spec.Operator = &opv1.OperatorSpec{
		Image: spec.Operator.Image,
		Tag:   spec.Operator.Tag,
	}

	objects := resources.OperatorObjects(install)

	for _, obj := range objects {
		data, err := k8syaml.Marshal(obj)
		if err != nil {
			return err
		}
		_, err = f.Write([]byte("---\n"))
		if err != nil {
			return err
		}
		_, err = f.Write(data)
		if err != nil {
			return err
		}
	}

	return nil
}

func init() {
	rootCmd.AddCommand(generateCmd)

	generateCmd.Flags().StringVarP(&path, "path", "p", "../../", "Path to the root of the home-cloud-io/core repository")
	generateCmd.Flags().StringVarP(&out, "out", "o", "out/", "Output path to write generated files to")
}

// TODO: move to github.com/steady-bytes/draft/tools/dctl/input
func GetWithDefault(d string) string {
	i := input.Get()
	if i == "" {
		i = d
	}
	return i
}
