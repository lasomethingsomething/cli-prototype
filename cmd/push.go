package cmd

import (
	"fmt"
	"time"

	"github.com/charmbracelet/huh"
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
		artifactFlag, _ := cmd.Flags().GetString("artifact")
		registryFlag, _ := cmd.Flags().GetString("registry")
		destinationFlag, _ := cmd.Flags().GetString("destination")
		manifestFlag, _ := cmd.Flags().GetString("manifest")
		signFlag, _ := cmd.Flags().GetBool("sign")
		signerFlag, _ := cmd.Flags().GetString("signer")
		provenanceFlag, _ := cmd.Flags().GetBool("generate-provenance")
		modelPathFlag, _ := cmd.Flags().GetString("model-path")

		// Interactive prompts
		var modelPath string
		if modelPathFlag != "" {
			modelPath = modelPathFlag
		} else {
			// Try to derive from artifact name or use current directory
			modelPath = "."
		}

		var artifact string
		if artifactFlag != "" {
			artifact = artifactFlag
		} else {
			if err := huh.NewInput().
				Title("Artifact to push:").
				Description("The OCI artifact reference to push (e.g., my-model:latest)").
				Value(&artifact).
				Run(); err != nil {
				return err
			}
		}

		var registry string
		if registryFlag != "" {
			registry = registryFlag
		} else if cfg.Registry == "" {
			if err := huh.NewSelect[string]().
				Title("Select Registry tool:").
				Description("Choose how to push your artifact").
				Options(huh.NewOptions("oras", "modelpack")...).
				Value(&registry).
				Run(); err != nil {
				return err
			}
			cfg.Registry = registry
			config.Save(cfg)
		} else {
			registry = cfg.Registry
		}

		var destination string
		if destinationFlag != "" {
			destination = destinationFlag
		} else {
			if err := huh.NewInput().
				Title("Destination registry:").
				Description("Where to push (e.g., ghcr.io/my-org, docker.io/myuser)").
				Value(&destination).
				Run(); err != nil {
				return err
			}
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
		var annotations map[string]string
		if manifestFlag != "" {
			manifest, err := workflow.ReadManifest(manifestFlag)
			if err != nil {
				return err
			}
			annotations = manifest.Annotations
		}

		fmt.Printf("\nPushing '%s' to '%s' using %s...\n", artifact, destination, registry)

		fullArtifact := destination + "/" + artifact

		// Push the artifact
		if err := provider.Push(artifact, destination, annotations); err != nil {
			return err
		}

		fmt.Printf("\n✓ Pushed artifact: %s\n", fullArtifact)
		fmt.Println("✓ OCI layers stored in registry")
		if len(annotations) > 0 {
			fmt.Println("✓ Manifest with CNCF AI annotations available for validation")
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

func init() {
	rootCmd.AddCommand(pushCmd)
	pushCmd.Flags().String("artifact", "", "OCI artifact reference to push (e.g., my-model:latest)")
	pushCmd.Flags().String("registry", "", "Registry tool: oras or modelpack")
	pushCmd.Flags().String("destination", "", "Destination registry (e.g., ghcr.io/my-org)")
	pushCmd.Flags().String("manifest", "", "Path to the OCI manifest.json produced by 'model-cli package', used to attach CNCF AI annotations")
	pushCmd.Flags().Bool("sign", false, "Sign the artifact automatically after pushing")
	pushCmd.Flags().String("signer", "", "Signing tool: sigstore or notary (default: sigstore)")
	pushCmd.Flags().Bool("generate-provenance", true, "Generate SLSA provenance attestation (default: true)")
	pushCmd.Flags().String("model-path", "", "Path to the model directory (for provenance source info)")
}
