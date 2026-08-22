package cmd

import (
	"fmt"

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

Examples:
  model-cli push
  model-cli push --artifact my-model:latest --registry oras --destination ghcr.io/my-org
  model-cli push --artifact my-model:latest --registry modelpack`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Load()

		// Get flags
		artifactFlag, _ := cmd.Flags().GetString("artifact")
		registryFlag, _ := cmd.Flags().GetString("registry")
		destinationFlag, _ := cmd.Flags().GetString("destination")
		manifestFlag, _ := cmd.Flags().GetString("manifest")

		// Interactive prompts
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

		fmt.Println("\nPhase 2 Complete: Enterprise OCI Registry")
		fmt.Println("Next: Run manifest-level validation with model-cli validate")

		return nil
	},
}

func init() {
	rootCmd.AddCommand(pushCmd)
	pushCmd.Flags().String("artifact", "", "OCI artifact reference to push (e.g., my-model:latest)")
	pushCmd.Flags().String("registry", "", "Registry tool: oras or modelpack")
	pushCmd.Flags().String("destination", "", "Destination registry (e.g., ghcr.io/my-org)")
	pushCmd.Flags().String("manifest", "", "Path to the OCI manifest.json produced by 'model-cli package', used to attach CNCF AI annotations")
}
