package cmd

import (
	"fmt"

	"github.com/lasomethingsomething/cli-prototype/config"
	"github.com/lasomethingsomething/cli-prototype/internal/workflow"
	"github.com/spf13/cobra"
)

var signCmd = &cobra.Command{
	Use:   "sign",
	Short: "Sign a model artifact with Sigstore or Notary v2 and record its provenance",
	Long: `Sign your OCI model artifact to establish provenance and trust.

This is Phase 1, Step 3 (Supply Chain Check): a separate step that runs after
'model-cli package' has produced the artifact. It signs the artifact and then
generates a SLSA provenance attestation for it. The attestation is written to
<artifact>.provenance.json (where 'model-cli verify' looks for it) and, when
the artifact lives in a registry, attached to it as an OCI referrer with the
artifact's digest as the subject.

This command supports the OpenSSF Model Signing Specification (OMS).
Supported signing tools (mutually exclusive):
` + workflow.SignerOptions().Bullets() + `

Note: SPIFFE/SPIRE is for identity, not signing.

Examples:
  model-cli sign
  model-cli sign --artifact my-model:latest --signer cosign
  model-cli sign --artifact my-model:latest --signer notary --key my-key`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Load()

		// Get flags
		useKeyFlag, _ := cmd.Flags().GetBool("use-key")
		registryFlag, _ := cmd.Flags().GetString("registry")

		// Interactive prompts
		var artifact string
		if err := askString(cmd, "artifact", &artifact, "Artifact to sign:", "The OCI artifact reference to sign (e.g., my-registry/my-model:latest)"); err != nil {
			return err
		}

		if err := askSelectToolIfEmpty(cmd, "signer", &cfg.Signer, "Select signing tool:", "Choose a signing provider (aligns with OSSF Model Signing Spec)", workflow.SignerOptions()); err != nil {
			return err
		}
		signer := cfg.Signer

		// A key reference implies a key; otherwise --use-key or a confirmation decides.
		useKey := cmd.Flags().Changed("key") || useKeyFlag
		if !useKey {
			if err := askConfirm(cmd, "use-key", &useKey, "Use a specific key?", "Do you want to specify a signing key? (Otherwise, default key will be used)"); err != nil {
				return err
			}
		}
		var keyRef string
		if useKey {
			if err := askString(cmd, "key", &keyRef, "Key reference:", "The key to use for signing (e.g., cosign-key.pub or notation-key)"); err != nil {
				return err
			}
		}

		if err := requireValues("artifact", artifact); err != nil {
			return err
		}

		// Get signing provider
		sp, err := workflow.GetSigningProvider(signer)
		if err != nil {
			return err
		}

		// The signer is known to be valid: remember it for next time.
		warnIfSaveFails(config.Save(cfg))

		// Check if tool is installed
		if !sp.IsInstalled() {
			return fmt.Errorf("%s not installed. Install with: %s", sp.Name(), sp.InstallInstructions())
		}

		fmt.Printf("\nSigning artifact '%s' with %s...\n", artifact, signer)

		if useKey {
			if err := sp.Sign(artifact, keyRef); err != nil {
				return err
			}
		} else {
			if err := sp.Sign(artifact, ""); err != nil {
				return err
			}
		}

		sigPath := sp.GetSignaturePath(artifact)
		fmt.Printf("\n✓ Signature created: %s\n", sigPath)
		fmt.Println("✓ Artifact is now signed and verifiable")

		// Provenance is generated here, after packaging and signing, never
		// during `model-cli package` (Phase 1, Step 3: Supply Chain Check).
		fmt.Println("\n=== Provenance ===")
		fmt.Println("→ Generating SLSA provenance attestation for the signed artifact...")

		destination := extractDestinationFromArtifact(artifact)
		var provider workflow.RegistryProvider
		if destination == "" {
			fmt.Println("  Note: the reference has no registry prefix; the attestation is kept locally only")
		} else {
			registry := registryFlag
			if registry == "" {
				registry = cfg.Registry
			}
			if registry == "" {
				registry = workflow.RegistryOptions().Recommended()
			}
			provider, err = workflow.GetRegistryProvider(registry)
			if err != nil {
				return err
			}
			if !provider.IsInstalled() {
				fmt.Printf("  Note: %s not installed (install with: %s); the attestation is kept locally only\n", provider.Name(), provider.InstallInstructions())
				provider = nil
			}
		}

		result, err := workflow.AttestSignedArtifact(provider, sp, workflow.SignedArtifact{
			Destination:   destination,
			Name:          extractArtifactNameFromArtifact(artifact),
			Signer:        signer,
			SignaturePath: sigPath,
		}, workflow.GetAttestationPath(artifact))
		if err != nil {
			return fmt.Errorf("failed to generate provenance attestation: %v", err)
		}

		predicate := result.Attestation.Statement.Predicate
		fmt.Println("  Attestation contains:")
		fmt.Printf("    - Build ID: %s\n", predicate.BuildID)
		fmt.Printf("    - Build Type: %s\n", predicate.BuildType)
		fmt.Printf("    - Builder: %s\n", predicate.Builder.ID)
		for algo, digest := range result.Attestation.StatementHeader.Subject[0].Digest {
			fmt.Printf("    - Subject digest: %s:%s\n", algo, digest)
		}

		fmt.Println("\nNext steps:")
		fmt.Println("  - Verify with: model-cli verify --artifact " + artifact)
		fmt.Println("  - Deploy with: model-cli deploy")
		fmt.Println("  - View signature: cosign triangle " + artifact)
		if !result.Attached {
			fmt.Println("  - Attestation kept locally at: " + result.Path)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(signCmd)
	signCmd.Flags().String("artifact", "", "OCI artifact reference to sign (e.g., my-registry/my-model:latest)")
	signCmd.Flags().String("signer", "", "Signing tool: "+workflow.SignerOptions().Summary())
	signCmd.Flags().String("key", "", "Key reference for signing")
	signCmd.Flags().Bool("use-key", false, "Use a specific key for signing")
	signCmd.Flags().String("registry", "", "Registry tool: "+workflow.RegistryOptions().Summary()+" (for attaching the provenance attestation; default: saved config, then "+workflow.RegistryOptions().Recommended()+")")
}
