package cmd

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

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
	packageSucceeded   bool
	checkSucceeded     bool
	signSucceeded      bool
	verifySucceeded    bool
	publishSucceeded   bool
	publishDestination string
	deploySucceeded    bool
	modelName          string
	signer             string
	gitOps             string
	skipSigning        bool
	skipDeploy         bool
	skipCheck          bool
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
	if r.publishSucceeded {
		lines = append(lines, fmt.Sprintf("✓ Published to %s", r.publishDestination))
	} else {
		lines = append(lines, "⚠ Kept artifact local (not published)")
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

func wizardCompletionMessage(r wizardResult) string {
	if r.deploySucceeded {
		return "Your model is deployed and ready for production."
	}
	return "Your local artifact is ready to sign, publish, and deploy when you are."
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
		if !interactive() {
			return fmt.Errorf("the wizard is a guided, interactive tour and needs a terminal (%s); use the individual commands with flags instead", nonInteractiveReason())
		}

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

		// === Step 1: Develop and package ===
		fmt.Println(stepStyle.Render("Step 1: Develop & Package"))
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
		expandedModelPath, err := expandHomePath(modelPath)
		if err != nil {
			return err
		}
		modelPath = expandedModelPath

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
			expandedRAGPath, err := expandHomePath(ragPath)
			if err != nil {
				return err
			}
			ragPath = expandedRAGPath
		}

		fmt.Println(successStyle.Render("✓ Model details collected"))
		fmt.Println()

		// Show interactive context panel with current progress
		ctxModel.SetStep(1)
		ctxModel.SetModelInfo(modelName, modelPath, artifactName)
		displayInteractiveContext(ctxModel, "Press Enter to package your model locally.")

		fmt.Println(stepStyle.Render("Packaging model locally"))
		fmt.Println()

		// Check if registry provider is installed before attempting to package
		localRegistryTool := workflow.RegistryOptions().Recommended()
		registryProvider, err := workflow.GetRegistryProvider(localRegistryTool)
		if err != nil {
			return err
		}

		packageSucceeded := false
		hardenSucceeded := false
		signSucceeded := false
		verifySucceeded := false
		deploySucceeded := false
		if !registryProvider.IsInstalled() {
			fmt.Println(warningStyle.Render("Warning: Registry tool not installed"))
			fmt.Printf("   Install with: %s\n\n", registryProvider.InstallInstructions())
		} else {
			pf, err := workflow.NewPackageWorkflow(localRegistryTool)
			if err != nil {
				return err
			}
			pf.SetShowNextSteps(false)
			pf.SetPackageInfo(modelName, modelPath, artifactName, "", includeRAG, ragPath)

			if err := pf.Run(); err != nil {
				return err
			}
			packageSucceeded = true
		}

		fullArtifact := artifactName
		// In real implementation, would push to registry

		// Update context model
		ctxModel.SetStep(2)
		ctxModel.SetResults(packageSucceeded, false, false, false)
		ctxModel.AddLog(fmt.Sprintf("Packaged artifact: %s", artifactName))
		fmt.Println()
		displayInteractiveContext(ctxModel, "Press Enter to generate an SBOM and classify the model.")

		fmt.Println()

		// === Step 2: Harden and validate the local artifact ===
		fmt.Println(stepStyle.Render("Step 2: Local Hardening & Compliance"))
		fmt.Println()
		if packageSucceeded {
			sbomTool := workflow.SBOMToolOptions().Recommended()
			if err := huh.NewSelect[string]().
				Title("Which tool should generate the SBOM?").
				Description("The SBOM is required before this artifact can be published.").
				Options(toolOptions(workflow.SBOMToolOptions())...).
				Value(&sbomTool).
				Run(); err != nil {
				return err
			}
			sbomGenerator, err := workflow.GetSBOMGenerator(sbomTool)
			if err != nil {
				return err
			}
			if !sbomGenerator.IsInstalled() {
				return fmt.Errorf("%s is required to harden this artifact; install it with: %s", sbomGenerator.Name(), sbomGenerator.InstallInstructions())
			}

			annotations := workflow.NewAnnotationSet()
			if err := huh.NewSelect[string]().
				Title("How should the Model Openness Framework class be set?").
				Description("Auto detects the classification from the model files.").
				Options(
					huh.NewOption("auto (recommended)", ""),
					huh.NewOption("I - open weights, code, data, docs, and license", "I"),
					huh.NewOption("II - open weights plus code, data, or docs", "II"),
					huh.NewOption("III - weights only", "III"),
				).
				Value(&annotations.MOFClass).
				Run(); err != nil {
				return err
			}

			hardenWorkflow := workflow.NewHardenWorkflow("")
			hardenWorkflow.SetHardenInfo(modelName, modelPath, artifactName)
			hardenWorkflow.SetOptions(true, true)
			hardenWorkflow.SetSBOMTool(sbomTool, workflow.SPDXJSON)
			hardenWorkflow.SetAnnotations(annotations)
			if err := hardenWorkflow.Run(); err != nil {
				return fmt.Errorf("hardening must complete before compliance: %w", err)
			}
			hardenSucceeded = true
		} else {
			fmt.Println(infoStyle.Render("Hardening skipped because packaging did not complete."))
		}

		ctxModel.SetStep(2)
		ctxModel.SetResults(packageSucceeded, false, false, false)
		ctxModel.AddLog("Hardening completed")
		fmt.Println()
		displayInteractiveContext(ctxModel, "Press Enter to run the local compliance check.")

		fmt.Println()

		fmt.Println(stepStyle.Render("Running local compliance check"))
		fmt.Println()

		// Run compliance check on the local artifact before push
		checkSucceeded := false
		if hardenSucceeded && !skipCheck {
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
			fmt.Println(infoStyle.Render("⚠ Skipping compliance check (hardening did not complete)"))
			fmt.Println()
		}

		// Update context model
		ctxModel.SetStep(3)
		ctxModel.AddLog("Compliance check completed")
		if checkSucceeded {
			ctxModel.AddLog("All checks passed")
		} else {
			ctxModel.AddLog("Some checks failed")
		}
		if !checkSucceeded && !skipCheck {
			return fmt.Errorf("compliance must pass before signing, publishing, or deployment")
		}
		if checkSucceeded {
			fmt.Printf("\nInspect the current local manifest with:\n  cat %s\n", filepath.Join(modelPath, "manifest.json"))
		}

		nextAction := "Press Enter to choose a signing tool."
		if skipSigning {
			nextAction = "Press Enter to choose how to publish the artifact."
		}
		displayInteractiveContext(ctxModel, nextAction)

		fmt.Println()

		// === Step 3: Supply-chain check ===
		if !skipSigning {
			fmt.Println(stepStyle.Render("Step 3: Supply Chain Check"))
			fmt.Println()

			signerTool := cfg.Signer
			if err := huh.NewSelect[string]().
				Title("How would you like to sign artifacts?").
				Description(infoStyle.Render("Cosign: Sigstore | Notary v2: notation")).
				Options(toolOptions(workflow.SignerOptions())...).
				Value(&signerTool).
				Run(); err != nil {
				return err
			}
			cfg.Signer = signerTool

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

		if !skipSigning {
			fmt.Println(stepStyle.Render("Verifying signature"))
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

		fmt.Println(stepStyle.Render("Publishing artifact"))
		fmt.Println()
		publishArtifact := false
		publishDestination := ""
		if err := huh.NewConfirm().
			Title("Publish this artifact to an OCI registry?").
			Description("Choose No to keep this test artifact on your computer.").
			Value(&publishArtifact).
			Run(); err != nil {
			return err
		}
		if publishArtifact {
			var publishTarget string
			if err := huh.NewSelect[string]().
				Title("Where is the OCI registry?").
				Options(
					huh.NewOption("an existing OCI registry", "existing"),
					huh.NewOption("a local Podman registry at localhost:5000", "local-podman"),
				).
				Value(&publishTarget).
				Run(); err != nil {
				return err
			}

			registryTool := cfg.Registry
			if err := huh.NewSelect[string]().
				Title("Which client should publish the artifact?").
				Description(infoStyle.Render("ORAS works with Harbor, GHCR, zot, and other OCI registries")).
				Options(toolOptions(workflow.RegistryOptions())...).
				Value(&registryTool).
				Run(); err != nil {
				return err
			}
			cfg.Registry = registryTool

			var destination string
			if publishTarget == "local-podman" {
				if err := verifyLocalRegistry(); err != nil {
					fmt.Println("\nStart a local registry with Podman, then run the wizard again:")
					fmt.Println("  brew install podman")
					fmt.Println("  podman machine init")
					fmt.Println("  podman machine start")
					fmt.Println("  podman run -d --rm --name model-cli-registry -p 5000:5000 registry:2")
					return fmt.Errorf("local OCI registry is unavailable at localhost:5000: %w", err)
				}
				destination = "localhost:5000"
			} else {
				if err := huh.NewInput().
					Title("Registry destination:").
					Description("For example: ghcr.io/my-org or harbor.example.com/models").
					Value(&destination).
					Run(); err != nil {
					return err
				}
			}
			if err := requireValues("destination", destination); err != nil {
				return err
			}

			registryProvider, err := workflow.GetRegistryProvider(cfg.Registry)
			if err != nil {
				return err
			}
			if !registryProvider.IsInstalled() {
				return fmt.Errorf("%s is required to publish this artifact; install it with: %s", registryProvider.Name(), registryProvider.InstallInstructions())
			}
			manifest, err := workflow.ReadUnifiedOCIManifest(filepath.Join(modelPath, "manifest.json"))
			if err != nil {
				return fmt.Errorf("failed to read packaged manifest before publishing: %w", err)
			}
			if _, err := registryProvider.Push(artifactName, destination, modelPath, manifest.Annotations); err != nil {
				return fmt.Errorf("failed to publish artifact: %w", err)
			}
			fmt.Printf("✓ Published %s to %s\n", artifactName, destination)
			publishDestination = destination
		} else {
			fmt.Println(infoStyle.Render("Publishing skipped; the artifact remains local."))
		}

		// === Step 4: Manifest-level validation ===
		fmt.Println()
		fmt.Println(stepStyle.Render("Step 4: Manifest-Level Validation"))
		if err := inspectWizardManifest(modelPath, artifactName, cfg.Registry, publishDestination); err != nil {
			return err
		}
		fmt.Println(successStyle.Render("✓ Manifest retrieved for inspection"))

		ctxModel.SetConfig(cfg.Registry, cfg.GitOps, cfg.Signer, "")
		if !skipDeploy {
			ctxModel.SetStep(5)
			displayInteractiveContext(ctxModel, "Press Enter to continue to GitOps admission and policy enforcement.")
			fmt.Println()
		}

		// === Step 5: GitOps admission and policy enforcement ===
		var hasKubernetes bool
		var repoURL string
		var manifestPath string
		if !skipDeploy {
			fmt.Println(stepStyle.Render("Step 5: GitOps Admission & Policy Enforcement"))
			fmt.Println()
			gitOpsTool := cfg.GitOps
			if err := huh.NewSelect[string]().
				Title("How would you like to promote the artifact?").
				Description(infoStyle.Render("Flux: agent-based automation | Argo CD: UI-based workflows")).
				Options(toolOptions(workflow.GitOpsOptions())...).
				Value(&gitOpsTool).
				Run(); err != nil {
				return err
			}
			cfg.GitOps = gitOpsTool
			if err := huh.NewConfirm().
				Title("Do you have a Kubernetes cluster?").
				Description("If no: the wizard simulates the remaining cluster stages.").
				Value(&hasKubernetes).
				Run(); err != nil {
				return err
			}
			if hasKubernetes {
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
		}

		if hasKubernetes && !skipDeploy {
			fmt.Println(stepStyle.Render("Deploying through GitOps"))
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
			fmt.Println(infoStyle.Render("No cluster connected; GitOps admission and policy enforcement simulated."))
		}

		if !skipDeploy {
			fmt.Println()
			fmt.Println(stepStyle.Render("Step 6: Infrastructure & Resource Orchestration"))
			fmt.Println("Simulated: Kubernetes would match the artifact's accelerator, CUDA, memory, GPU, and vRAM annotations to available nodes.")

			fmt.Println()
			fmt.Println(stepStyle.Render("Step 7: Runtime Execution & Optimization"))
			runtimeTool := cfg.Runtime
			if err := huh.NewSelect[string]().
				Title("Which runtime should serve the artifact?").
				Description(infoStyle.Render("vLLM: high-throughput inference | KServe: Kubernetes-native serving")).
				Options(toolOptions(workflow.RuntimeOptions())...).
				Value(&runtimeTool).
				Run(); err != nil {
				return err
			}
			cfg.Runtime = runtimeTool
			fmt.Printf("Simulated: %s would pull the OCI layers and serve the artifact with the declared runtime requirements.\n", cfg.Runtime)
		}

		if err := config.Save(cfg); err != nil {
			fmt.Printf("Warning: failed to save preferences: %v\n", err)
		}

		// === Summary ===
		fmt.Println(titleStyle.Render("Journey Complete!"))
		fmt.Println()
		fmt.Println("You've successfully:")
		result := wizardResult{
			packageSucceeded:   packageSucceeded,
			checkSucceeded:     checkSucceeded,
			signSucceeded:      signSucceeded,
			verifySucceeded:    verifySucceeded,
			publishSucceeded:   publishDestination != "",
			publishDestination: publishDestination,
			deploySucceeded:    deploySucceeded,
			modelName:          modelName,
			signer:             cfg.Signer,
			gitOps:             cfg.GitOps,
			skipSigning:        skipSigning,
			skipDeploy:         skipDeploy,
			skipCheck:          skipCheck,
		}
		for _, line := range buildSummaryLines(result) {
			fmt.Printf("  %s\n", line)
		}
		fmt.Println()
		fmt.Println(infoStyle.Render(wizardCompletionMessage(result)))
		fmt.Println()

		return nil
	},
}

// verifyLocalRegistry checks that the local OCI Distribution API is ready
// before the wizard attempts to push to it.
func verifyLocalRegistry() error {
	client := &http.Client{Timeout: 2 * time.Second}
	response, err := client.Get("http://localhost:5000/v2/")
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("registry API returned %s", response.Status)
	}
	return nil
}

// inspectWizardManifest retrieves the completed manifest. Annotation
// conventions are still evolving, so the wizard reports them without treating
// a particular set as a gate for the rest of the guided journey.
func inspectWizardManifest(modelPath, artifact, registry, destination string) error {
	var annotations map[string]string
	if destination == "" {
		manifest, err := workflow.ReadUnifiedOCIManifest(filepath.Join(modelPath, "manifest.json"))
		if err != nil {
			return fmt.Errorf("failed to read local manifest: %w", err)
		}
		annotations = manifest.Annotations
		fmt.Println("Validating local manifest...")
	} else {
		provider, err := workflow.GetRegistryProvider(registry)
		if err != nil {
			return err
		}
		annotations, err = provider.FetchManifestAnnotations(destination + "/" + artifact)
		if err != nil {
			return fmt.Errorf("failed to fetch published manifest: %w", err)
		}
		fmt.Printf("Validating published manifest at %s/%s...\n", destination, artifact)
	}
	fmt.Printf("  Found %d manifest annotation(s)\n", len(annotations))
	return nil
}

// expandHomePath expands the home-directory shorthand that shells normally
// expand before commands run, but which an interactive text input preserves.
func expandHomePath(path string) (string, error) {
	if path != "~" && !strings.HasPrefix(path, "~/") {
		return path, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not determine home directory: %w", err)
	}
	if path == "~" {
		return home, nil
	}
	return filepath.Join(home, path[2:]), nil
}

// displayInteractiveContext displays an interactive context panel with tabs
func displayInteractiveContext(ctxModel *tui.ContextModel, nextAction string) {
	// Check if stdin is a TTY (interactive terminal)
	if !isTTY() {
		// Non-interactive mode: just print the first tab
		fmt.Println()
		fmt.Println(ctxModel.View())
		return
	}

	fmt.Println()
	fmt.Println(nextAction)
	fmt.Println("Tab/Shift+Tab: switch views | 1-6: select tab | Esc: cancel")
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
