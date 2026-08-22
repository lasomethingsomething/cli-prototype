package cmd

import (
	"fmt"

	"github.com/lasomethingsomething/cli-prototype/internal/workflow"
	"github.com/spf13/cobra"
)

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Local compliance check before pushing artifact",
	Long: `Local Compliance Check

Validates a locally-built AI artifact for compliance before pushing to a registry.

This command checks that all required components are present:
  • Required annotations (org.cncf.ai.artifact.type, runtime, accelerator)
  • SBOM (Software Bill of Materials)
  • MOF (Model Openness Framework) classification

The check runs against local artifact files/layers before push.

Examples:
  # Check a local artifact
  model-cli check --model-path ./models --artifact-path ./output

  # Check with explicit paths
  model-cli check --model phi-4-mini --model-path ./models/my-model --artifact my-model:v1

  # Fail on any compliance issue
  model-cli check --strict`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get flags
		modelName, _ := cmd.Flags().GetString("model")
		modelPath, _ := cmd.Flags().GetString("model-path")
		artifactPath, _ := cmd.Flags().GetString("artifact-path")
		artifactName, _ := cmd.Flags().GetString("artifact")
		strict, _ := cmd.Flags().GetBool("strict")

		// Use artifact-path as model-path if not specified
		if modelPath == "" && artifactPath != "" {
			modelPath = artifactPath
		}

		// Create check workflow
		checkWorkflow := workflow.NewCheckWorkflow()
		checkWorkflow.SetCheckInfo(modelPath, artifactPath)

		// Run the check
		err := checkWorkflow.Run()

		if err != nil {
			if strict {
				return err
			}
			// In non-strict mode, just print the error but don't fail
			fmt.Println()
			fmt.Println("Warning: Compliance check failed but --strict was not set")
			return nil
		}

		// Print summary
		fmt.Println()
		fmt.Println("=== Compliance Check Summary ===")
		fmt.Printf("Model:        %s\n", modelName)
		fmt.Printf("Model Path:   %s\n", modelPath)
		fmt.Printf("Artifact:     %s\n", artifactName)
		fmt.Println()
		fmt.Printf("Overall:      %s\n", checkResult(checkWorkflow.Passed()))
		fmt.Printf("SBOM:        %s\n", checkResult(checkWorkflow.SBOMCheck()))
		fmt.Printf("MOF:         %s\n", checkResult(checkWorkflow.MOFCheck()))

		if len(checkWorkflow.Missing()) > 0 {
			fmt.Println()
			fmt.Println("Missing required items:")
			for _, item := range checkWorkflow.Missing() {
				fmt.Printf("  ✗ %s\n", item)
			}
		}

		if !checkWorkflow.Passed() && strict {
			return fmt.Errorf("compliance check failed")
		}

		return nil
	},
}

func checkResult(passed bool) string {
	if passed {
		return "✓ PASS"
	}
	return "✗ FAIL"
}

func init() {
	rootCmd.AddCommand(checkCmd)
	checkCmd.Flags().StringP("model", "m", "", "Model name")
	checkCmd.Flags().StringP("model-path", "p", "", "Path to model directory")
	checkCmd.Flags().StringP("artifact-path", "a", "", "Path to artifact directory")
	checkCmd.Flags().String("artifact", "", "Artifact name/reference")
	checkCmd.Flags().BoolP("strict", "s", false, "Fail on any compliance issue")
}
