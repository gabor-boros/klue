package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		_, err := fmt.Fprintf(
			cmd.OutOrStdout(),
			"klue %s\n"+
				"commit:  %s\n"+
				"built:   %s\n"+
				"go:      %s\n"+
				"os/arch: %s/%s\n",
			info.version, info.commit, info.date, runtime.Version(), runtime.GOOS, runtime.GOARCH,
		)
		return err
	},
}
