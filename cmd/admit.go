package cmd

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/lasomethingsomething/cli-prototype/internal/workflow"
	"github.com/spf13/cobra"
)

var admitCmd = &cobra.Command{
	Use:   "admit",
	Short: "Evaluate artifact for GitOps admission",
	Long: `Evaluate an OCI artifact's trust profile for GitOps admission control.

This command handles Phase 3, Step 5: GitOps Admission & Policy Enforcement.
The cluster evaluates the artifact's Trust Profile before allowing deployment.

Specifically checks:
- Valid signature (Sigstore/Notary v2)
- SBOM presence
- Infrastructure dependency matching
- Compliance profile contract

If any check fails, admission is blocked.

Examples:
  model-cli admit
  model-cli admit --artifact my-registry/my-model:latest
  model-cli admit --artifact my-registry/my-model:latest --strict
  model-cli admit --artifact my-registry/my-model:latest --env production`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check for flags
		strictFlag, _ := cmd.Flags().GetBool("strict")
		environmentFlag, _ := cmd.Flags().GetString("env")

		// Interactive prompts
		var artifact string
		if err := huh.NewInput().
			Title("Artifact to evaluate:").
			Description("The OCI artifact reference to check (e.g., ghcr.io/my-org/my-model:latest)").
			Value(&artifact).
			Run(); err != nil {
			return err
		}

		var environment string
		if environmentFlag != "" {
			environment = environmentFlag
		} else {
			if err := huh.NewSelect[string]().
				Title("Target environment:").
				Description("Destination environment for deployment").
				Options(huh.NewOptions("development", "staging", "production", "air-gapped")...).
				Value(&environment).
				Run(); err != nil {
				return err
			}
		}

		strict := strictFlag

		fmt.Printf("\nEvaluating artifact '%s' for admission to '%s'...\n\n", artifact, environment)

		// === Trust Profile Evaluation ===
		fmt.Println("=== Trust Profile ===")

		// Signature check
		fmt.Println("✓ Checking signature...")
		// In real implementation: verify signature using signing provider
		fmt.Println("  ✓ Signature valid (signed with sigstore-cosign)")
		fmt.Println("  ✓ Provenance chain intact")

		// SBOM check
		fmt.Println("✓ Checking SBOM...")
		// In real implementation: check for attached SBOM
		fmt.Println("  ✓ SBOM present (spdx-json)")
		fmt.Println("  ✓ SBOM attached to manifest")

		// Compliance profile
		fmt.Println("✓ Checking compliance profile...")
		// In real implementation: validate compliance contract
		fmt.Println("  ✓ Compliance profile valid")

		// === Infrastructure Dependency Matching ===
		fmt.Println("\n=== Infrastructure Dependencies ===")

		// Get sample annotations to demonstrate
		annotations := workflow.NewAnnotationSet()
		annotations.Runtime = "vllm"
		annotations.Accelerator = "nvidia-gpu"
		annotations.CUDAMin = "12.1"
		annotations.MemoryMin = "24GiB"

		fmt.Println("  Declared requirements:")
		fmt.Printf("    Runtime: %s\n", annotations.Runtime)
		fmt.Printf("    Accelerator: %s\n", annotations.Accelerator)
		fmt.Printf("    CUDA min: %s\n", annotations.CUDAMin)
		fmt.Printf("    Memory min: %s\n", annotations.MemoryMin)

		// Environment-specific matching
		fmt.Println("\n  Environment matching:")
		switch environment {
		case "production":
			fmt.Println("    ✓ Production environment supports nvidia-gpu")
			fmt.Println("    ✓ CUDA 12.1+ available")
			fmt.Println("    ✓ 24GiB+ memory available")
		case "staging":
			fmt.Println("    ✓ Staging environment supports nvidia-gpu")
			fmt.Println("    ✓ CUDA 12.1+ available")
			fmt.Println("    ✓ 24GiB+ memory available")
		case "development":
			fmt.Println("    ✓ Development environment supports nvidia-gpu")
			fmt.Println("    ⚠ CUDA version: 11.8 (below minimum 12.1)")
			if strict {
				fmt.Println("    ✗ ADMISSION DENIED: CUDA version requirement not met")
				return fmt.Errorf("admission denied: CUDA 12.1+ required, found 11.8")
			} else {
				fmt.Println("    ⚠ WARNING: CUDA requirement not met (strict mode disabled)")
			}
		case "air-gapped":
			fmt.Println("    ✓ Air-gapped environment validated")
			fmt.Println("    ✓ All dependencies pre-loaded")
		}

		// === Policy Enforcement ===
		fmt.Println("\n=== Policy Enforcement ===")

		if strict {
			fmt.Println("  ✓ Strict mode enabled")
			fmt.Println("  ✓ All requirements enforced")
		} else {
			fmt.Println("  ⚠ Strict mode disabled")
			fmt.Println("  ⚠ Warnings allowed, not blocking")
		}

		fmt.Println("  ✓ Air-gapped/hybrid-cloud safety policies checked")

		// === Final Decision ===
		fmt.Println("\n=== Admission Decision ===")
		fmt.Println("✓ ARTIFACT ADMITTED")
		fmt.Println("\nThe artifact:")
		fmt.Println("  - Has valid signature and provenance")
		fmt.Println("  - Has required SBOM attached")
		fmt.Println("  - Meets infrastructure dependencies")
		fmt.Println("  - Passes compliance profile validation")
		fmt.Println("\nThe GitOps controller can proceed with deployment.")

		return nil
	},
}

func init() {
	rootCmd.AddCommand(admitCmd)
	admitCmd.Flags().Bool("strict", false, "Strict mode: block admission on warnings")
	admitCmd.Flags().String("env", "", "Target environment (development, staging, production, air-gapped)")
	admitCmd.Flags().String("artifact", "", "OCI artifact reference to evaluate")
}
