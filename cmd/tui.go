package cmd

import (
	"fmt"
	"os"

	"github.com/lasomethingsomething/cli-prototype/internal/tui"
	"github.com/spf13/cobra"
)

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Launch the interactive TUI",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := tui.Run(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return err
		}
		return nil
	},
}
