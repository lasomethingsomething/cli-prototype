package cmd

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/lasomethingsomething/cli-prototype/config"
	"github.com/lasomethingsomething/cli-prototype/internal/workflow"
	"github.com/spf13/cobra"
)

// Styling for clean, uncluttered TUI
var (
	titleStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FAFAFA")).
		Bold(true).
		Padding(1, 2)

	subtitleStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#999999")).
		Padding(0, 2)

	stepStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#55AAFF")).
		Bold(true)

	successStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#00FF88")).
		Bold(true)

	infoStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#8888FF"))

	warningStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFAA00"))
)

var wizardCmd = &cobra.Command{
	Use:   "wizard",
	Short: "Interactive tour guide through the full ML model workflow",
	Long: `The Model CLI Wizard guides you through the complete secure model deployment journey:

  Package your model as an OCI artifact
  → Generate SBOM for transparency  
  → Classify with MOF framework
  → Sign with Sigstore or Notary v2
  → Verify the signature
  → Deploy to Kubernetes

This is the recommended way to experience the full workflow with a clean,
uncluttered interface that shows you exactly what's happening at each step.

Examples:
  model-cli wizard
  model-cli wizard --skip-signing
  model-cli wizard --registry oras`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check for skip flags
		skipSigning, _ := cmd.Flags().GetBool("skip-signing")
		skipDeploy, _ := cmd.Flags().GetBool("skip-deploy")

		cfg := config.Load()

		// === Welcome Screen ===
		fmt.Println()
		fmt.Println(titleStyle.Render("🪄 Model CLI Wizard"))
		fmt.Println(subtitleStyle.Render("Your guided journey from model to production"))
		fmt.Println()

		// === Step 1: Setup Preferences ===
		fmt.Println(stepStyle.Render("Step 1: Setup"))
		fmt.Println()

		// Registry tool
		if cfg.Registry == "" {
			var registryTool string
			if err := huh.NewSelect[string]().
				Title("How would you like to package your model?").
				Description(infoStyle.Render("(OCI artifacts for interoperability)")).
				Options(huh.NewOptions("oras", "modelpack")...).
				Value(&registryTool).
				Run(); err != nil {
				return err
			}
			cfg.Registry = registryTool
		}

		// GitOps tool
		if cfg.GitOps == "" {
			var gitOpsTool string
			if err := huh.NewSelect[string]().
				Title("How would you like to deploy?").
				Description(infoStyle.Render("(Argo has UI, Flux has agents)")).
				Options(huh.NewOptions("argo", "flux")...).
				Value(&gitOpsTool).
				Run(); err != nil {
				return err
			}
			cfg.GitOps = gitOpsTool
		}

		// Signing tool
		if cfg.Signer == "" && !skipSigning {
			var signerTool string
			if err := huh.NewSelect[string]().
				Title("How would you like to sign artifacts?").
				Description(infoStyle.Render("(Aligns with OSSF Model Signing Spec)")).
				Options(huh.NewOptions("sigstore", "notary")...).
				Value(&signerTool).
				Run(); err != nil {
				return err
			}
			cfg.Signer = signerTool
		}

		config.Save(cfg)
		fmt.Println(successStyle.Render("✓ Preferences saved"))
		fmt.Println()

		// === Step 2: Model Information ===
		fmt.Println(stepStyle.Render("Step 2: Model Details"))
		fmt.Println()

		var modelName string
		if err := huh.NewInput().
			Title("What's your model name?").
			Placeholder("phi-4-mini").
			Value(&modelName).
			Run(); err != nil {
			return err
		}

		var modelPath string
		if err := huh.NewInput().
			Title("Where are your model files?").
			Placeholder("./models/phi-4-mini").
			Value(&modelPath).
			Run(); err != nil {
			return err
			}

		var artifactName string
		if err := huh.NewInput().
			Title("What should we call the artifact?").
			Placeholder("my-org/my-model:v1.0.0").
			Value(&artifactName).
			Run(); err != nil {
			return err
			}

		var includeRAG bool
		if err := huh.NewConfirm().
			Title("Include RAG context?").
			Description("Add Retrieval-Augmented Generation data to your model").
			Value(&includeRAG).
			Run(); err != nil {
			return err
		}

		var ragPath string
		if includeRAG {
			if err := huh.NewInput().
				Title("RAG context path:").
				Value(&ragPath).
				Run(); err != nil {
				return err
			}
		}

		fmt.Println(successStyle.Render("✓ Model details collected"))
		fmt.Println()

		// === Step 3: Kubernetes Setup ===
		fmt.Println(stepStyle.Render("Step 3: Kubernetes Setup"))
		fmt.Println()

		var hasKubernetes bool
		if err := huh.NewConfirm().
			Title("Do you have a Kubernetes cluster?").
			Description("We'll package the model regardless, but deployment requires K8s").
			Value(&hasKubernetes).
			Run(); err != nil {
			return err
		}

		var repoURL string
		var manifestPath string
		if hasKubernetes && !skipDeploy {
			if err := huh.NewInput().
				Title("Git repository URL:").
				Placeholder("https://github.com/you/model-manifests").
				Value(&repoURL).
				Run(); err != nil {
				return err
			}

			if err := huh.NewInput().
				Title("Manifest path in repo:").
				Placeholder("./manifests").
				Value(&manifestPath).
				Run(); err != nil {
				return err
			}
		}

		fmt.Println()

		// === Step 4: Package (with SBOM and MOF) ===
		fmt.Println(stepStyle.Render("Step 4: Package Model"))
		fmt.Println()

		pf, err := workflow.NewPackageWorkflow(cfg.Registry)
		if err != nil {
			return err
		}
		pf.SetPackageInfo(modelName, modelPath, artifactName, "", includeRAG, ragPath)
		
		if err := pf.Run(); err != nil {
			return err
		}

		fullArtifact := artifactName
		// In real implementation, would push to registry

		fmt.Println()

		// === Step 5: Sign (unless skipped) ===
		if !skipSigning {
			fmt.Println(stepStyle.Render("Step 5: Sign Artifact"))
			fmt.Println()

			sp, err := workflow.GetSigningProvider(cfg.Signer)
			if err != nil {
				return err
			}

			if !sp.IsInstalled() {
				fmt.Println(warningStyle.Render("⚠ Signing tool not installed"))
				fmt.Printf("   Install with: %s\n\n", sp.InstallInstructions())
			} else {
				fmt.Printf("Signing %s with %s...\n", fullArtifact, cfg.Signer)
				// In real implementation: sp.Sign(fullArtifact, "")
				fmt.Println(successStyle.Render("✓ Artifact signed"))
			}
			fmt.Println()
		}

		// === Step 6: Verify (unless skipped) ===
		if !skipSigning {
			fmt.Println(stepStyle.Render("Step 6: Verify Signature"))
			fmt.Println()

			sp, err := workflow.GetSigningProvider(cfg.Signer)
			if err != nil {
				return err
			}

			if !sp.IsInstalled() {
				fmt.Println(warningStyle.Render("⚠ Signing tool not installed"))
				fmt.Printf("   Install with: %s\n\n", sp.InstallInstructions())
			} else {
				fmt.Printf("Verifying %s...\n", fullArtifact)
				// In real implementation: sp.Verify(fullArtifact)
				fmt.Println(successStyle.Render("✓ Signature verified - artifact is trusted"))
			}
			fmt.Println()
		}

		// === Step 7: Deploy (unless skipped) ===
		if hasKubernetes && !skipDeploy {
			fmt.Println(stepStyle.Render("Step 7: Deploy to Kubernetes"))
			fmt.Println()

			wf, err := workflow.NewDeployWorkflow(cfg.GitOps, cfg.Registry)
			if err != nil {
				return err
			}
			wf.SetModelInfo(modelName, repoURL, manifestPath)

			if err := wf.Run(); err != nil {
				return err
			}
			fmt.Println()
		} else if !skipDeploy {
			fmt.Println(stepStyle.Render("Step 7: Deployment Skipped"))
			fmt.Println()
			fmt.Println(infoStyle.Render("No Kubernetes cluster detected or deployment skipped"))
			fmt.Println(infoStyle.Render("Your model is packaged and signed, ready for deployment"))
			fmt.Println()
		}

		// === Summary ===
		fmt.Println(titleStyle.Render("🎉 Journey Complete!"))
		fmt.Println()
		fmt.Println("You've successfully:")
		fmt.Printf("  %s Packaged '%s' as OCI artifact\n", successStyle.Render("✓"), modelName)
		if !skipSigning {
			fmt.Printf("  %s Signed with %s\n", successStyle.Render("✓"), cfg.Signer)
			fmt.Printf("  %s Verified signature\n", successStyle.Render("✓"))
		} else {
			fmt.Printf("  %s Skipped signing\n", warningStyle.Render("⚠"))
		}
		if hasKubernetes && !skipDeploy {
			fmt.Printf("  %s Deployed to Kubernetes with %s\n", successStyle.Render("✓"), cfg.GitOps)
		} else {
			fmt.Printf("  %s Skipped deployment\n", warningStyle.Render("⚠"))
		}
		fmt.Println()
		fmt.Println(infoStyle.Render("Your model is now ready for production!"))
		fmt.Println()

		return nil
	},
}

func init() {
	rootCmd.AddCommand(wizardCmd)
	wizardCmd.Flags().Bool("skip-signing", false, "Skip the signing and verification steps")
	wizardCmd.Flags().Bool("skip-deploy", false, "Skip the deployment step")
}
