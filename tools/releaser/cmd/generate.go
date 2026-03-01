package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"

	"github.com/home-cloud-io/core/services/platform/operator/resources"
)

var (
	path        string
	out         string
	operatorTag string

	crdFiles = []string{
		"home-cloud.io_apps.yaml",
		"home-cloud.io_installs.yaml",
		"home-cloud.io_wireguards.yaml",
	}
)

// generateCmd represents the generate command
var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate Home Cloud release files",
	RunE: func(cmd *cobra.Command, args []string) error {

		fmt.Printf("Input path: %s\n", path)
		fmt.Printf("Output path: %s\n", out)
		fmt.Printf("Operator tag: %s\n", operatorTag)

		err := crdsRelease()
		if err != nil {
			return err
		}

		err = operatorRelease()
		if err != nil {
			return err
		}

		return nil
	},
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

func operatorRelease() error {
	f, err := os.Create(filepath.Join(out, "operator.yaml"))
	if err != nil {
		return err
	}
	defer f.Close()

	install := resources.DefaultInstall
	install.Spec.Operator.Image = "ghcr.io/home-cloud-io/operator"
	install.Spec.Operator.Tag = operatorTag

	objects := resources.OperatorObjects(install)

	for _, obj := range objects {
		data, err := yaml.Marshal(obj)
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
	generateCmd.Flags().StringVarP(&operatorTag, "operator-tag", "t", "latest", "Operator tag")
}
