package cmd

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/lasomethingsomething/cli-prototype/config"
	"github.com/lasomethingsomething/cli-prototype/internal/workflow"
	"github.com/spf13/cobra"
)

var verifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "Verify a model artifact signature",
	Long: `Verify the signature of an OCI model artifact to ensure trust and provenance.

This command validates signatures created by Sigstore (cosign) or Notary v2 (notation),
aligning with the OpenSSF Model Signing Specification (OMS).

Examples:
  model-cli verify
  model-cli verify --artifact my-model:latest
  model-cli verify --artifact my-model:latest --signer sigstore`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Load()

		// Interactive prompts
		var artifact string
		if err := huh.NewInput().
			Title("Artifact to verify:").
			Description("The OCI artifact reference to verify (e.g., my-registry/my-model:latest)").
			Value(&artifact).
			Run(); err != nil {
			return err
		}

		var signer string
		if cfg.Signer == "" {
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
			return fmt.Errorf("%s not installed. Install with: %s", signer, sp.InstallInstructions())
		}

		fmt.Printf("\nVerifying artifact '%s' with %s...\n", artifact, signer)
		
		if err := sp.Verify(artifact); err != nil {
			return err
		}

		fmt.Println("\n✓ Signature is VALID")
		fmt.Println("✓ Artifact provenance verified")
		fmt.Println("\nThis means:")
		fmt.Println("  - The artifact was signed by a trusted key")
		fmt.Println("  - The artifact has not been tampered with")
		fmt.Println("  - You can safely deploy this model")

		return nil
	},
}

func init() {
	rootCmd.AddCommand(verifyCmd)
}
