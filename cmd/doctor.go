package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/lasomethingsomething/cli-prototype/internal/workflow"
	"github.com/spf13/cobra"
)

var doctorFixFlag bool
var doctorYesFlag bool

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check and fix prerequisite tools and dependencies",
	Long: `The doctor command checks all prerequisite tools and dependencies needed for
model-cli. It reports which tools are installed or missing and provides
installation instructions.

Categories:
  - brew-installable: Tools that can be auto-installed via 'brew install'
  - environment: System-level prerequisites (Xcode CLT, SSH keys, etc.)
  - cluster: Kubernetes cluster-side dependencies (Flux, KServe, etc.)

With --fix: Automatically installs all missing brew-installable tools. Prompts for
confirmation before installing each tool and stops on the first failure.

Examples:
  model-cli doctor              # Check all prerequisites
  model-cli doctor --fix       # Auto-install missing brew-installable tools (interactive)
  model-cli doctor --fix --yes # Auto-install without prompting
  model-cli doctor --list      # List all checked tools`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check if we should list tools
		listFlag, _ := cmd.Flags().GetBool("list")
		if listFlag {
			return listTools()
		}

		// Check if we should fix
		fixFlag, _ := cmd.Flags().GetBool("fix")
		yesFlag, _ := cmd.Flags().GetBool("yes")
		
		// Get the report
		report := workflow.CheckAll()
		
		if fixFlag {
			return runFix(cmd, report, yesFlag)
		}
		
		// Print the report (read-only mode)
		printDoctorReport(report)
		
		// Check exit code: exit 1 if anything is missing
		if !report.AllInstalled() {
			os.Exit(1)
		}
		
		return nil
	},
}

// runFix handles the --fix flag logic
func runFix(cmd *cobra.Command, report *workflow.DoctorReport, yesFlag bool) error {
	// Get missing brew tools
	missing := report.MissingBrewTools()
	
	if len(missing) == 0 {
		fmt.Println("All brew-installable tools are already installed.")
		return nil
	}
	
	fmt.Printf("Found %d missing brew-installable tool(s):\n", len(missing))
	for _, tool := range missing {
		fmt.Printf("  - %s: %s\n", tool.Name(), tool.Description())
	}
	fmt.Println()
	
	// If not --yes, ask for confirmation
	if !yesFlag {
		if !interactive() {
			return fmt.Errorf("fix requires interactive mode or --yes flag")
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
	for _, tool := range missing {
		fmt.Printf("Installing %s...\n", tool.Name())
		
		if err := workflow.InstallTool(tool); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			fmt.Fprintf(os.Stderr, "Failed to install %s. Run: %s\n", tool.Name(), tool.InstallInstructions())
			os.Exit(1)
		}
		
		fmt.Printf("Successfully installed %s\n", tool.Name())
	}
	
	fmt.Println("All missing brew-installable tools have been installed.")
	
	// Print updated report
	fmt.Println()
	fmt.Println("Updated status:")
	updatedReport := workflow.CheckAll()
	printDoctorReport(updatedReport)
	
	return nil
}

func init() {
	rootCmd.AddCommand(doctorCmd)
	
	// Flags for doctor command
	doctorCmd.Flags().BoolVar(&doctorFixFlag, "fix", false, "Auto-install missing brew-installable tools")
	doctorCmd.Flags().BoolVar(&doctorYesFlag, "yes", false, "Non-interactive: install all missing tools without prompting (use with --fix)")
	doctorCmd.Flags().Bool("list", false, "List all tools that doctor checks")
}

// printDoctorReport prints the doctor report to stdout
func printDoctorReport(report *workflow.DoctorReport) {
	fmt.Println("model-cli Doctor")
	fmt.Println("===============")
	fmt.Println()
	
	// Group by category
	brewTools := make([]workflow.ToolResult, 0)
	envTools := make([]workflow.ToolResult, 0)
	clusterTools := make([]workflow.ToolResult, 0)
	
	for _, result := range report.Results {
		switch result.Tool.Category() {
		case workflow.CategoryBrew:
			brewTools = append(brewTools, result)
		case workflow.CategoryEnvironment:
			envTools = append(envTools, result)
		case workflow.CategoryCluster:
			clusterTools = append(clusterTools, result)
		}
	}
	
	// Print each category
	printCategory("Brew-installable tools", brewTools)
	fmt.Println()
	printCategory("Environment prerequisites", envTools)
	fmt.Println()
	printCategory("Cluster-side dependencies", clusterTools)
	
	// Summary
	fmt.Println()
	fmt.Println("Summary")
	fmt.Println("-------")
	
	brewMissing, brewTotal, otherMissing := report.DoctorSummary()
	
	if brewMissing > 0 {
		fmt.Printf("⚠ %d/%d brew-installable tools missing\n", brewMissing, brewTotal)
	} else {
		fmt.Printf("✓ All %d brew-installable tools installed\n", brewTotal)
	}
	
	if otherMissing > 0 {
		fmt.Printf("⚠ %d environment/cluster tools need attention\n", otherMissing)
	}
	
	if report.AllInstalled() {
		fmt.Println("✓ All prerequisites are satisfied!")
	}
}

// printCategory prints results for a single category
func printCategory(title string, results []workflow.ToolResult) {
	fmt.Printf("%s\n", title)
	fmt.Println(strings.Repeat("-", len(title)))
	
	if len(results) == 0 {
		fmt.Println("  (none)")
		return
	}
	
	for _, result := range results {
		status := "✓"
		if !result.Installed {
			status = "✗"
		}
		
		fmt.Printf("  %s %-15s %s\n", status, result.Tool.Name(), result.Tool.Description())
		
		if !result.Installed {
			fmt.Printf("      Install: %s\n", result.Tool.InstallInstructions())
		}
	}
}

// listTools lists all tools that doctor checks
func listTools() error {
	fmt.Println("Tools checked by doctor:")
	fmt.Println()
	
	for _, tool := range workflow.AllTools() {
		fmt.Printf("  %-15s [%s] %s\n", tool.Name(), tool.Category(), tool.Description())
		fmt.Printf("                Install: %s\n", tool.InstallInstructions())
		fmt.Println()
	}
	
	return nil
}
