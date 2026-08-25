package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

// Version is the current version of Model CLI.
const Version = "0.1.0"

// Commit is the git commit hash, injected at build time via
// -ldflags "-X github.com/lasomethingsomething/cli-prototype/cmd.Commit=...".
var Commit string

// Date is the build date, injected at build time the same way as Commit.
var Date string

// GetVersion returns the version string including build metadata when present.
func GetVersion() string {
	v := Version
	if Commit != "" {
		v += " (commit: " + Commit
		if Date != "" {
			v += ", built: " + Date
		}
		v += ")"
	}
	return v
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the model-cli version",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Fprintf(cmd.OutOrStdout(), "model-cli %s\n", GetVersion())
		fmt.Fprintf(cmd.OutOrStdout(), "go: %s %s/%s\n", runtime.Version(), runtime.GOOS, runtime.GOARCH)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
