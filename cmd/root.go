// Package cmd implements the klue command-line interface.
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// buildInfo holds the version metadata populated from the main package at
// startup. It is consumed by the version command.
type buildInfo struct {
	version string
	commit  string
	date    string
}

var info buildInfo

var rootCmd = &cobra.Command{
	Use:   "klue",
	Short: "klue is a command-line tool",
	Long: `klue is a command-line tool.

Run "klue version" to print build information.`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute stores the provided build metadata and runs the root command. It is
// the single entrypoint called from main.
func Execute(version, commit, date string) {
	info = buildInfo{version: version, commit: commit, date: date}

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringP("namespace", "n", "default", "The namespace to diagnose the resource in")
	rootCmd.PersistentFlags().String("kubeconfig", "", "Path to the kubeconfig file (defaults to standard discovery)")
	rootCmd.PersistentFlags().String("context", "", "The kubeconfig context to use")
	registerRootFlags()

	// Single "why" command for built-in and custom resources.
	rootCmd.AddCommand(newWhyCommand())

	// General commands
	rootCmd.AddCommand(versionCmd)
}
