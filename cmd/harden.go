package cmd

import (
	"fmt"
	"sort"

	"github.com/charmbracelet/huh"
	"github.com/lasomethingsomething/cli-prototype/internal/workflow"
	"github.com/spf13/cobra"
)

var hardenCmd = &cobra.Command{
	Use:   "harden",
	Short: "Apply local hardening and compliance to model artifact",
	Long: `Local Hardening & Compliance

Performs local hardening on a packaged AI model artifact before pushing to a
registry. Run it on the same --model-path after 'model-cli package'; it
records its results in the manifest.json that packaging wrote.
This includes:
  • SBOM (Software Bill of Materials) generation with syft (recommended), trivy or cdxgen
  • MOF (Model Openness Framework) classification, detected from the model files
  • MOF metadata config file generation with CC-BY-4.0 license by default
  • Security annotations

The hardening workflow ensures your model meets compliance requirements
before it's shared or deployed.

Supported SBOM tools (mutually exclusive):
` + workflow.SBOMToolOptions().Bullets() + `

Examples:
  # Package first, then harden with defaults (syft, CC-BY-4.0 license)
  model-cli package --model phi-4-mini --model-path ./models --artifact my-model:v1
  model-cli harden --model-path ./models --model phi-4-mini --artifact my-model:v1

  # Harden with custom license
  model-cli harden --model-path ./models --model phi-4-mini --artifact my-model:v1 --license MIT

  # Harden with specific SBOM tool
  model-cli harden --model-path ./models --model phi-4-mini --artifact my-model:v1 --sbom-tool syft --sbom-format spdx-json

  # Harden with Trivy
  model-cli harden --model-path ./models --model phi-4-mini --artifact my-model:v1 --sbom-tool trivy --sbom-format cyclonedx-json

  # Declare the MOF class instead of detecting it
  model-cli harden --model-path ./models --model phi-4-mini --artifact my-model:v1 --mof-class II`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get flags
		modelName, _ := cmd.Flags().GetString("model")
		modelPath, _ := cmd.Flags().GetString("model-path")
		artifactName, _ := cmd.Flags().GetString("artifact")
		registry, _ := cmd.Flags().GetString("registry")
		license, _ := cmd.Flags().GetString("license")
		generateSBOM, _ := cmd.Flags().GetBool("generate-sbom")
		includeMOF, _ := cmd.Flags().GetBool("include-mof")
		sbomFormat, _ := cmd.Flags().GetString("sbom-format")

		if modelName == "" {
			return fmt.Errorf("model name is required")
		}
		if modelPath == "" {
			return fmt.Errorf("model-path is required")
		}
		if artifactName == "" {
			return fmt.Errorf("artifact name is required")
		}

		// SBOM tool: the flag wins, otherwise offer the supported tools with
		// syft preselected as the recommended one.
		sbomTool := "syft"
		if generateSBOM {
			if err := askSelectLabeled(cmd, "sbom-tool", &sbomTool, "SBOM tool:", "Which tool should generate the Software Bill of Materials?", []huh.Option[string]{
				huh.NewOption("syft (recommended)", "syft"),
				huh.NewOption("trivy", "trivy"),
				huh.NewOption("cdxgen", "cdxgen"),
			}); err != nil {
				return err
			}
		}

		// MOF classification: detected from the model files unless declared.
		annotations := workflow.NewAnnotationSet()
		if includeMOF {
			if err := askSelectLabeled(cmd, "mof-class", &annotations.MOFClass, "MOF Class:", "Model Openness Framework classification (auto = detect from the model files)", []huh.Option[string]{
				huh.NewOption("auto", ""),
				huh.NewOption("I - open weights, code, training data, docs and license", "I"),
				huh.NewOption("II - open weights plus code, data or docs", "II"),
				huh.NewOption("III - weights only", "III"),
			}); err != nil {
				return err
			}

			if err := askString(cmd, "mof-components", &annotations.MOFComponents, "MOF Components:", "Comma-separated MOF components (e.g., weights,training-data,code); leave empty to detect"); err != nil {
				return err
			}
		}

		// Create hardening workflow
		hw := workflow.NewHardenWorkflow(registry)
		hw.SetHardenInfo(modelName, modelPath, artifactName)
		hw.SetOptions(generateSBOM, includeMOF)
		hw.SetSBOMTool(sbomTool, workflow.SBOMFormat(sbomFormat))
		hw.SetLicense(license)
		hw.SetAnnotations(annotations)

		// Run the workflow
		fmt.Println()
		if err := hw.Run(); err != nil {
			return err
		}

		// Print results
		fmt.Println()
		fmt.Println("Hardening Complete!")
		fmt.Println("==================")
		if hw.SBOMPath() != "" {
			fmt.Printf("  SBOM: %s\n", hw.SBOMPath())
		}
		if hw.MOFClass() != "" {
			fmt.Printf("  MOF Class: %s\n", hw.MOFClass())
		}
		if hw.MOFConfigPath() != "" {
			fmt.Printf("  MOF Config: %s\n", hw.MOFConfigPath())
		}
		fmt.Printf("  Manifest: %s\n", hw.ManifestPath())
		if applied := hw.AppliedAnnotations(); len(applied) > 0 {
			fmt.Println("  Annotations recorded:")
			keys := make([]string, 0, len(applied))
			for key := range applied {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			for _, key := range keys {
				fmt.Printf("    %s: %s\n", key, applied[key])
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(hardenCmd)

	// Model flags
	hardenCmd.Flags().String("model", "", "Name of the model")
	hardenCmd.Flags().String("model-path", "", "Path to the packaged model directory (must contain manifest.json from 'model-cli package')")
	hardenCmd.Flags().String("artifact", "", "Artifact name (e.g., my-org/my-model:v1.0.0)")
	hardenCmd.Flags().String("registry", "", "Registry to use (e.g., ghcr.io)")

	// Hardening options
	hardenCmd.Flags().Bool("generate-sbom", true, "Generate SBOM")
	hardenCmd.Flags().Bool("include-mof", true, "Include MOF classification")
	hardenCmd.Flags().String("license", "CC-BY-4.0", "License for MOF metadata (default: CC-BY-4.0)")
	hardenCmd.Flags().String("mof-class", "", "MOF Class: I, II, or III (default: detected from the model files)")
	hardenCmd.Flags().String("mof-components", "", "MOF components, comma-separated (default: detected from the model files)")

	// SBOM options
	hardenCmd.Flags().String("sbom-tool", workflow.SBOMToolOptions().Recommended(), "SBOM generation tool: "+workflow.SBOMToolOptions().Summary())
	hardenCmd.Flags().String("sbom-format", "spdx-json", "SBOM format (spdx-json, cyclonedx-json, spdx-tag-value, cyclonedx-xml)")

	// Mark required flags
	hardenCmd.MarkFlagRequired("model")
	hardenCmd.MarkFlagRequired("model-path")
	hardenCmd.MarkFlagRequired("artifact")
}
