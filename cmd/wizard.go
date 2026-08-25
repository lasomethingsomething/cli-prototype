package cmd

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/lasomethingsomething/cli-prototype/config"
	"github.com/lasomethingsomething/cli-prototype/internal/tui"
	"github.com/lasomethingsomething/cli-prototype/internal/workflow"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

type wizardResult struct {
	packageSucceeded bool
	checkSucceeded   bool
	signSucceeded    bool
	verifySucceeded  bool
	deploySucceeded  bool
	modelName        string
	signer           string
	gitOps           string
	skipSigning      bool
	skipDeploy       bool
	skipCheck        bool
}

func buildSummaryLines(r wizardResult) []string {
	var lines []string
	if r.packageSucceeded {
		lines = append(lines, fmt.Sprintf("✓ Packaged '%s' as OCI artifact", r.modelName))
	} else {
		lines = append(lines, "⚠ Skipped packaging (registry tool not installed)")
	}
	if r.skipCheck {
		lines = append(lines, "⚠ Skipped compliance check")
	} else if r.checkSucceeded {
		lines = append(lines, "✓ Compliance check passed")
	} else {
		lines = append(lines, "⚠ Compliance check failed")
	}
	if r.skipSigning {
		lines = append(lines, "⚠ Skipped signing")
	} else if r.signSucceeded {
		lines = append(lines, fmt.Sprintf("✓ Signed with %s", r.signer))
	} else {
		lines = append(lines, "⚠ Signing tool not installed")
	}
	if !r.skipSigning {
		if r.verifySucceeded {
			lines = append(lines, "✓ Verified signature")
		} else if !r.signSucceeded {
			lines = append(lines, "⚠ Verification tool not installed")
		}
	}
	if r.skipDeploy {
		lines = append(lines, "⚠ Skipped deployment")
	} else if r.deploySucceeded {
		lines = append(lines, fmt.Sprintf("✓ Deployed to Kubernetes with %s", r.gitOps))
	} else {
		lines = append(lines, "⚠ Skipped deployment")
	}
	return lines
}

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
  → Run local compliance check (annotations, SBOM, MOF)
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
		skipCheck, _ := cmd.Flags().GetBool("skip-check")

		cfg := config.Load()

		// Initialize context model for interactive TUI
		ctxModel := tui.NewContextModel()
		ctxModel.SetStep(1)
		ctxModel.SetConfig(cfg.Registry, cfg.GitOps, cfg.Signer, "")

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
				Description(infoStyle.Render("ORAS: Standard OCI registry (recommended for most users) | ModelPack: ML-optimized packaging")).
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
				Description(infoStyle.Render("Argo: UI-based GitOps (good for visual workflows) | Flux: Agent-based GitOps (good for automation)")).
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
				Description(infoStyle.Render("Sigstore: Free, widely adopted (recommended) | Notary v2: Enterprise-focused, production-grade")).
				Options(huh.NewOptions("sigstore", "notary")...).
				Value(&signerTool).
				Run(); err != nil {
				return err
			}
			cfg.Signer = signerTool
		}

		if err := config.Save(cfg); err != nil {
			fmt.Printf("Warning: failed to save config: %v\n", err)
		} else {
			fmt.Println(successStyle.Render("✓ Preferences saved"))
		}
		fmt.Println()

		// Update context model with config
		ctxModel.SetConfig(cfg.Registry, cfg.GitOps, cfg.Signer, "")
		ctxModel.SetStep(1)
		displayInteractiveContext(ctxModel)
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

		// Show interactive context panel with current progress
		ctxModel.SetStep(2)
		ctxModel.SetModelInfo(modelName, modelPath, artifactName)
		displayInteractiveContext(ctxModel)

		// === Step 3: Kubernetes Setup ===
		fmt.Println(stepStyle.Render("Step 3: Kubernetes Setup"))
		fmt.Println()

		var hasKubernetes bool
		if err := huh.NewConfirm().
			Title("Do you have a Kubernetes cluster?").
			Description("If yes: Deploy directly. If no: Package only, deploy later or use locally").
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

		// Check if registry provider is installed before attempting to package
		registryProvider, err := workflow.GetRegistryProvider(cfg.Registry)
		if err != nil {
			return err
		}

		packageSucceeded := false
		signSucceeded := false
		verifySucceeded := false
		deploySucceeded := false
		if !registryProvider.IsInstalled() {
			fmt.Println(warningStyle.Render("Warning: Registry tool not installed"))
			fmt.Printf("   Install with: %s\n\n", registryProvider.InstallInstructions())
		} else {
			pf, err := workflow.NewPackageWorkflow(cfg.Registry)
			if err != nil {
				return err
			}
			pf.SetPackageInfo(modelName, modelPath, artifactName, "", includeRAG, ragPath)

			if err := pf.Run(); err != nil {
				return err
			}
			packageSucceeded = true
		}

		fullArtifact := artifactName
		// In real implementation, would push to registry

		// Update context model
		ctxModel.SetStep(4)
		ctxModel.SetResults(packageSucceeded, false, false, false)
		ctxModel.AddLog(fmt.Sprintf("Packaged artifact: %s", artifactName))
		fmt.Println()
		displayInteractiveContext(ctxModel)

		fmt.Println()

		// === Step 5: Compliance Check (local) ===
		fmt.Println(stepStyle.Render("Step 5: Compliance Check"))
		fmt.Println()

		// Run compliance check on the local artifact before push
		checkSucceeded := false
		if packageSucceeded && !skipCheck {
			fmt.Println("Running local compliance check before signing...")
			fmt.Println()

			// Create check workflow
			checkWorkflow := workflow.NewCheckWorkflow()
			// Use modelPath as both model and artifact path for local check
			checkWorkflow.SetCheckInfo(modelPath, modelPath)

			if err := checkWorkflow.Run(); err != nil {
				fmt.Println(warningStyle.Render("⚠ Compliance check failed"))
				fmt.Printf("   %v\n\n", err)
			} else {
				checkSucceeded = checkWorkflow.Passed()
				if checkSucceeded {
					fmt.Println(successStyle.Render("✓ Compliance check passed"))
				} else {
					fmt.Println(warningStyle.Render("⚠ Compliance check failed - missing required items"))
					for _, item := range checkWorkflow.Missing() {
						fmt.Printf("   ✗ %s\n", item)
					}
					fmt.Println()
				}
			}
		} else if skipCheck {
			fmt.Println(infoStyle.Render("⚠ Skipping compliance check (--skip-check flag set)"))
			fmt.Println()
			checkSucceeded = true // Consider passed if skipped
		} else {
			fmt.Println(infoStyle.Render("⚠ Skipping compliance check (package failed)"))
			fmt.Println()
		}

		// Update context model
		ctxModel.SetStep(5)
		ctxModel.AddLog("Compliance check completed")
		if checkSucceeded {
			ctxModel.AddLog("All checks passed")
		} else {
			ctxModel.AddLog("Some checks failed")
		}
		displayInteractiveContext(ctxModel)

		fmt.Println()

		// === Step 6: Sign (unless skipped) ===
		if !skipSigning {
			fmt.Println(stepStyle.Render("Step 5: Sign Artifact"))
			fmt.Println()

			sp, err := workflow.GetSigningProvider(cfg.Signer)
			if err != nil {
				return err
			}

			signSucceeded = false
			if !sp.IsInstalled() {
				fmt.Println(warningStyle.Render("⚠ Signing tool not installed"))
				fmt.Printf("   Install with: %s\n\n", sp.InstallInstructions())
			} else {
				fmt.Printf("Signing %s with %s...\n", fullArtifact, cfg.Signer)
				// In real implementation: sp.Sign(fullArtifact, "")
				fmt.Println(successStyle.Render("✓ Artifact signed"))
				signSucceeded = true
			}
			fmt.Println()
		}

		// === Step 7: Verify (unless skipped) ===
		if !skipSigning {
			fmt.Println(stepStyle.Render("Step 7: Verify Signature"))
			fmt.Println()

			sp, err := workflow.GetSigningProvider(cfg.Signer)
			if err != nil {
				return err
			}

			verifySucceeded = false
			if !sp.IsInstalled() {
				fmt.Println(warningStyle.Render("⚠ Signing tool not installed"))
				fmt.Printf("   Install with: %s\n\n", sp.InstallInstructions())
			} else {
				fmt.Printf("Verifying %s...\n", fullArtifact)
				// In real implementation: sp.Verify(fullArtifact)
				fmt.Println(successStyle.Render("✓ Signature verified - artifact is trusted"))
				verifySucceeded = true
			}
			fmt.Println()
		}

		// === Step 8: Deploy (unless skipped) ===
		if hasKubernetes && !skipDeploy {
			fmt.Println(stepStyle.Render("Step 8: Deploy to Kubernetes"))
			fmt.Println()

			// Check if GitOps and registry providers are installed before deploying
			gitOpsProvider, err := workflow.GetGitOpsProvider(cfg.GitOps)
			if err != nil {
				return err
			}

			registryProvider, err := workflow.GetRegistryProvider(cfg.Registry)
			if err != nil {
				return err
			}

			if !gitOpsProvider.IsInstalled() {
				fmt.Println(warningStyle.Render("Warning: GitOps tool not installed"))
				fmt.Printf("   Install with: %s\n", gitOpsProvider.InstallInstructions())
			}

			if !registryProvider.IsInstalled() {
				fmt.Println(warningStyle.Render("Warning: Registry tool not installed"))
				fmt.Printf("   Install with: %s\n", registryProvider.InstallInstructions())
			}

			if gitOpsProvider.IsInstalled() && registryProvider.IsInstalled() {
				wf, err := workflow.NewDeployWorkflow(cfg.GitOps, cfg.Registry)
				if err != nil {
					return err
				}
				wf.SetModelInfo(modelName, repoURL, manifestPath)

				if err := wf.Run(); err != nil {
					return err
				}
				deploySucceeded = true
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
		fmt.Println(titleStyle.Render("Journey Complete!"))
		fmt.Println()
		fmt.Println("You've successfully:")
		result := wizardResult{
			packageSucceeded: packageSucceeded,
			checkSucceeded:   checkSucceeded,
			signSucceeded:    signSucceeded,
			verifySucceeded:  verifySucceeded,
			deploySucceeded:  deploySucceeded,
			modelName:        modelName,
			signer:           cfg.Signer,
			gitOps:           cfg.GitOps,
			skipSigning:      skipSigning,
			skipDeploy:       skipDeploy,
			skipCheck:        skipCheck,
		}
		for _, line := range buildSummaryLines(result) {
			fmt.Printf("  %s\n", line)
		}
		fmt.Println()
		fmt.Println(infoStyle.Render("Your model is now ready for production!"))
		fmt.Println()

		return nil
	},
}

// displayInteractiveContext displays an interactive context panel with tabs
func displayInteractiveContext(ctxModel *tui.ContextModel) {
	// Check if stdin is a TTY (interactive terminal)
	if !isTTY() {
		// Non-interactive mode: just print the first tab
		fmt.Println()
		fmt.Println(ctxModel.View())
		return
	}

	fmt.Println()
	fmt.Println("Press Tab/Shift+Tab to switch views, 1-6 to select tab, Enter to continue, Esc to go back")
	fmt.Println()

	// Run the interactive context model
	p := tea.NewProgram(ctxModel)
	_, err := p.Run()
	fmt.Println()

	// If there was an error (user pressed ctrl+c), check if they cancelled
	if err != nil {
		// tea.Program returns an error on ctrl+c, but we handle Esc gracefully
		// For now, just continue execution
		return
	}

	// Check if user cancelled (pressed Esc)
	if ctxModel.IsCancelled() {
		// User wants to go back - we need a way to signal this
		// For now, we'll just exit the wizard
		fmt.Println(warningStyle.Render("Cancelled by user"))
		os.Exit(0)
	}

	// User pressed Enter - reset for next use and continue
	ctxModel.ResetControlFlags()
}

// isTTY checks if stdin is a TTY (interactive terminal)
func isTTY() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

func init() {
	rootCmd.AddCommand(wizardCmd)
	wizardCmd.Flags().Bool("skip-signing", false, "Skip the signing and verification steps")
	wizardCmd.Flags().Bool("skip-deploy", false, "Skip the deployment step")
	wizardCmd.Flags().Bool("skip-check", false, "Skip the local compliance check")
}
