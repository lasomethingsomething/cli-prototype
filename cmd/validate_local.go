package cmd

import (
	"fmt"

	"github.com/lasomethingsomething/cli-prototype/internal/workflow"
	"github.com/spf13/cobra"
)

func newValidateLocalCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "local",
		Short: "Local compliance check of a model directory before pushing",
		Long: `Check a packaged model directory locally before it is pushed: required
annotations in manifest.json, SBOM presence and MOF classification.

Examples:
  model-cli validate local --model-path ./my-model
  model-cli validate local --model-path ./my-model --strict`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			modelName, _ := cmd.Flags().GetString("model")
			modelPath, _ := cmd.Flags().GetString("model-path")
			artifactPath, _ := cmd.Flags().GetString("artifact-path")
			artifactName, _ := cmd.Flags().GetString("artifact")
			strict, _ := cmd.Flags().GetBool("strict")

			if modelPath == "" && artifactPath != "" {
				modelPath = artifactPath
			}
			if err := requireValues("model-path", modelPath); err != nil {
				return err
			}

			checkWorkflow := workflow.NewCheckWorkflow()
			checkWorkflow.SetCheckInfo(modelPath, artifactPath)

			if err := checkWorkflow.Run(); err != nil {
				if strict {
					return err
				}
				fmt.Println()
				fmt.Println("Warning: Compliance check failed but --strict was not set")
				return nil
			}

			fmt.Println()
			fmt.Println("=== Compliance Check Summary ===")
			fmt.Printf("Model:        %s\n", modelName)
			fmt.Printf("Model Path:   %s\n", modelPath)
			fmt.Printf("Artifact:     %s\n", artifactName)
			fmt.Println()
			fmt.Printf("Overall:      %s\n", checkResult(checkWorkflow.Passed()))
			fmt.Printf("SBOM:         %s\n", checkResult(checkWorkflow.SBOMCheck()))
			fmt.Printf("MOF:          %s\n", checkResult(checkWorkflow.MOFCheck()))

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
	c.Flags().StringP("model", "m", "", "Model name")
	c.Flags().StringP("model-path", "p", "", "Path to model directory")
	c.Flags().StringP("artifact-path", "a", "", "Path to artifact directory")
	c.Flags().String("artifact", "", "Artifact name/reference")
	c.Flags().BoolP("strict", "s", false, "Fail on any compliance issue")
	return c
}

func checkResult(passed bool) string {
	if passed {
		return "✓ PASS"
	}
	return "✗ FAIL"
}
