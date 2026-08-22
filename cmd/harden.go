package cmd

import (
	"fmt"

	"github.com/lasomethingsomething/cli-prototype/internal/workflow"
	"github.com/spf13/cobra"
)

var hardenCmd = &cobra.Command{
	Use:   "harden",
	Short: "Apply local hardening and compliance to model artifact",
	Long: `Local Hardening & Compliance

Performs local hardening on your AI model artifact before pushing to a registry.
This includes:
  • SBOM (Software Bill of Materials) generation
  • MOF (Model Openness Framework) classification
  • MOF metadata config file generation with CC-BY-4.0 license by default
  • Security annotations

The hardening workflow ensures your model meets compliance requirements
before it's shared or deployed.

Examples:
  # Harden with defaults (CC-BY-4.0 license)
  model-cli harden --model-path ./models --model phi-4-mini --artifact my-model:v1

  # Harden with custom license
  model-cli harden --model-path ./models --model phi-4-mini --artifact my-model:v1 --license MIT

  # Harden with specific SBOM tool
  model-cli harden --model-path ./models --model phi-4-mini --artifact my-model:v1 --sbom-tool syft --sbom-format spdx-json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get flags
		modelName, _ := cmd.Flags().GetString("model")
		modelPath, _ := cmd.Flags().GetString("model-path")
		artifactName, _ := cmd.Flags().GetString("artifact")
		registry, _ := cmd.Flags().GetString("registry")
		license, _ := cmd.Flags().GetString("license")
		generateSBOM, _ := cmd.Flags().GetBool("generate-sbom")
		includeMOF, _ := cmd.Flags().GetBool("include-mof")
		sbomTool, _ := cmd.Flags().GetString("sbom-tool")
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
		
		// Create hardening workflow
		hw := workflow.NewHardenWorkflow(registry)
		hw.SetHardenInfo(modelName, modelPath, artifactName)
		hw.SetOptions(generateSBOM, includeMOF)
		hw.SetSBOMTool(sbomTool, workflow.SBOMFormat(sbomFormat))
		hw.SetLicense(license)
		
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
		if hw.Annotations() != nil {
			fmt.Println("  Annotations:")
			for key, value := range hw.Annotations().ToMap() {
				fmt.Printf("    %s: %s\n", key, value)
			}
		}
		
		return nil
	},
}

func init() {
	rootCmd.AddCommand(hardenCmd)
	
	// Model flags
	hardenCmd.Flags().String("model", "", "Name of the model")
	hardenCmd.Flags().String("model-path", "", "Path to the model directory")
	hardenCmd.Flags().String("artifact", "", "Artifact name (e.g., my-org/my-model:v1.0.0)")
	hardenCmd.Flags().String("registry", "", "Registry to use (e.g., ghcr.io)")
	
	// Hardening options
	hardenCmd.Flags().Bool("generate-sbom", true, "Generate SBOM")
	hardenCmd.Flags().Bool("include-mof", true, "Include MOF classification")
	hardenCmd.Flags().String("license", "CC-BY-4.0", "License for MOF metadata (default: CC-BY-4.0)")
	
	// SBOM options
	hardenCmd.Flags().String("sbom-tool", "syft", "SBOM generation tool (syft, trivy, cdxgen)")
	hardenCmd.Flags().String("sbom-format", "spdx-json", "SBOM format (spdx-json, cyclonedx, spdx)")
	
	// Mark required flags
	hardenCmd.MarkFlagRequired("model")
	hardenCmd.MarkFlagRequired("model-path")
	hardenCmd.MarkFlagRequired("artifact")
}
