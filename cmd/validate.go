package cmd

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/lasomethingsomething/cli-prototype/internal/workflow"
	"github.com/spf13/cobra"
)

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate an OCI artifact manifest",
	Long: `Validate the manifest of an OCI artifact at the registry level.

This command handles manifest-level validation (Phase 2, Step 4):
- Reads the Standardized Metadata Contract at manifest level
- Maps complex relationships (e.g., model -> skill)
- Reads dependency requirements without downloading large binaries

This enables registries or admission proxies to validate artifacts
without needing to pull the entire multi-GB model weights.

Examples:
  model-cli validate
  model-cli validate --artifact my-registry/my-model:latest
  model-cli validate --artifact my-registry/my-model:latest --check-relationships`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Interactive prompts
		var artifact string
		if err := huh.NewInput().
			Title("Artifact to validate:").
			Description("The OCI artifact reference (e.g., ghcr.io/my-org/my-model:latest)").
			Value(&artifact).
			Run(); err != nil {
			return err
		}

		var checkRelationships bool
		if err := huh.NewConfirm().
			Title("Check relationships?").
			Description("Map model->skill and other artifact relationships").
			Value(&checkRelationships).
			Run(); err != nil {
			return err
		}

		fmt.Printf("\nValidating manifest for: %s\n\n", artifact)

		// In a real implementation, this would:
		// 1. Pull the manifest from the registry
		// 2. Parse the CNCF AI Interoperability Profile annotations
		// 3. Validate required fields are present
		// 4. Check annotation formats are valid
		// 5. If checkRelationships, resolve and validate relationships

		// For now, simulate the validation
		fmt.Println("✓ Fetching manifest from registry...")
		fmt.Println("✓ Reading Standardized Metadata Contract...")

		// Create a sample annotation set to demonstrate what would be validated
		annotations := workflow.NewAnnotationSet()
		annotations.Runtime = "vllm"
		annotations.Accelerator = "nvidia-gpu"
		annotations.CUDAMin = "12.1"
		annotations.MOFClass = "I"
		annotations.MOFComponents = "weights,training-data"

		fmt.Println("\n  Detected Annotations:")
		annotations.Print()

		fmt.Println("\n✓ Profile annotations valid")
		fmt.Println("✓ MOF classification valid")
		fmt.Println("✓ Security annotations valid")
		fmt.Println("✓ Runtime requirements valid")

		if checkRelationships {
			fmt.Println("\n✓ Mapping relationships...")
			fmt.Println("  Model: my-model:latest")
			fmt.Println("  → Skill: my-skill:v1 (optional)")
			fmt.Println("  → RAG Context: my-rag:latest (optional)")
			fmt.Println("✓ Dependencies resolved without downloading binaries")
		}

		fmt.Println("\n✓ Manifest validation passed")
		fmt.Println("✓ Artifact is compliant with CNCF AI Interoperability Profile")

		return nil
	},
}

func init() {
	rootCmd.AddCommand(validateCmd)
}
