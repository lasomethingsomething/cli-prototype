package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "cli-prototype",
	Short: "A CLI/TUI prototype built with Cobra and Charmbracelet",
	Long: `cli-prototype is a skeleton project demonstrating how to combine
Cobra (for CLI commands) with Charmbracelet's Bubble Tea (for interactive TUIs)
and Lip Gloss (for terminal styling).`,
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(tuiCmd)
}
