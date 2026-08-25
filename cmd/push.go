package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/lasomethingsomething/cli-prototype/config"
	"github.com/lasomethingsomething/cli-prototype/internal/workflow"
	"github.com/spf13/cobra"
)

var pushCmd = &cobra.Command{
	Use:   "push",
	Short: "Push an OCI artifact to a registry",
	Long: `Push your OCI model artifact to a container registry for distribution.

This command handles Phase 2 of the workflow: Enterprise OCI Registry.
The registry receives and stores OCI-aligned layers, enabling manifest-level
validation and relationship mapping without downloading large binaries.

The command also supports automatic signing at the point of push using
Sigstore (cosign) or Notary v2 (notation) for supply chain security, and
generates SLSA provenance attestations by default for immutable provenance tracking.

Examples:
  model-cli push
  model-cli push --artifact my-model:latest --registry oras --destination ghcr.io/my-org
  model-cli push --artifact my-model:latest --registry modelpack
  model-cli push --artifact my-model:latest --sign --signer sigstore
  model-cli push --artifact my-model:latest --generate-provenance --sign --signer sigstore`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Load()

		// Get flags
		targetFlag, _ := cmd.Flags().GetString("target")
		manifestFlag, _ := cmd.Flags().GetString("manifest")
		signFlag, _ := cmd.Flags().GetBool("sign")
		signerFlag, _ := cmd.Flags().GetString("signer")
		provenanceFlag, _ := cmd.Flags().GetBool("generate-provenance")
		modelPathFlag, _ := cmd.Flags().GetString("model-path")
		artifactTypeFlag, _ := cmd.Flags().GetString("artifact-type")
		manifestOutputFlag, _ := cmd.Flags().GetString("manifest-output")
		modelTypeFlag, _ := cmd.Flags().GetString("model-type")
		skillTypeFlag, _ := cmd.Flags().GetString("skill-type")
		pipelineTypeFlag, _ := cmd.Flags().GetString("pipeline-type")
		relationshipsFlag, _ := cmd.Flags().GetStringSlice("relationships")

		// Interactive prompts
		var modelPath string
		if modelPathFlag != "" {
			modelPath = modelPathFlag
		} else {
			// Try to derive from artifact name or use current directory
			modelPath = "."
		}

		// Handle --target flag which combines destination and artifact
		var artifact, destination string
		if targetFlag != "" {
			// Parse target to extract destination and artifact
			// Format: registry.example.com/my-model:v1
			// or: registry.example.com/my-org/my-model:v1
			lastSlash := -1
			for i := len(targetFlag) - 1; i >= 0; i-- {
				if targetFlag[i] == '/' {
					lastSlash = i
					break
				}
				if targetFlag[i] == ':' {
					// Found tag, but no registry prefix - this is just an artifact name
					break
				}
			}
			if lastSlash > 0 {
				destination = targetFlag[:lastSlash]
				artifact = targetFlag[lastSlash+1:]
			} else {
				artifact = targetFlag
			}
		} else {
			if err := askString(cmd, "artifact", &artifact, "Artifact to push:", "The OCI artifact reference to push (e.g., my-model:latest)"); err != nil {
				return err
			}
		}

		if err := askSelectIfEmpty(cmd, "registry", &cfg.Registry, "Select Registry tool:", "Choose how to push your artifact", []string{"oras", "modelpack"}); err != nil {
			return err
		}
		registry := cfg.Registry
		warnIfSaveFails(config.Save(cfg))

		if cmd.Flags().Changed("destination") || destination == "" {
			if err := askString(cmd, "destination", &destination, "Destination registry:", "Where to push (e.g., ghcr.io/my-org, docker.io/myuser)"); err != nil {
				return err
			}
		}

		// Determine artifact type
		var artifactType workflow.ArtifactType
		if artifactTypeFlag != "" {
			artifactType = workflow.ArtifactType(artifactTypeFlag)
		} else {
			// Try to determine from existing manifest or default to model
			artifactType = workflow.ArtifactTypeModel
		}

		// Parse relationships from flags
		relationships := parseRelationships(relationshipsFlag)

		// Create unified OCI manifest based on artifact type
		fmt.Println("\n=== Creating Unified OCI Manifest ===")
		unifiedManifest, err := createUnifiedOCIManifest(artifactType, artifact, modelTypeFlag, skillTypeFlag, pipelineTypeFlag, relationships)
		if err != nil {
			return fmt.Errorf("failed to create unified OCI manifest: %v", err)
		}

		// Validate the manifest
		if err := workflow.ValidateOCIManifest(unifiedManifest); err != nil {
			return fmt.Errorf("failed to validate unified OCI manifest: %v", err)
		}
		fmt.Println("✓ Unified OCI manifest created and validated")

		// Write manifest to file if requested (for debugging)
		if manifestOutputFlag != "" {
			if err := workflow.WriteUnifiedOCIManifest(unifiedManifest, manifestOutputFlag); err != nil {
				return fmt.Errorf("failed to write unified OCI manifest: %v", err)
			}
			fmt.Printf("✓ Unified OCI manifest written to: %s\n", manifestOutputFlag)
		}

		// Serialize manifest to JSON for pushing
		manifestJSON, err := json.MarshalIndent(unifiedManifest, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal unified OCI manifest: %v", err)
		}

		// Create a temporary directory for the manifest
		tmpDir := "tmp-oci-manifest"
		if err := os.MkdirAll(tmpDir, 0755); err != nil {
			return fmt.Errorf("failed to create temp directory: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		manifestFilePath := filepath.Join(tmpDir, "manifest.json")
		if err := os.WriteFile(manifestFilePath, manifestJSON, 0644); err != nil {
			return fmt.Errorf("failed to write manifest file: %v", err)
		}

		// Get registry provider
		provider, err := workflow.GetRegistryProvider(registry)
		if err != nil {
			return err
		}

		// Check if tool is installed
		if !provider.IsInstalled() {
			return fmt.Errorf("%s not installed. Install with: %s", provider.Name(), provider.InstallInstructions())
		}

		// If a manifest produced by `model-cli package` was given, read its
		// CNCF AI annotations so they're attached to the manifest on push.
		var existingAnnotations map[string]string
		if manifestFlag != "" {
			existingManifest, err := workflow.ReadUnifiedOCIManifest(manifestFlag)
			if err != nil {
				return err
			}
			existingAnnotations = existingManifest.Annotations
		}

		// Merge annotations from unified manifest with existing annotations
		// Unified manifest annotations take precedence
		mergedAnnotations := make(map[string]string)
		for k, v := range existingAnnotations {
			mergedAnnotations[k] = v
		}
		for k, v := range unifiedManifest.Annotations {
			mergedAnnotations[k] = v
		}

		fmt.Printf("\nPushing '%s' to '%s' using %s...\n", artifact, destination, registry)

		fullArtifact := destination + "/" + artifact

		// Push the artifact with unified OCI manifest annotations
		pushedDigest, err := provider.Push(artifact, destination, modelPath, mergedAnnotations)
		if err != nil {
			return err
		}

		fmt.Printf("\n✓ Pushed artifact: %s\n", fullArtifact)
		if pushedDigest != "" {
			fmt.Printf("  Digest: %s\n", pushedDigest)
		}
		fmt.Println("✓ OCI layers stored in registry")
		fmt.Println("✓ Unified OCI manifest with standardized metadata attached")
		if len(mergedAnnotations) > 0 {
			fmt.Printf("✓ %d manifest-level annotations available for validation\n", len(mergedAnnotations))
		}

		// Determine signer to use for provenance
		signerToUse := signerFlag
		if signerToUse == "" {
			signerToUse = cfg.Signer
		}
		if signerToUse == "" {
			signerToUse = "sigstore" // Default to sigstore
		}

		// Generate provenance attestation if requested (default: true)
		if provenanceFlag {
			fmt.Println("\n=== Provenance ===")
			fmt.Println("→ Generating SLSA provenance attestation with build info, source, materials, and timestamp...")

			// Create provenance generator with real metadata
			pg := workflow.NewProvenanceGenerator()
			pg.SetSourceInfo(modelPath, destination)
			pg.SetArtifactInfo(fullArtifact, nil) // Digest will be added by registry after push

			// Add destination as a material
			if destination != "" {
				pg.AddMaterial("oci://"+destination, nil)
			}

			pg.SetRecipeInfo(
				"https://model-cli.dev/recipe/push/v1",
				fmt.Sprintf("registry:%s", registry),
				"push",
			)

			// Generate and push attestation as a referrer to the registry
			// This makes it immutable and attached to the artifact per OSSF Model Signing Spec
			if err := pushProvenanceAttestation(provider, registry, destination, artifact, pg, signFlag, signerToUse, modelPath); err != nil {
				fmt.Printf("  ⚠ Failed to push provenance attestation to registry: %v\n", err)
				fmt.Println("  Falling back to local file generation...")

				// Write attestation to a local file as fallback
				attestationPath := artifact + ".provenance.json"
				if _, err := pg.WriteToFile(attestationPath); err != nil {
					fmt.Printf("  Warning: Failed to generate local provenance attestation: %v\n", err)
				} else {
					fmt.Printf("✓ Provenance attestation generated locally: %s\n", attestationPath)
				}
			} else {
				fmt.Println("✓ Provenance attestation frozen and attached to artifact as immutable referrer")
			}
		}

		// Sign the artifact if requested (at point of creation/during push)
		if signFlag {
			fmt.Println("\n=== Signing ===")
			fmt.Printf("→ Signing artifact '%s'...\n", fullArtifact)

			// Get signing provider
			sp, err := workflow.GetSigningProvider(signerToUse)
			if err != nil {
				return fmt.Errorf("failed to get signing provider: %v", err)
			}

			// Check if tool is installed
			if !sp.IsInstalled() {
				return fmt.Errorf("%s not installed. Install with: %s", sp.Name(), sp.InstallInstructions())
			}

			// Sign the artifact
			if err := sp.Sign(fullArtifact, ""); err != nil {
				return fmt.Errorf("failed to sign artifact: %v", err)
			}

			sigPath := sp.GetSignaturePath(fullArtifact)
			fmt.Printf("✓ Signed artifact: %s\n", fullArtifact)
			fmt.Printf("  Signature: %s\n", sigPath)
		}

		fmt.Println("\nPhase 2 Complete: Enterprise OCI Registry")

		if signFlag {
			fmt.Println("Next: Run manifest-level validation with model-cli validate")
		} else {
			fmt.Println("Next: Run manifest-level validation with model-cli validate")
			fmt.Println("  - Or sign with: model-cli sign --artifact " + fullArtifact)
		}

		return nil
	},
}

// pushProvenanceAttestation generates and pushes a provenance attestation as a registry referrer
func pushProvenanceAttestation(provider workflow.RegistryProvider, registryTool, destination, artifact string, pg *workflow.ProvenanceGenerator, sign bool, signerName string, modelPath string) error {
	// Determine the signer to use
	signerToUse := signerName
	if signerToUse == "" {
		signerToUse = "sigstore" // Default to sigstore
	}

	// Get signing provider (may be nil if not installed, but we'll try anyway)
	sp, err := workflow.GetSigningProvider(signerToUse)
	if err != nil {
		// Log but don't fail - we can still push unsigned attestation
		fmt.Printf("  Note: Signing provider not available: %v\n", err)
		sp = nil
	}

	// Update provenance generator with additional metadata from the push operation
	// Set the invocation info for traceability
	pg.SetInvocationInfo("", map[string]interface{}{
		"registry":    registryTool,
		"destination": destination,
		"modelPath":   modelPath,
		"pushedAt":    time.Now().UTC().Format(time.RFC3339),
	})

	// Create attestation manager
	am := workflow.NewAttestationManager(sp, provider, destination)

	// Push the attestation as a referrer
	if err := am.AttestAndPush(artifact, pg, sign && sp != nil); err != nil {
		return err
	}

	return nil
}

// parseRelationships parses relationship flags in format "type=ref" into a map
func parseRelationships(relationships []string) map[string][]string {
	result := make(map[string][]string)
	for _, rel := range relationships {
		// Split on the first "=" to handle refs that might contain "="
		idx := -1
		for i, c := range rel {
			if c == '=' {
				idx = i
				break
			}
		}
		if idx == -1 {
			// No "=" found, skip or treat as unknown type
			continue
		}
		typ := rel[:idx]
		ref := rel[idx+1:]
		if ref != "" {
			result[typ] = append(result[typ], ref)
		}
	}
	return result
}

// createUnifiedOCIManifest creates a unified OCI manifest based on artifact type and configuration
func createUnifiedOCIManifest(artifactType workflow.ArtifactType, artifactName, modelType, skillType, pipelineType string, relationships map[string][]string) (*workflow.UnifiedOCIManifest, error) {
	// Create the manifest with appropriate layers
	// For now, we create an empty layer list - in production this would contain
	// the actual OCI layers (model weights, config files, etc.)
	layers := []workflow.OCILayer{}

	manifest := workflow.NewUnifiedOCIManifest(artifactType, artifactName, layers)

	// Customize the config based on artifact type and provided flags
	switch artifactType {
	case workflow.ArtifactTypeModel:
		config := workflow.AIModelConfig{
			Architecture:  "amd64",
			OS:            "linux",
			ModelType:     modelType,
			ModelFormat:   "pytorch",                      // Default, can be overridden
			InputFormat:   "text",                         // Default
			OutputFormat:  "text",                         // Default
			Capabilities:  []string{"chat", "completion"}, // Default capabilities
			Runtime:       "vllm",
			Accelerator:   "nvidia-gpu",
			Relationships: relationships,
			Description:   fmt.Sprintf("Model artifact: %s", artifactName),
			Version:       "1.0.0",
			Author:        "model-cli",
			License:       "Apache-2.0",
		}
		// If modelType was provided, override the default
		if modelType != "" {
			config.ModelType = modelType
		}
		manifest.SetModelConfig(config)

	case workflow.ArtifactTypeSkill:
		config := workflow.AISkillConfig{
			Architecture: "amd64",
			OS:           "linux",
			SkillType:    skillType,
			Dependencies: relationships,
			Runtime:      "python",
			Accelerator:  "cpu",
			Description:  fmt.Sprintf("Skill artifact: %s", artifactName),
			Version:      "1.0.0",
			Author:       "model-cli",
		}
		if skillType != "" {
			config.SkillType = skillType
		}
		manifest.SetSkillConfig(config)

	case workflow.ArtifactTypePipeline:
		config := workflow.AIPipelineConfig{
			Architecture: "amd64",
			OS:           "linux",
			PipelineType: pipelineType,
			Dependencies: relationships,
			Description:  fmt.Sprintf("Pipeline artifact: %s", artifactName),
			Version:      "1.0.0",
			Author:       "model-cli",
		}
		if pipelineType != "" {
			config.PipelineType = pipelineType
		}
		manifest.SetPipelineConfig(config)
	}

	// Add standardized OCI annotations
	manifest.Annotations["org.opencontainers.image.title"] = artifactName
	manifest.Annotations["org.opencontainers.image.description"] = fmt.Sprintf("%s artifact pushed by model-cli", artifactType)
	manifest.Annotations["org.opencontainers.image.version"] = "1.0.0"

	// Add Trust Profile annotations for GitOps admission (Story #63)
	// These annotations allow GitOps tools (Argo CD, Flux) + policy engines
	// (Sigstore Policy Controller, OPA/Gatekeeper) to evaluate artifact trust
	manifest.Annotations[workflow.AnnotationSigningFramework] = "sigstore-cosign"
	manifest.Annotations[workflow.AnnotationSBOMFormat] = "spdx-json"
	manifest.Annotations[workflow.AnnotationProvenanceType] = "slsa-v1.0"

	// Add MOF classification for compliance checking
	manifest.Annotations[workflow.AnnotationMOFClass] = "I" // Default to most open
	manifest.Annotations[workflow.AnnotationMOFVersion] = "1.0"
	manifest.Annotations[workflow.AnnotationMOFComponents] = "weights,training-data,code"

	// Add Infrastructure Requirement annotations for GitOps admission (Story #64)
	// These annotations allow policy engines (OPA/Gatekeeper, Kyverno) to verify
	// that the artifact's requirements match the destination environment
	// Values are extracted from the AI config
	if manifest.AIConfig != nil {
		switch config := manifest.AIConfig.(type) {
		case workflow.AIModelConfig:
			if config.Runtime != "" {
				manifest.Annotations[workflow.AnnotationRuntime] = config.Runtime
			}
			if config.Accelerator != "" {
				manifest.Annotations[workflow.AnnotationAccelerator] = config.Accelerator
			}
			if config.CUDAMin != "" {
				manifest.Annotations[workflow.AnnotationCUDAVersionMin] = config.CUDAMin
			}
			if config.MemoryMin != "" {
				manifest.Annotations[workflow.AnnotationMemoryMin] = config.MemoryMin
			}
		case workflow.AISkillConfig:
			if config.Runtime != "" {
				manifest.Annotations[workflow.AnnotationRuntime] = config.Runtime
			}
			if config.Accelerator != "" {
				manifest.Annotations[workflow.AnnotationAccelerator] = config.Accelerator
			}
			if config.CUDAMin != "" {
				manifest.Annotations[workflow.AnnotationCUDAVersionMin] = config.CUDAMin
			}
			if config.MemoryMin != "" {
				manifest.Annotations[workflow.AnnotationMemoryMin] = config.MemoryMin
			}
		case workflow.AIPipelineConfig:
			// Pipeline runtime requirements - set defaults that can be overridden
			// In production, these would come from pipeline component analysis
			if config.Runtime != "" {
				manifest.Annotations[workflow.AnnotationRuntime] = config.Runtime
			} else {
				manifest.Annotations[workflow.AnnotationRuntime] = "vllm"
			}
			if config.Accelerator != "" {
				manifest.Annotations[workflow.AnnotationAccelerator] = config.Accelerator
			} else {
				manifest.Annotations[workflow.AnnotationAccelerator] = "nvidia-gpu"
			}
			if config.CUDAMin != "" {
				manifest.Annotations[workflow.AnnotationCUDAVersionMin] = config.CUDAMin
			}
			if config.MemoryMin != "" {
				manifest.Annotations[workflow.AnnotationMemoryMin] = config.MemoryMin
			}
		}
	}

	// Ensure at least default values for infrastructure requirements
	if _, ok := manifest.Annotations[workflow.AnnotationRuntime]; !ok {
		manifest.Annotations[workflow.AnnotationRuntime] = "vllm"
	}
	if _, ok := manifest.Annotations[workflow.AnnotationAccelerator]; !ok {
		manifest.Annotations[workflow.AnnotationAccelerator] = "nvidia-gpu"
	}
	if _, ok := manifest.Annotations[workflow.AnnotationCUDAVersionMin]; !ok {
		manifest.Annotations[workflow.AnnotationCUDAVersionMin] = "12.1"
	}
	if _, ok := manifest.Annotations[workflow.AnnotationMemoryMin]; !ok {
		manifest.Annotations[workflow.AnnotationMemoryMin] = "24GiB"
	}

	return manifest, nil
}

func init() {
	rootCmd.AddCommand(pushCmd)
	pushCmd.Flags().String("artifact", "", "OCI artifact reference to push (e.g., my-model:latest)")
	pushCmd.Flags().String("target", "", "Full target reference (e.g., registry.example.com/my-model:v1) - combines destination and artifact")
	pushCmd.Flags().String("registry", "", "Registry tool: oras or modelpack")
	pushCmd.Flags().String("destination", "", "Destination registry (e.g., ghcr.io/my-org)")
	pushCmd.Flags().String("manifest", "", "Path to the OCI manifest.json produced by 'model-cli package', used to attach CNCF AI annotations")
	pushCmd.Flags().Bool("sign", false, "Sign the artifact automatically after pushing")
	pushCmd.Flags().String("signer", "", "Signing tool: sigstore or notary (default: sigstore)")
	pushCmd.Flags().Bool("generate-provenance", true, "Generate SLSA provenance attestation (default: true)")
	pushCmd.Flags().String("model-path", "", "Path to the model directory (for provenance source info)")
	pushCmd.Flags().String("artifact-type", "", "Type of AI artifact: model, skill, or pipeline")
	pushCmd.Flags().String("manifest-output", "", "Path to write the unified OCI manifest (optional, for debugging)")
	pushCmd.Flags().String("model-type", "", "Model type for AI-specific config (e.g., text-generation, embedding)")
	pushCmd.Flags().String("skill-type", "", "Skill type for AI-specific config (e.g., rag, classification)")
	pushCmd.Flags().String("pipeline-type", "", "Pipeline type for AI-specific config (e.g., inference, training)")
	pushCmd.Flags().StringSlice("relationships", []string{}, "Relationships to other assets (format: type=ref, e.g., skill=my-skill:v1)")
}
