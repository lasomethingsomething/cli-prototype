package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/lasomethingsomething/cli-prototype/internal/workflow"
	"github.com/spf13/cobra"
)

var enforceCmd = &cobra.Command{
	Use:   "enforce",
	Short: "Enforce Standardized Metadata Contract at manifest level",
	Long: `Enforce the Standardized Metadata Contract at manifest level for registry validation.

This command handles Phase 2, Step 4.5: Enforce Standardized Metadata Contract.
It simulates what a registry or admission proxy would do when receiving
an OCI manifest push request.

The command:
- Reads and parses the Standardized Metadata Contract from manifest annotations
- Validates required fields (e.g., model.framework, skill.pipeline_ref)
- Rejects pushes with invalid or missing metadata
- Maps relationships (e.g., model → skill → pipeline)
- Validates dependencies without downloading binaries

Examples:
  model-cli enforce
  model-cli enforce --manifest ./manifest.json
  model-cli enforce --manifest ./manifest.json --strict
  model-cli enforce --artifact my-registry/my-model:v1 --simulate-push
  model-cli enforce --webhook --port 8443`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get flags
		manifestFlag, _ := cmd.Flags().GetString("manifest")
		artifactFlag, _ := cmd.Flags().GetString("artifact")
		strictFlag, _ := cmd.Flags().GetBool("strict")
		webhookFlag, _ := cmd.Flags().GetBool("webhook")
		portFlag, _ := cmd.Flags().GetInt("port")
		simulatePushFlag, _ := cmd.Flags().GetBool("simulate-push")
		artifactTypeFlag, _ := cmd.Flags().GetString("artifact-type")

		// If webhook mode, start the admission proxy server
		if webhookFlag {
			return runWebhookServer(portFlag)
		}

		// Interactive prompts for manifest path
		var manifestPath string
		if manifestFlag != "" {
			manifestPath = manifestFlag
		} else {
			if err := huh.NewInput().
				Title("Manifest path:").
				Description("Path to the OCI manifest JSON file to validate").
				Value(&manifestPath).
				Run(); err != nil {
				return err
			}
		}

		// Read the manifest
		manifest, err := workflow.ReadUnifiedOCIManifest(manifestPath)
		if err != nil {
			return fmt.Errorf("failed to read manifest: %v", err)
		}

		// If artifact flag is provided, use it to simulate a push
		if artifactFlag != "" {
			manifest.Annotations["org.opencontainers.image.title"] = artifactFlag
		}

		// Determine or prompt for artifact type
		var artifactType workflow.ArtifactType
		if artifactTypeFlag != "" {
			artifactType = workflow.ArtifactType(artifactTypeFlag)
		} else if at, ok := manifest.Annotations[workflow.AnnotationArtifactType]; ok {
			artifactType = workflow.ArtifactType(at)
		} else {
			if err := huh.NewSelect[workflow.ArtifactType]().
				Title("Artifact type:").
				Description("Select the type of AI artifact").
				Options(
					huh.NewOption(string(workflow.ArtifactTypeModel), workflow.ArtifactTypeModel),
					huh.NewOption(string(workflow.ArtifactTypeSkill), workflow.ArtifactTypeSkill),
					huh.NewOption(string(workflow.ArtifactTypePipeline), workflow.ArtifactTypePipeline),
				).
				Value(&artifactType).
				Run(); err != nil {
				return err
			}
		}

		// Set artifact type annotation if not already set
		if _, ok := manifest.Annotations[workflow.AnnotationArtifactType]; !ok {
			manifest.Annotations[workflow.AnnotationArtifactType] = string(artifactType)
		}

		fmt.Printf("\n=== Enforcing Standardized Metadata Contract ===\n\n")
		fmt.Printf("Artifact: %s\n", manifest.Annotations["org.opencontainers.image.title"])
		fmt.Printf("Artifact Type: %s\n\n", artifactType)

		// Check if we need to add a sample metadata contract
		if _, hasContract := manifest.Annotations[workflow.AnnotationMetadataContract]; !hasContract && simulatePushFlag {
			fmt.Println("⚠ No metadata contract found in manifest annotations.")
			
			// Prompt to add one
			var addContract bool
			if err := huh.NewConfirm().
				Title("Add sample metadata contract?").
				Description("Add a sample ai.assets metadata contract for testing").
				Value(&addContract).
				Run(); err != nil {
				return err
			}

			if addContract {
				contract := createSampleMetadataContract(artifactType)
				contractJSON, err := json.Marshal(contract)
				if err != nil {
					return fmt.Errorf("failed to marshal metadata contract: %v", err)
				}
				manifest.Annotations[workflow.AnnotationMetadataContract] = string(contractJSON)
				fmt.Println("✓ Added sample metadata contract to manifest annotations")
			}
		}

		// Print manifest annotations
		fmt.Println("\n=== Manifest Annotations ===")
		for k, v := range manifest.Annotations {
			fmt.Printf("  %s: %s\n", k, v)
		}

		// Validate using the admission webhook logic
		fmt.Println("\n=== Validating Metadata Contract ===")
		
		// Create admission request
		request := &workflow.OCIManifestAdmissionRequest{
			Artifact:    manifest.Annotations["org.opencontainers.image.title"],
			Annotations: manifest.Annotations,
			Operation:   "push",
			Registry:    "test-registry",
		}

		// Create webhook and validate
		webhook := workflow.NewMetadataContractAdmissionWebhook()
		webhook.StrictMode = strictFlag
		
		response := webhook.Admit(request)

		// Print validation results
		if !response.Allowed {
			fmt.Println("\n✗ ADMISSION DENIED")
			fmt.Printf("  Reason: %s\n", response.Reason)
			fmt.Printf("  Message: %s\n", response.Message)
			
			if len(response.Errors) > 0 {
				fmt.Println("\n  Errors:")
				for _, err := range response.Errors {
					fmt.Printf("    - %s\n", err)
				}
			}
			
			if len(response.Warnings) > 0 {
				fmt.Println("\n  Warnings:")
				for _, warn := range response.Warnings {
					fmt.Printf("    - %s\n", warn)
				}
			}
			
			if response.Validation != nil {
				if len(response.Validation.MissingFields) > 0 {
					fmt.Println("\n  Missing required fields:")
					for _, field := range response.Validation.MissingFields {
						fmt.Printf("    - %s\n", field)
					}
				}
			}
			
			return fmt.Errorf("admission denied: %s", response.Message)
		}

		fmt.Println("\n✓ ADMISSION ALLOWED")
		fmt.Printf("  Message: %s\n", response.Message)
		
		if len(response.Warnings) > 0 {
			fmt.Println("\n  Warnings:")
			for _, warn := range response.Warnings {
				fmt.Printf("    - %s\n", warn)
			}
		}

		// If we have a validation result, print it
		if response.Validation != nil {
			fmt.Println("\n=== Validation Details ===")
			fmt.Printf("  Artifact Type: %s\n", response.Validation.ArtifactType)
			fmt.Printf("  Valid: %v\n", response.Validation.Valid)
			
			if len(response.Validation.RequiredFields) > 0 {
				fmt.Println("  Required Fields:")
				for _, field := range response.Validation.RequiredFields {
					fmt.Printf("    - %s\n", field)
				}
			}

			if len(response.Validation.MissingFields) > 0 {
				fmt.Println("  Missing Fields:")
				for _, field := range response.Validation.MissingFields {
					fmt.Printf("    - %s\n", field)
				}
			}
		}

		// Parse and display the metadata contract if present
		if contractJSON, ok := manifest.Annotations[workflow.AnnotationMetadataContract]; ok {
			fmt.Println("\n=== Metadata Contract ===")
			
			// Pretty print the contract
			var contract workflow.MetadataContract
			if err := json.Unmarshal([]byte(contractJSON), &contract); err == nil {
				prettyJSON, _ := json.MarshalIndent(contract, "  ", "  ")
				fmt.Printf("  %s\n", string(prettyJSON))
			} else {
				fmt.Printf("  (raw) %s\n", contractJSON)
			}

			// Display relationships
			fmt.Println("\n=== Relationship Mapping ===")
			
			switch artifactType {
			case workflow.ArtifactTypeModel:
				if contract.Assets.Model != nil && len(contract.Assets.Model.Relationships) > 0 {
					fmt.Println("  Model relationships:")
					for relType, refs := range contract.Assets.Model.Relationships {
						fmt.Printf("    %s: %v\n", relType, refs)
					}
				} else {
					fmt.Println("  No relationships defined")
				}
			case workflow.ArtifactTypeSkill:
				if contract.Assets.Skill != nil {
					fmt.Println("  Skill type:", contract.Assets.Skill.Type)
					if len(contract.Assets.Skill.Dependencies) > 0 {
						fmt.Println("  Dependencies:")
						for depType, refs := range contract.Assets.Skill.Dependencies {
							fmt.Printf("    %s: %v\n", depType, refs)
						}
					} else {
						fmt.Println("  No dependencies defined")
					}
				}
			case workflow.ArtifactTypePipeline:
				if contract.Assets.Pipeline != nil {
					fmt.Println("  Pipeline type:", contract.Assets.Pipeline.Type)
					fmt.Println("  Stages:", strings.Join(contract.Assets.Pipeline.Stages, ", "))
					if len(contract.Assets.Pipeline.Dependencies) > 0 {
						fmt.Println("  Dependencies:")
						for depType, refs := range contract.Assets.Pipeline.Dependencies {
							fmt.Printf("    %s: %v\n", depType, refs)
						}
					}
					if len(contract.Assets.Pipeline.Components) > 0 {
						fmt.Println("  Components:")
						for _, comp := range contract.Assets.Pipeline.Components {
							fmt.Printf("    - %s (%s): %s\n", comp.Name, comp.Type, comp.Reference)
						}
					}
				}
			}
		}

		fmt.Println("\n=== Enforcement Complete ===")
		fmt.Println("✓ Standardized Metadata Contract validated at manifest level")
		fmt.Println("✓ Relationships mapped without downloading binaries")
		fmt.Println("✓ Dependencies validated")
		
		return nil
	},
}

// runWebhookServer starts the admission webhook server
func runWebhookServer(port int) error {
	if port <= 0 {
		port = 8443 // Default port
	}
	
	proxy := workflow.NewRegistryAdmissionProxy(port)
	
	fmt.Printf("Starting Registry Admission Proxy on port %d...\n", port)
	fmt.Println("This server validates OCI manifests for the Standardized Metadata Contract")
	fmt.Println("Send POST requests to /validate with an OCIManifestAdmissionRequest")
	fmt.Println("\nPress Ctrl+C to stop the server\n")
	
	return proxy.Run()
}

// createSampleMetadataContract creates a sample metadata contract for testing
func createSampleMetadataContract(artifactType workflow.ArtifactType) *workflow.MetadataContract {
	contract := &workflow.MetadataContract{
		Assets: workflow.ContractAssetMetadata{},
	}
	
	switch artifactType {
	case workflow.ArtifactTypeModel:
		contract.Assets.Model = &workflow.ContractModelMetadata{
			Type:      "llm",
			Framework: "pytorch",
			Input:     "text",
			Output:    "text",
			Capabilities: []string{"chat", "completion", "embeddings"},
			Runtime:    "vllm",
			Accelerator: "nvidia-gpu",
			Relationships: map[string][]string{
				"skills": {"my-skill:v1", "my-rag:v1"},
			},
			Description: "Sample LLM model",
			Version:    "1.0.0",
			Author:     "model-cli",
			License:    "Apache-2.0",
		}
	case workflow.ArtifactTypeSkill:
		contract.Assets.Skill = &workflow.ContractSkillMetadata{
			Type:        "rag",
			PipelineRef: "my-pipeline:v1",
			Dependencies: map[string][]string{
				"models": {"model:sha256:abc123", "embedding-model:sha256:def456"},
				"pipelines": {"my-pipeline:v1"},
			},
			Runtime:    "python",
			Accelerator: "cpu",
			Description: "Sample RAG skill",
			Version:    "1.0.0",
			Author:     "model-cli",
		}
	case workflow.ArtifactTypePipeline:
		contract.Assets.Pipeline = &workflow.ContractPipelineMetadata{
			Type:    "inference",
			Stages:  []string{"preprocess", "inference", "postprocess"},
			Dependencies: map[string][]string{
				"models":  {"model:v1"},
				"skills":  {"skill:v1"},
				"datasets": {"dataset:v1"},
			},
			Components: []workflow.ContractPipelineComponent{
				{Name: "preprocessor", Type: "preprocessor", Reference: "preprocessor:v1"},
				{Name: "model", Type: "model", Reference: "model:v1"},
				{Name: "postprocessor", Type: "postprocessor", Reference: "postprocessor:v1"},
			},
			Description: "Sample inference pipeline",
			Version:    "1.0.0",
			Author:     "model-cli",
		}
	}
	
	return contract
}

func init() {
	rootCmd.AddCommand(enforceCmd)
	enforceCmd.Flags().String("manifest", "", "Path to the OCI manifest JSON file to validate")
	enforceCmd.Flags().String("artifact", "", "OCI artifact reference to validate")
	enforceCmd.Flags().Bool("strict", false, "Strict mode: reject on warnings")
	enforceCmd.Flags().Bool("webhook", false, "Run as a webhook server")
	enforceCmd.Flags().Int("port", 8443, "Port for webhook server")
	enforceCmd.Flags().Bool("simulate-push", false, "Simulate a push operation")
	enforceCmd.Flags().String("artifact-type", "", "Artifact type: model, skill, or pipeline")
}
