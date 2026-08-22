package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

const version = "0.1.0"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version of cli-prototype",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("cli-prototype v%s\n", version)
	},
}
