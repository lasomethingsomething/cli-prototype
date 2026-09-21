package cmd

import (
	"fmt"
	"os"

	"github.com/lasomethingsomething/cli-prototype/internal/workflow"
	"github.com/spf13/cobra"
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Install all prerequisite tools for model-cli",
	Long: `The setup command checks all prerequisite tools and installs any missing
brew-installable tools in a single operation. It:
  1. Checks what tools are already installed
  2. Prompts once to install all missing brew-installable tools
  3. Installs everything missing via brew
  4. Re-runs the check and shows the final status

This ensures you have all prerequisites before starting the Test Drive.

Examples:
  model-cli setup              # Interactive: prompt to install all missing tools
  model-cli setup --yes        # Non-interactive: install all missing tools without prompting`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get the report
		report := workflow.CheckAll()

		// Get missing brew tools
		missing := report.MissingBrewTools()

		if len(missing) == 0 {
			fmt.Println("All prerequisite tools are already installed.")
			fmt.Println()
			printDoctorReport(report)
			return nil
		}

		// Show what's missing
		fmt.Printf("Found %d missing brew-installable tool(s):\n", len(missing))
		for _, tool := range missing {
			fmt.Printf("  - %s: %s\n", tool.Name(), tool.Description())
		}
		fmt.Println()

		// Check if we should prompt
		yesFlag, _ := cmd.Flags().GetBool("yes")

		if !yesFlag {
			if !interactive() {
				return fmt.Errorf("setup requires interactive mode or --yes flag")
			}

			var proceed bool
			if err := askConfirm(cmd, "yes", &proceed,
				"Install all missing brew-installable tools?",
				"This will run 'brew install' for each missing tool"); err != nil {
				return err
			}

			if !proceed {
				fmt.Println("Aborted.")
				return nil
			}
		}

		// Install each missing tool
		fmt.Println()
		for _, tool := range missing {
			fmt.Printf("Installing %s...\n", tool.Name())

			if err := workflow.InstallTool(tool); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				fmt.Fprintf(os.Stderr, "Failed to install %s. Run: %s\n", tool.Name(), tool.InstallInstructions())
				os.Exit(1)
			}

			fmt.Printf("Successfully installed %s\n", tool.Name())
		}

		fmt.Println()
		fmt.Println("All missing brew-installable tools have been installed.")
		fmt.Println()

		// Print updated report
		fmt.Println("Final status:")
		fmt.Println()
		updatedReport := workflow.CheckAll()
		printDoctorReport(updatedReport)

		// Exit with error if anything is still missing
		if !updatedReport.AllInstalled() {
			os.Exit(1)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(setupCmd)

	// Flags for setup command
	setupCmd.Flags().Bool("yes", false, "Non-interactive: install all missing tools without prompting")
}
