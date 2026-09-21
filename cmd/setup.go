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
		var failures []workflow.InstallToolResult
		for _, tool := range missing {
			fmt.Printf("Installing %s...\n", tool.Name())

			result, err := workflow.InstallTool(tool)
			if err != nil || result.Error != nil {
				// Collect failure but continue
				if result != nil {
					failures = append(failures, *result)
				} else {
					failures = append(failures, workflow.InstallToolResult{
						Tool:   tool,
						Error:  err,
						Stderr: "",
					})
				}
				fmt.Printf("⚠ Failed to install %s\n", tool.Name())
			} else {
				fmt.Printf("✓ Successfully installed %s\n", tool.Name())
			}
		}

		fmt.Println()
		
		// Print summary of failures if any
		if len(failures) > 0 {
			fmt.Printf("Encountered %d failure(s) during installation:\n", len(failures))
			for _, f := range failures {
				fmt.Printf("\n  Tool: %s\n", f.Tool.Name())
				if f.Error != nil {
					fmt.Printf("    Error: %v\n", f.Error)
				}
				if f.Stderr != "" {
					fmt.Printf("    Details: %s\n", f.Stderr)
				}
				fmt.Printf("    Hint: %s\n", f.Tool.InstallInstructions())
			}
		}

		// Print updated report
		fmt.Println()
		fmt.Println("Final status:")
		fmt.Println()
		updatedReport := workflow.CheckAll()
		printDoctorReport(updatedReport)

		// Exit with error if there were failures or anything is still missing
		if len(failures) > 0 || !updatedReport.AllInstalled() {
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
