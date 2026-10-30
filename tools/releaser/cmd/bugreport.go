package cmd

import (
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/printers"

	bugreport "github.com/home-cloud-io/core/pkg/bug-report"
	"github.com/home-cloud-io/core/pkg/kube"
)

var (
	root         string
	skippedTypes []string
	archive      bool
)

var bugReportCmd = &cobra.Command{
	Use:     "bug-report",
	Aliases: []string{"br", "report"},
	Short:   "Generate a bug report",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		c, err := kube.NewClient()
		if err != nil {
			return err
		}

		printer := bugreport.Printer{
			Root:         root,
			SkippedTypes: skippedTypes,
			Client: c,
			Printer: &printers.OmitManagedFieldsPrinter{
				Delegate: &printers.YAMLPrinter{},
			},
		}

		err = printer.Print(ctx)
		if err != nil {
			return err
		}

		if archive {
			err := printer.Archive()
			if err != nil {
				return err
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(bugReportCmd)

	bugReportCmd.Flags().StringVarP(&root, "out", "o", ".bug-report", "output directory for bug report")
	bugReportCmd.Flags().StringSliceVar(&skippedTypes, "skip", []string{}, "types to skip (e.g. pods, secrets)")
	bugReportCmd.Flags().BoolVarP(&archive, "archive", "a", false, "archive the report as a tar")
}
