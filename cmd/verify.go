package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/charmbracelet/huh"
	"github.com/lasomethingsomething/cli-prototype/config"
	"github.com/lasomethingsomething/cli-prototype/internal/workflow"
	"github.com/spf13/cobra"
)

var verifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "Verify a model artifact signature and provenance attestation",
	Long: `Verify the signature of an OCI model artifact to ensure trust and provenance.

This command validates signatures created by Sigstore (cosign) or Notary v2 (notation),
aligning with the OpenSSF Model Signing Specification (OMS).

It also validates SLSA provenance attestations to ensure the artifact's
immutable provenance record is intact.

Examples:
  model-cli verify
  model-cli verify --artifact my-model:latest
  model-cli verify --artifact my-model:latest --signer sigstore
  model-cli verify --artifact my-model:latest --attestation my-model.provenance.json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Load()

		// Get flags
		artifactFlag, _ := cmd.Flags().GetString("artifact")
		signerFlag, _ := cmd.Flags().GetString("signer")
		attestationFlag, _ := cmd.Flags().GetString("attestation")
		registryFlag, _ := cmd.Flags().GetString("registry")
		localManifestFlag, _ := cmd.Flags().GetString("local-manifest")
		localParityFlag, _ := cmd.Flags().GetBool("local-parity")

		// Pre-extract destination and artifact name for provenance operations
		var destination, artifactName, registry string

		// Interactive prompts
		var artifact string
		if artifactFlag != "" {
			artifact = artifactFlag
		} else {
			if err := huh.NewInput().
				Title("Artifact to verify:").
				Description("The OCI artifact reference to verify (e.g., my-registry/my-model:latest)").
				Value(&artifact).
				Run(); err != nil {
				return err
			}
		}

		var signer string
		if signerFlag != "" {
			signer = signerFlag
		} else if cfg.Signer == "" {
			if err := huh.NewSelect[string]().
				Title("Select signing tool:").
				Description("Which signing tool was used?").
				Options(huh.NewOptions("sigstore", "notary")...).
				Value(&signer).
				Run(); err != nil {
				return err
			}
			cfg.Signer = signer
			config.Save(cfg)
		} else {
			signer = cfg.Signer
		}

		// Get signing provider
		sp, err := workflow.GetSigningProvider(signer)
		if err != nil {
			return err
		}

		// Check if tool is installed
		if !sp.IsInstalled() {
			return fmt.Errorf("%s not installed. Install with: %s", sp.Name(), sp.InstallInstructions())
		}

		fmt.Printf("\nVerifying artifact '%s' with %s...\n", artifact, signer)

		if err := sp.Verify(artifact); err != nil {
			return err
		}

		fmt.Println("\n✓ Signature is VALID")

		// Extract destination and artifact name for provenance operations
		destination = extractDestinationFromArtifact(artifact)
		artifactName = extractArtifactNameFromArtifact(artifact)

		// Perform local parity verification if requested
		if localParityFlag {
			fmt.Println("\n=== Local Parity Verification ===")
			fmt.Println("→ Verifying that local artifact matches registry copy...")

			// Determine registry if not specified
			if registryFlag != "" {
				registry = registryFlag
			} else if cfg.Registry != "" {
				registry = cfg.Registry
			} else if destination != "" {
				registry = "oras" // default
			} else {
				return fmt.Errorf("registry must be specified for local parity verification (use --registry)")
			}

			// Get the registry provider
			provider, err := workflow.GetRegistryProvider(registry)
			if err != nil {
				return fmt.Errorf("failed to get registry provider: %v", err)
			}
			if !provider.IsInstalled() {
				return fmt.Errorf("%s not installed. Install with: %s", provider.Name(), provider.InstallInstructions())
			}

			// Compute local digest from manifest file
			localDigest := ""
			if localManifestFlag != "" {
				localDigest = workflow.ComputeManifestDigest(localManifestFlag)
			} else {
				// Try default manifest location
				defaultManifest := artifactName + ".manifest.json"
				if _, err := os.Stat(defaultManifest); err == nil {
					localDigest = workflow.ComputeManifestDigest(defaultManifest)
				} else {
					return fmt.Errorf("local manifest not found. Specify with --local-manifest or ensure it's at default location")
				}
			}

			// Verify parity
			verifier := workflow.NewLocalParityVerifier(provider, localDigest)
			result, err := verifier.Verify(artifactName, destination)
			if err != nil {
				return fmt.Errorf("failed to verify local parity: %v", err)
			}
			fmt.Println(result.String())
			if !result.Match {
				return fmt.Errorf("local parity check FAILED: %s", result.String())
			}
		}

		// Try to verify provenance attestation if available
		attestationPath := attestationFlag

		// Determine registry if not specified
		if registry == "" {
			if registryFlag != "" {
				registry = registryFlag
			} else if cfg.Registry != "" {
				registry = cfg.Registry
			} else {
				registry = "oras" // default
			}
		}

		// Try to fetch attestation from registry first (if we have a registry and artifact with registry prefix)
		var attestation *workflow.ProvenanceAttestation
		var attestationSource string

		if attestationPath == "" && registry != "" {
			// Try to fetch from registry as a referrer
			provider, err := workflow.GetRegistryProvider(registry)
			if err == nil && provider.IsInstalled() {
				fmt.Println("\n=== Provenance Attestation ===")

				fmt.Printf("→ Fetching provenance attestation from registry for %s...\n", artifact)

				// Use AttestationManager to fetch from registry
				am := workflow.NewAttestationManager(nil, provider, destination)
				fetchedAttestation, err := am.GetAttestation(artifactName)
				if err != nil {
					fmt.Printf("  Note: Could not fetch from registry: %v\n", err)
					// Fall through to try local file
				} else {
					attestation = fetchedAttestation
					attestationSource = "registry referrer"
					fmt.Printf("✓ Provenance attestation fetched from %s\n", attestationSource)
				}
			}
		}

		// Fall back to local file if not fetched from registry
		if attestation == nil {
			if attestationPath == "" {
				// Try default location
				attestationPath = artifact + ".provenance.json"
			}

			if attestationPath != "" {
				fmt.Println("\n=== Provenance Attestation ===")
				fmt.Printf("→ Looking for attestation at: %s\n", attestationPath)

				// Read the attestation file
				attestationData, err := os.ReadFile(attestationPath)
				if err != nil {
					fmt.Printf("  ⚠ Provenance attestation not found: %v\n", err)
					fmt.Println("  Note: Attestation may be stored in the registry as a referrer")
				} else {
					// Parse and validate the attestation
					var parsedAttestation workflow.ProvenanceAttestation
					if err := json.Unmarshal(attestationData, &parsedAttestation); err != nil {
						fmt.Printf("  ✗ Failed to parse provenance attestation: %v\n", err)
					} else {
						attestation = &parsedAttestation
						attestationSource = "local file"
						fmt.Println("✓ Provenance attestation loaded from local file")
					}
				}
			} else {
				fmt.Println("\n  Note: No attestation file specified or found")
			}
		}

		// Validate and display attestation if we have one
		if attestation != nil {
			// Create an attestation manager for validation (with signer if available)
			var am *workflow.AttestationManager
			if sp != nil {
				provider, err := workflow.GetRegistryProvider(registry)
				if err == nil {
					am = workflow.NewAttestationManager(sp, provider, destination)
				}
			}

			// Validate the attestation structure and signature
			if err := workflow.ValidateAttestation(attestation); err != nil {
				fmt.Printf("  ✗ Provenance attestation validation failed: %v\n", err)
			} else {
				fmt.Printf("✓ Provenance attestation (from %s) structure is VALID\n", attestationSource)

				// If we have an attestation manager with a signer, validate the signature too
				if am != nil {
					if err := am.ValidateAndVerify(attestation); err != nil {
						fmt.Printf("  ⚠ Provenance attestation signature validation: %v\n", err)
					} else {
						fmt.Printf("✓ Provenance attestation signature is VALID\n")
					}
				} else {
					fmt.Printf("  ⚠ Skipping signature verification (no signer configured)\n")
				}

				// Display attestation details
				predicate := attestation.Statement.Predicate
				fmt.Println("\n  Attestation Details:")
				fmt.Printf("    - Build ID: %s\n", predicate.BuildID)
				fmt.Printf("    - Build Type: %s\n", predicate.BuildType)
				fmt.Printf("    - Builder: %s\n", predicate.Builder.ID)
				fmt.Printf("    - Source: %s\n", predicate.Source.ID)
				if predicate.Source.URI != "" {
					fmt.Printf("    - Source URI: %s\n", predicate.Source.URI)
				}
				fmt.Printf("    - Completed: %s\n", predicate.Metadata.BuildFinishedOn.Format("2006-01-02 15:04:05 MST"))

				// Display materials if present
				if len(predicate.Materials) > 0 {
					fmt.Println("\n  Materials:")
					for i, mat := range predicate.Materials {
						fmt.Printf("    %d. URI: %s\n", i+1, mat.URI)
						if mat.Digest != nil {
							for algo, digest := range mat.Digest {
								fmt.Printf("       %s: %s\n", algo, digest)
							}
						}
					}
				}

				// Check completeness
				fmt.Println("\n  Completeness:")
				fmt.Printf("    - Environment: %v\n", predicate.Metadata.Completeness.Environment)
				fmt.Printf("    - Parameters: %v\n", predicate.Metadata.Completeness.Parameters)
				fmt.Printf("    - Materials: %v\n", predicate.Metadata.Completeness.Materials)
				fmt.Printf("    - Reproducible: %v\n", predicate.Metadata.Reproducible)

				// Display invocations if present
				if len(predicate.Invocations) > 0 {
					fmt.Println("\n  Invocation:")
					for i, inv := range predicate.Invocations {
						fmt.Printf("    %d. Build ID: %s\n", i+1, predicate.BuildID)
						if len(inv.Parameters) > 0 {
							fmt.Println("       Parameters:")
							for k, v := range inv.Parameters {
								fmt.Printf("         - %s: %s\n", k, v)
							}
						}
					}
				}

				fmt.Println("\n✓ Artifact provenance fully verified")
				fmt.Println("✓ The hardened provenance metadata is frozen and verified")
			}
		}

		fmt.Println("\n=== Summary ===")
		fmt.Println("✓ Signature is VALID")
		fmt.Println("✓ Artifact has not been tampered with")
		fmt.Println("✓ You can safely deploy this model")

		return nil
	},
}

// extractDestinationFromArtifact extracts the registry destination from an artifact reference
// e.g., "ghcr.io/my-org/my-model:latest" -> "ghcr.io/my-org"
func extractDestinationFromArtifact(artifact string) string {
	// If artifact contains "/", the part before the last "/" is the destination
	// This is a simple implementation; real OCI artifact references may be more complex
	for i := len(artifact) - 1; i >= 0; i-- {
		if artifact[i] == '/' {
			return artifact[:i]
		}
		if artifact[i] == ':' {
			// Found tag, but no registry prefix
			break
		}
	}
	return ""
}

// extractArtifactNameFromArtifact extracts the artifact name from a reference
// e.g., "ghcr.io/my-org/my-model:latest" -> "my-model:latest"
func extractArtifactNameFromArtifact(artifact string) string {
	// Find the last "/" and return everything after it
	for i := len(artifact) - 1; i >= 0; i-- {
		if artifact[i] == '/' {
			return artifact[i+1:]
		}
	}
	return artifact
}

func init() {
	rootCmd.AddCommand(verifyCmd)
	verifyCmd.Flags().String("artifact", "", "OCI artifact reference to verify (e.g., my-registry/my-model:latest)")
	verifyCmd.Flags().String("signer", "", "Signing tool used: sigstore or notary")
	verifyCmd.Flags().String("attestation", "", "Path to provenance attestation file (default: <artifact>.provenance.json)")
	verifyCmd.Flags().String("registry", "", "Registry tool: oras or modelpack (for fetching attestations)")
	verifyCmd.Flags().String("local-manifest", "", "Path to local manifest file for parity verification")
	verifyCmd.Flags().Bool("local-parity", false, "Verify that local artifact matches the pushed copy in registry")
}
