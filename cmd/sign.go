package cmd

import (
	"fmt"

	"github.com/lasomethingsomething/cli-prototype/config"
	"github.com/lasomethingsomething/cli-prototype/internal/workflow"
	"github.com/spf13/cobra"
)

var signCmd = &cobra.Command{
	Use:   "sign",
	Short: "Sign a model artifact with Sigstore or Notary v2",
	Long: `Sign your OCI model artifact to establish provenance and trust.

This command supports the OpenSSF Model Signing Specification (OMS) and
integrates with Sigstore (cosign) and Notary v2 (notation) for signing
AI/ML model artifacts.

Examples:
  model-cli sign
  model-cli sign --artifact my-model:latest --signer sigstore
  model-cli sign --artifact my-model:latest --signer notary --key my-key`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Load()

		// Get flags
		useKeyFlag, _ := cmd.Flags().GetBool("use-key")

		// Interactive prompts
		var artifact string
		if err := askString(cmd, "artifact", &artifact, "Artifact to sign:", "The OCI artifact reference to sign (e.g., my-registry/my-model:latest)"); err != nil {
			return err
		}

		if err := askSelectIfEmpty(cmd, "signer", &cfg.Signer, "Select signing tool:", "Choose a signing provider (aligns with OSSF Model Signing Spec)", []string{"sigstore", "notary"}); err != nil {
			return err
		}
		signer := cfg.Signer
		warnIfSaveFails(config.Save(cfg))

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

		// Get signing provider
		sp, err := workflow.GetSigningProvider(signer)
		if err != nil {
			return err
		}

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
		fmt.Println("\nNext steps:")
		fmt.Println("  - Verify with: model-cli verify")
		fmt.Println("  - Deploy with: model-cli deploy")
		fmt.Println("  - View signature: cosign triangle " + artifact)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(signCmd)
	signCmd.Flags().String("artifact", "", "OCI artifact reference to sign (e.g., my-registry/my-model:latest)")
	signCmd.Flags().String("signer", "", "Signing tool: sigstore or notary")
	signCmd.Flags().String("key", "", "Key reference for signing")
	signCmd.Flags().Bool("use-key", false, "Use a specific key for signing")
}
