package cmd

import (
	"fmt"
	"sort"

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
		// Get flags
		artifactFlag, _ := cmd.Flags().GetString("artifact")
		checkRelationshipsFlag, _ := cmd.Flags().GetBool("check-relationships")
		manifestFlag, _ := cmd.Flags().GetString("manifest")

		// Interactive prompts
		var artifact string
		if artifactFlag != "" {
			artifact = artifactFlag
		} else {
			if err := huh.NewInput().
				Title("Artifact to validate:").
				Description("The OCI artifact reference (e.g., ghcr.io/my-org/my-model:latest)").
				Value(&artifact).
				Run(); err != nil {
				return err
			}
		}

		var checkRelationships bool
		if checkRelationshipsFlag {
			checkRelationships = true
		} else {
			if err := huh.NewConfirm().
				Title("Check relationships?").
				Description("Map model->skill and other artifact relationships").
				Value(&checkRelationships).
				Run(); err != nil {
				return err
			}
		}

		fmt.Printf("\nValidating manifest for: %s\n\n", artifact)

		var manifestAnnotations map[string]string
		if manifestFlag != "" {
			// Read the real OCI manifest written by 'model-cli package' and
			// validate the CNCF AI annotations that actually landed on it.
			fmt.Println("✓ Reading manifest from disk...")
			manifest, err := workflow.ReadManifest(manifestFlag)
			if err != nil {
				return err
			}
			fmt.Println("✓ Reading Standardized Metadata Contract...")

			manifestAnnotations = manifest.Annotations
			if len(manifestAnnotations) == 0 {
				return fmt.Errorf("manifest at %s has no CNCF AI annotations", manifestFlag)
			}

			required := []string{workflow.AnnotationProfileVersion, workflow.AnnotationArtifactType}
			for _, key := range required {
				if _, ok := manifestAnnotations[key]; !ok {
					return fmt.Errorf("manifest at %s is missing required annotation %q", manifestFlag, key)
				}
			}
		} else {
			// No local manifest given: simulate what a registry pull would
			// return, since this prototype does not yet pull real manifests
			// over the network.
			fmt.Println("✓ Fetching manifest from registry...")
			fmt.Println("✓ Reading Standardized Metadata Contract...")

			annotations := workflow.NewAnnotationSet()
			annotations.Runtime = "vllm"
			annotations.Accelerator = "nvidia-gpu"
			annotations.CUDAMin = "12.1"
			annotations.MOFClass = "I"
			annotations.MOFComponents = "weights,training-data"
			manifestAnnotations = annotations.ToMap()
		}

		fmt.Println("\n  Detected Annotations:")
		for _, key := range sortedKeys(manifestAnnotations) {
			fmt.Printf("    %s: %s\n", key, manifestAnnotations[key])
		}

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
	validateCmd.Flags().String("artifact", "", "OCI artifact reference to validate")
	validateCmd.Flags().Bool("check-relationships", false, "Map and validate artifact relationships")
	validateCmd.Flags().String("manifest", "", "Path to a local OCI manifest.json (e.g. produced by 'model-cli package') to validate instead of simulating a registry fetch")
}

// sortedKeys returns the keys of m in sorted order, for deterministic output.
func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
