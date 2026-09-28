package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
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

// parseMemory parses a Kubernetes memory string to int64 Ki (base 2)
// Handles k8s suffixes: Ki, Mi, Gi, Ti and IEC suffixes: KiB, MiB, GiB, TiB
// Note: Both formats use base-2, and numerically: 1 MiB = 1 Mi = 1024 Ki, 1 GiB = 1 Gi = 1048576 Ki
func parseMemory(s string) int64 {
	if s == "" {
		return 0
	}
	// Normalize IEC suffixes (KiB, MiB, GiB, TiB) to k8s suffixes (Ki, Mi, Gi, Ti)
	// This allows both annotation format (256MiB) and k8s format (256Mi) to work
	if strings.HasSuffix(s, "GiB") {
		s = strings.TrimSuffix(s, "GiB") + "Gi"
	} else if strings.HasSuffix(s, "MiB") {
		s = strings.TrimSuffix(s, "MiB") + "Mi"
	} else if strings.HasSuffix(s, "TiB") {
		s = strings.TrimSuffix(s, "TiB") + "Ti"
	} else if strings.HasSuffix(s, "KiB") {
		s = strings.TrimSuffix(s, "KiB") + "Ki"
	}
	// Handle k8s-style suffixes: Ki, Mi, Gi, Ti (base 2)
	if strings.HasSuffix(s, "Gi") {
		val, _ := strconv.ParseInt(strings.TrimSuffix(s, "Gi"), 10, 64)
		return val * 1024 * 1024
	} else if strings.HasSuffix(s, "Mi") {
		val, _ := strconv.ParseInt(strings.TrimSuffix(s, "Mi"), 10, 64)
		return val * 1024
	} else if strings.HasSuffix(s, "Ti") {
		val, _ := strconv.ParseInt(strings.TrimSuffix(s, "Ti"), 10, 64)
		return val * 1024 * 1024 * 1024
	} else if strings.HasSuffix(s, "Ki") {
		val, _ := strconv.ParseInt(strings.TrimSuffix(s, "Ki"), 10, 64)
		return val
	}
	// Plain number (assume Ki)
	val, _ := strconv.ParseInt(s, 10, 64)
	return val
}

type wizardResult struct {
	packageSucceeded   bool
	checkSucceeded     bool
	signSucceeded      bool
	signSimulated      bool // True if signing was simulated
	verifySucceeded    bool
	verifySimulated    bool // True if verification was simulated
	publishSucceeded   bool
	publishDestination string
	deploySucceeded    bool
	deployVerified     bool // True if InferenceService reached Ready
	predictionVerified bool // True if a prediction was successfully served
	deployNoOp         bool // True if deploy was a no-op (already deployed)
	deployNewCommit    bool // True if deploy created a new commit
	modelName          string
	artifactName       string
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
		if r.signSimulated {
			lines = append(lines, fmt.Sprintf("⚠ Signed with %s (simulated)", r.signer))
		} else {
			lines = append(lines, fmt.Sprintf("✓ Signed with %s", r.signer))
		}
	} else {
		lines = append(lines, "⚠ Signing tool not installed")
	}
	if !r.skipSigning {
		if r.verifySucceeded {
			if r.verifySimulated {
				lines = append(lines, "⚠ Verified signature (simulated)")
			} else {
				lines = append(lines, "✓ Verified signature")
			}
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
	} else if r.deployNoOp && !r.deployNewCommit {
		if r.predictionVerified {
			lines = append(lines, fmt.Sprintf("⚠ Already deployed with %s (no changes, prediction verified)", r.gitOps))
		} else {
			lines = append(lines, fmt.Sprintf("⚠ Already deployed with %s (no changes, InferenceService still Ready)", r.gitOps))
		}
	} else if r.deployVerified {
		if r.predictionVerified {
			if r.deployNewCommit {
				lines = append(lines, fmt.Sprintf("✓ Deployed with %s (prediction verified)", r.gitOps))
			} else {
				lines = append(lines, fmt.Sprintf("✓ Deployed to Kubernetes with %s (InferenceService Ready, prediction verified)", r.gitOps))
				lines = append(lines, "✓ Verified: model served a prediction")
			}
		} else {
			if r.deployNewCommit {
				lines = append(lines, fmt.Sprintf("✓ Deployed with %s", r.gitOps))
			} else {
				lines = append(lines, fmt.Sprintf("✓ Deployed to Kubernetes with %s (InferenceService verified Ready)", r.gitOps))
			}
		}
	} else if r.deploySucceeded {
		lines = append(lines, fmt.Sprintf("✓ Deployed to Kubernetes with %s (GitOps commit successful, Flux reconciliation initiated)", r.gitOps))
	} else {
		lines = append(lines, "⚠ Deployment not completed")
	}
	return lines
}

func wizardCompletionMessage(r wizardResult) string {
	if r.deployVerified && r.predictionVerified {
		return "Your model is deployed, InferenceService is Ready, and serving was verified with a prediction."
	}
	if r.deployVerified {
		return "Your model is deployed and the InferenceService reached Ready state."
	}
	if r.deploySucceeded {
		return "Your model deployment was initiated via GitOps. Check Flux reconciliation and InferenceService status."
	}
	if r.publishSucceeded {
		return "Your artifact is published and ready for signing or deployment."
	}
	return "Your local artifact is ready to sign, publish, and deploy when you are."
}

func wizardManifestInstructions(modelPath string, r wizardResult) []string {
	instructions := []string{fmt.Sprintf("Local manifest: cat %s", filepath.Join(modelPath, "manifest.json"))}
	if r.publishSucceeded {
		instructions = append(instructions, fmt.Sprintf("Published manifest: oras manifest fetch %s/%s", r.publishDestination, r.artifactName))
	}
	return instructions
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

// getServingTopologyOptions returns topology options with the recommended tag
// derived from the model format, reusing the same detection used for runtime.
// The recommended option is moved to the first position so Enter-through selects it.
func getServingTopologyOptions(modelFormat workflow.ModelFormat) workflow.ToolOptions {
	opts := workflow.ServingTopologyOptions()
	for i := range opts {
		opts[i].Recommended = false
	}

	switch modelFormat {
	case workflow.ModelFormatSklearn, workflow.ModelFormatONNX, workflow.ModelFormatTensorFlow:
		for i := range opts {
			if opts[i].Name == "kserve" {
				opts[i].Recommended = true
				break
			}
		}
	case workflow.ModelFormatPyTorch, workflow.ModelFormatHuggingFace:
		for i := range opts {
			if opts[i].Name == "kserve-vllm" {
				opts[i].Recommended = true
				break
			}
		}
	default:
		for i := range opts {
			if opts[i].Name == "kserve" {
				opts[i].Recommended = true
				break
			}
		}
	}

	// Reorder: move recommended option to first position
	// Preserve relative order of remaining options
	var recommendedIdx int = -1
	for i, o := range opts {
		if o.Recommended {
			recommendedIdx = i
			break
		}
	}
	if recommendedIdx > 0 {
		// Move recommended to front, shift others down
		recommended := opts[recommendedIdx]
		opts = append(opts[:recommendedIdx], opts[recommendedIdx+1:]...)
		opts = append([]workflow.ToolOption{recommended}, opts...)
	}

	return opts
}

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

		// === Step 1: Package ===
		fmt.Println(stepStyle.Render("Step 1: Develop & Package"))
		fmt.Println()

		// Check if sample model exists to provide sensible defaults
		sampleModelPath := "./models/iris"
		useSampleDefaults := false
		if info, err := os.Stat(sampleModelPath); err == nil && info.IsDir() {
			useSampleDefaults = true
		}

		var modelName string
		if useSampleDefaults {
			modelName = "iris"
		}
		if err := huh.NewInput().
			Title("What's your model name?").
			Value(&modelName).
			Run(); err != nil {
			return err
		}
		modelName = strings.TrimSpace(modelName)

		var modelPath string
		if useSampleDefaults {
			modelPath = "./models/iris"
		}
		if err := huh.NewInput().
			Title("Where are your model files?").
			Value(&modelPath).
			Run(); err != nil {
			return err
		}
		modelPath = strings.TrimSpace(modelPath)
		expandedModelPath, err := expandHomePath(modelPath)
		if err != nil {
			return err
		}
		modelPath = expandedModelPath

		var artifactName string
		if useSampleDefaults {
			artifactName = "test-model/iris"
		}
		if err := huh.NewInput().
			Title("What should we call the artifact?").
			Value(&artifactName).
			Run(); err != nil {
			return err
		}
		artifactName = strings.TrimSpace(artifactName)

		var includeRAG bool
		if err := huh.NewConfirm().
			Title("Demonstrate RAG context?").
			Description("The selected path is shown in the tour; RAG files are not packaged yet.").
			Value(&includeRAG).
			Run(); err != nil {
			return err
		}

		var ragPath string
		if includeRAG {
			if err := huh.NewInput().
				Title("RAG context path to demonstrate:").
				Value(&ragPath).
				Run(); err != nil {
				return err
			}
			ragPath = strings.TrimSpace(ragPath)
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
		signSimulated := false
		verifySucceeded := false
		verifySimulated := false
		deploySucceeded := false
		deployVerified := false
		predictionVerified := false
		deployNoOp := false
		deployNewCommit := false

		// Just-in-time tool check for registry provider
		if !registryProvider.IsInstalled() {
			installed, err := workflow.EnsureToolInstalled(registryProvider.Name(), "packaging", interactive())
			if err != nil {
				return err
			}
			if !installed {
				fmt.Println(warningStyle.Render("Warning: Registry tool not installed"))
				fmt.Printf("   Install with: %s\n\n", registryProvider.InstallInstructions())
			} else {
				// Tool was just installed, re-check
				registryProvider, _ = workflow.GetRegistryProvider(localRegistryTool)
			}
		}

		if !registryProvider.IsInstalled() {
			fmt.Println(warningStyle.Render("Warning: Registry tool not installed"))
			fmt.Printf("   Install with: %s\n\n", registryProvider.InstallInstructions())
		} else {
			// Derive runtime from model format before packaging
			packageAnnotations := workflow.NewAnnotationSet()
			modelFormat := workflow.DetectModelFormatFromPath(modelPath)
			if modelFormat == workflow.ModelFormatSklearn {
				packageAnnotations.Runtime = "kserve-sklearnserver"
				packageAnnotations.Accelerator = "cpu"
				packageAnnotations.CUDAMin = ""
				packageAnnotations.MemoryMin = "256MiB"
			} else if modelFormat == workflow.ModelFormatPyTorch {
				packageAnnotations.Runtime = "pytorch"
			} else if modelFormat == workflow.ModelFormatTensorFlow {
				packageAnnotations.Runtime = "tensorflow"
			} else if modelFormat == workflow.ModelFormatONNX {
				packageAnnotations.Runtime = "onnx"
			} else {
				// Keep defaults for unknown formats
				if strings.Contains(strings.ToLower(modelName), "sklearn") ||
					strings.Contains(strings.ToLower(modelPath), "sklearn") {
					packageAnnotations.Runtime = "kserve-sklearnserver"
					packageAnnotations.Accelerator = "cpu"
					packageAnnotations.CUDAMin = ""
					packageAnnotations.MemoryMin = "256MiB"
				}
			}

			pf, err := workflow.NewPackageWorkflow(localRegistryTool)
			if err != nil {
				return err
			}
			pf.SetShowNextSteps(false)
			pf.SetPackageInfo(modelName, modelPath, artifactName, "", includeRAG, ragPath)
			pf.SetAnnotations(packageAnnotations)

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

		// === Step 2: Harden ===
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
				// Just-in-time check with option to install
				installed, err := workflow.EnsureToolInstalled(sbomTool, "SBOM generation", interactive())
				if err != nil {
					return err
				}
				if !installed {
					return fmt.Errorf("%s is required to harden this artifact; install it with: %s", sbomGenerator.Name(), sbomGenerator.InstallInstructions())
				}
				// Tool was just installed, re-check
				sbomGenerator, err = workflow.GetSBOMGenerator(sbomTool)
				if err != nil {
					return err
				}
			}

			annotations := workflow.NewAnnotationSet()

			// Derive runtime from model format (e.g., sklearn -> sklearn/MLServer, not vllm)
			// Use the model classification to detect format
			modelFormat := workflow.DetectModelFormatFromPath(modelPath)
			if modelFormat == workflow.ModelFormatSklearn {
				annotations.Runtime = "sklearn"
			} else if modelFormat == workflow.ModelFormatPyTorch {
				annotations.Runtime = "pytorch"
			} else if modelFormat == workflow.ModelFormatTensorFlow {
				annotations.Runtime = "tensorflow"
			} else if modelFormat == workflow.ModelFormatONNX {
				annotations.Runtime = "onnx"
			} else {
				// Keep default vllm for unknown formats (but sklearn should NEVER be vllm)
				if strings.Contains(strings.ToLower(modelName), "sklearn") ||
					strings.Contains(strings.ToLower(modelPath), "sklearn") {
					annotations.Runtime = "sklearn"
				}
			}

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
			fmt.Printf("  Runtime set to: %s (derived from model format: %s)\n", annotations.Runtime, modelFormat)

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
				return err
			}
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
		nextAction := "Press Enter to choose a signing tool."
		if skipSigning {
			nextAction = "Press Enter to choose how to publish the artifact."
		}
		displayInteractiveContext(ctxModel, nextAction)

		fmt.Println()

		// === Step 3: Sign ===
		if !skipSigning {
			fmt.Println(stepStyle.Render("Step 3: Supply Chain Check"))
			fmt.Println()

			signerTool := cfg.Signer
			if err := huh.NewSelect[string]().
				Title("Which signing approach should the wizard demonstrate?").
				Options(toolOptions(workflow.SignerOptions())...).
				Value(&signerTool).
				Run(); err != nil {
				return err
			}
			cfg.Signer = signerTool
			fmt.Printf("To execute this step outside the wizard: model-cli sign --artifact %s --signer %s\n", fullArtifact, cfg.Signer)

			sp, err := workflow.GetSigningProvider(cfg.Signer)
			if err != nil {
				return err
			}

			signSucceeded = false
			signSimulated = true

			// Map the signer choice to canonical annotation value
			signingFramework := cfg.Signer
			if cfg.Signer == "cosign" || cfg.Signer == "sigstore" {
				signingFramework = "sigstore-cosign"
			} else if cfg.Signer == "notation" || cfg.Signer == "notary" || cfg.Signer == "notaryv2" {
				signingFramework = "notation"
			}

			// Always update manifest with the chosen signing framework
			manifestPath := filepath.Join(modelPath, "manifest.json")
			manifest, err := workflow.ReadUnifiedOCIManifest(manifestPath)
			if err != nil {
				fmt.Printf("⚠ Warning: failed to update signing framework in manifest: %v\n", err)
			} else {
				if manifest.Annotations == nil {
					manifest.Annotations = make(map[string]string)
				}
				manifest.Annotations[workflow.AnnotationSigningFramework] = signingFramework
				if err := workflow.WriteUnifiedOCIManifest(manifest, manifestPath); err != nil {
					fmt.Printf("⚠ Warning: failed to write updated manifest: %v\n", err)
				} else {
					fmt.Printf("✓ Signing framework annotation updated: %s\n", signingFramework)
				}
			}

			if !sp.IsInstalled() {
				// Just-in-time check with option to install
				installed, err := workflow.EnsureToolInstalled(cfg.Signer, "signing", interactive())
				if err != nil {
					return err
				}
				if !installed {
					fmt.Println(warningStyle.Render("⚠ Signing tool not installed"))
					fmt.Printf("   Install with: %s\n", sp.InstallInstructions())
				} else {
					// Tool was just installed, re-check
					sp, err = workflow.GetSigningProvider(cfg.Signer)
					if err != nil {
						return err
					}
				}
			}

			// Demonstrate signing simulation (always happens, tool installed or not)
			fmt.Printf("Simulating signing %s with %s...\n", fullArtifact, cfg.Signer)
			fmt.Println(successStyle.Render("✓ Signing path demonstrated"))
			signSucceeded = true

			fmt.Println()
		}

		if !skipSigning {
			fmt.Println(stepStyle.Render("Demonstrating signature verification"))
			fmt.Println()

			sp, err := workflow.GetSigningProvider(cfg.Signer)
			if err != nil {
				return err
			}

			verifySucceeded = false
			verifySimulated = true

			if !sp.IsInstalled() {
				// Just-in-time check with option to install
				installed, err := workflow.EnsureToolInstalled(cfg.Signer, "signature verification", interactive())
				if err != nil {
					return err
				}
				if !installed {
					fmt.Println(warningStyle.Render("⚠ Signing tool not installed"))
					fmt.Printf("   Install with: %s\n", sp.InstallInstructions())
				} else {
					// Tool was just installed, re-check
					sp, err = workflow.GetSigningProvider(cfg.Signer)
					if err != nil {
						return err
					}
				}
			}

			// Demonstrate verification simulation (always happens, tool installed or not)
			fmt.Printf("Simulating signature verification for %s...\n", fullArtifact)
			fmt.Println(successStyle.Render("✓ Verification path demonstrated"))
			verifySucceeded = true

			fmt.Println()
		}

		fmt.Println(stepStyle.Render("Publishing artifact"))
		fmt.Println()
		publishArtifact := true
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
					huh.NewOption("a local Podman registry at localhost:5000 (recommended)", "local-podman"),
					huh.NewOption("an existing OCI registry", "existing"),
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
					// Offer to set up the registry
					var setupRegistry bool
					if err := huh.NewConfirm().
						Title("Set up a local Podman registry now?").
						Description("This will install podman if needed, initialize the VM, and start the registry container.").
						Value(&setupRegistry).
						Run(); err != nil {
						return err
					}

					if setupRegistry {
						fmt.Println()
						fmt.Println(stepStyle.Render("Setting up local registry..."))
						fmt.Println()
						if err := workflow.RegistrySetup(); err != nil {
							return fmt.Errorf("failed to set up local registry: %v", err)
						}
					} else {
						fmt.Println("\nStart a local registry with Podman, then run the wizard again:")
						fmt.Println("  brew install podman")
						fmt.Println("  podman machine init")
						fmt.Println("  podman machine start")
						fmt.Println("  podman run -d --rm --name model-cli-registry -p 5000:5000 registry:2")
						return fmt.Errorf("local OCI registry is unavailable at localhost:5000: %w", err)
					}
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
		var hasKubernetes bool = true
		var repoURL string
		var manifestPath string = "clusters/minikube/apps"
		var setupCluster bool
		var setupFlux bool
		if !skipDeploy {
			fmt.Println(stepStyle.Render("Step 5: GitOps Admission & Policy Enforcement"))
			fmt.Println()
			gitOpsTool := cfg.GitOps
			if err := huh.NewSelect[string]().
				Title("How would you like to promote the artifact?").
				Options(toolOptions(workflow.GitOpsOptions())...).
				Value(&gitOpsTool).
				Run(); err != nil {
				return err
			}
			cfg.GitOps = gitOpsTool
			if err := huh.NewConfirm().
				Title("Do you have a Kubernetes cluster?").
				Description("If no: the wizard can set up minikube and Flux for you.").
				Value(&hasKubernetes).
				Run(); err != nil {
				return err
			}

			// If no cluster, offer to set it up
			if !hasKubernetes {
				if err := huh.NewConfirm().
					Title("Set up a local minikube cluster now?").
					Description("This will install minikube and kubectl if needed, then start a cluster.").
					Value(&setupCluster).
					Run(); err != nil {
					return err
				}

				if setupCluster {
					// Get cluster configuration
					var clusterCPUs string
					var clusterMemory string
					if err := huh.NewInput().
						Title("Number of CPUs for minikube:").
						Placeholder("4").
						Value(&clusterCPUs).
						Run(); err != nil {
						return err
					}
					if clusterCPUs == "" {
						clusterCPUs = "4"
					}

					if err := huh.NewInput().
						Title("Memory for minikube (e.g., 8g):").
						Placeholder("8g").
						Value(&clusterMemory).
						Run(); err != nil {
						return err
					}
					if clusterMemory == "" {
						clusterMemory = "8g"
					}

					// Set up the cluster
					fmt.Println()
					fmt.Println(stepStyle.Render("Setting up minikube cluster..."))
					fmt.Println()
					if err := workflow.ClusterSetup(clusterCPUs, clusterMemory); err != nil {
						return fmt.Errorf("failed to set up cluster: %v", err)
					}

					hasKubernetes = true

					// Now offer Flux bootstrap
					if err := huh.NewConfirm().
						Title("Bootstrap Flux on this cluster?").
						Description("This will install Flux and set up the GitOps pipeline.").
						Value(&setupFlux).
						Run(); err != nil {
						return err
					}

					if setupFlux {
						// Get repo URL
						if err := huh.NewInput().
							Title("Git repository URL for Flux bootstrap:").
							Placeholder("ssh://git@github.com/you/cli-prototype.git").
							Value(&repoURL).
							Run(); err != nil {
							return err
						}
						repoURL = strings.TrimSpace(repoURL)

						// Get manifest path
						if err := huh.NewInput().
							Title("Flux sync path in repo:").
							Placeholder("./clusters/minikube").
							Value(&manifestPath).
							Run(); err != nil {
							return err
						}
						manifestPath = strings.TrimSpace(manifestPath)

						// If manifestPath is empty, use default
						if manifestPath == "" {
							manifestPath = "./clusters/minikube"
						}

						fmt.Println()
						fmt.Println(stepStyle.Render("Bootstrapping Flux..."))
						fmt.Println()
						if err := workflow.FluxBootstrap(repoURL, manifestPath); err != nil {
							return fmt.Errorf("failed to bootstrap Flux: %v", err)
						}
					}
				}
			}

			if hasKubernetes {
				if err := huh.NewInput().
					Title("Git repository URL:").
					Placeholder("https://github.com/you/model-manifests").
					Value(&repoURL).
					Run(); err != nil {
					return err
				}
				repoURL = strings.TrimSpace(repoURL)
				if err := huh.NewInput().
					Title("Manifest path in repo:").
					Value(&manifestPath).
					Run(); err != nil {
					return err
				}
				manifestPath = strings.TrimSpace(manifestPath)
			}
			fmt.Println()
		}

		if hasKubernetes && !skipDeploy {
			fmt.Println(stepStyle.Render("Deploying through GitOps"))
			fmt.Println()

			// Select, do not install: show what the cluster already provides via Git
			ingredients := workflow.DetectIngredients()
			fmt.Println("Cluster ingredients (managed by Flux in Git):")
			for _, ing := range ingredients {
				status := "-"
				if ing.Present {
					status = "+"
				}
				fmt.Printf("  [%s] %s - %s\n", status, ing.Name, ing.Description)
			}
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
				// Step 6.5: Select serving topology (restored prompt)
				// This was removed in a2be6b0 and needs to be restored for LLM path accessibility
				modelFormat := workflow.DetectModelFormatFromPath(modelPath)
				servingTopology := cfg.ServingTopology
				if servingTopology == "" {
					servingTopology = cfg.Runtime
				}
				if servingTopology == "" {
					servingTopology = getServingTopologyOptions(modelFormat).Recommended()
				}
				
				// Prompt user for serving topology selection
				if err := huh.NewSelect[string]().
					Title("Which serving topology should the wizard demonstrate?").
					Options(toolOptions(getServingTopologyOptions(modelFormat))...).
					Value(&servingTopology).
					Run(); err != nil {
					return err
				}
				
				// For LLM topology, prompt for Hugging Face model id
				var hfModelID string
				if servingTopology == "kserve-vllm" {
					if err := huh.NewInput().
						Title("Hugging Face model ID:").
						Placeholder("facebook/opt-125m").
						Value(&hfModelID).
						Run(); err != nil {
						return err
					}
					hfModelID = strings.TrimSpace(hfModelID)
					if hfModelID == "" {
						hfModelID = "facebook/opt-125m" // default
					}
					// Derive a DNS-safe InferenceService name from the HF model ID
					// Use the last path segment (e.g., "facebook/opt-125m" -> "opt-125m")
					// and sanitize it (replace / with -)
					modelName = filepath.Base(hfModelID)
					// Replace any remaining invalid characters with -
					modelName = strings.ReplaceAll(modelName, "/", "-")
					modelName = strings.ReplaceAll(modelName, "_", "-")
					modelName = strings.ToLower(modelName)
				}
				
				// Map serving topology to actual runtime that exists in the cluster
				// kserve-vllm -> kserve-huggingfaceserver (which uses vLLM engine as backend)
				// kserve -> derive from model format (e.g., sklearn -> kserve-sklearnserver)
				// vllm -> not an InferenceService deployment, skip generation
				var deployRuntime string
				var skipDeployBecauseOfTopology bool
				
				switch servingTopology {
				case "kserve-vllm":
					deployRuntime = "kserve-huggingfaceserver"
				case "kserve":
					// Derive concrete runtime from model format
					var runtimeInfo *workflow.RuntimeInfo
					runtimeInfo, err = workflow.DeriveRuntimeFromModelPath(modelPath, modelName, repoURL, "")
					if err != nil {
						// Fallback to kserve-sklearnserver if we can't derive
						deployRuntime = "kserve-sklearnserver"
					} else {
						deployRuntime = runtimeInfo.Runtime
					}
				case "vllm":
					// Direct vLLM serving is not a KServe InferenceService deployment
					// Print honest message and skip InferenceService generation
					fmt.Println("Note: direct vLLM serving is demonstrated, not deployed through GitOps.")
					skipDeployBecauseOfTopology = true
				default:
					deployRuntime = servingTopology
				}
				
				cfg.Runtime = deployRuntime
				cfg.ServingTopology = servingTopology
				
				// If vllm topology (direct serving), skip the deploy
				if skipDeployBecauseOfTopology {
					skipDeploy = true
				}

				// If LLM topology is selected, update manifest.json with LLM annotations
				// so Step 6 can read them and show the GPU/memory requirements
				if servingTopology == "kserve-vllm" {
					manifestJSONPath := filepath.Join(modelPath, "manifest.json")
					if manifest, err := workflow.ReadUnifiedOCIManifest(manifestJSONPath); err == nil {
						// Ensure annotations map exists
						if manifest.Annotations == nil {
							manifest.Annotations = make(map[string]string)
						}
						// Add/update LLM-specific annotations
						manifest.Annotations[workflow.AnnotationRuntime] = "kserve-huggingfaceserver"
						manifest.Annotations[workflow.AnnotationAccelerator] = "gpu"
						manifest.Annotations[workflow.AnnotationMemoryMin] = "24Gi"
						// Write the updated manifest back
						if err := workflow.WriteUnifiedOCIManifest(manifest, manifestJSONPath); err != nil {
							// Don't fail the workflow, just log a warning
							fmt.Printf("Warning: failed to update manifest.json with LLM annotations: %v\n", err)
						}
					}
				}

				wf, err := workflow.NewDeployWorkflow(cfg.GitOps, cfg.Registry)
				if err != nil {
					return err
				}
				wf.SetModelInfo(modelName, modelPath, repoURL, manifestPath)
				wf.SetRuntime(cfg.Runtime)
				// Pass HF model ID to workflow for manifest generation
				if servingTopology == "kserve-vllm" {
					wf.SetHFModelID(hfModelID)
				}

				if err := wf.Run(); err != nil {
					return err
				}

				// Check if a manifest was generated but not yet committed
				if manifestFile := wf.ManifestGenerated(); manifestFile != "" {
					// Prompt: Commit and push the generated manifest?
					// Default to Yes for Enter-through golden path
					commitManifest := true
					if err := huh.NewConfirm().
						Title("Commit and push the generated manifest?").
						Description(fmt.Sprintf("The InferenceService manifest at %s is ready to be committed.", manifestFile)).
						Value(&commitManifest).
						Run(); err != nil {
						return err
					}
					
					if commitManifest {
						// Stage ONLY the manifest file
						if out, err := exec.Command("git", "add", manifestFile).CombinedOutput(); err != nil {
							return fmt.Errorf("git add %s failed: %s", manifestFile, out)
						}
						commit := exec.Command("git", "commit", "-m", fmt.Sprintf("wizard: deploy %s via GitOps", modelName))
						commit.Env = append(os.Environ(), "GIT_EDITOR=true")
						if commitOut, err := commit.CombinedOutput(); err != nil {
							outStr := string(commitOut)
							if !strings.Contains(outStr, "nothing to commit") &&
								!strings.Contains(outStr, "no changes added") {
								return fmt.Errorf("git commit failed: %s", outStr)
							}
							fmt.Printf("⚠ Manifest already committed\n")
						} else {
							fmt.Printf("✓ Committed manifest: %s\n", manifestFile)
							// Mark that a new commit was created for recap wording
							deployNewCommit = true
						}
						
						// Push to remote
						if pushOut, err := exec.Command("git", "push").CombinedOutput(); err != nil {
							return fmt.Errorf("git push failed: %s", pushOut)
						}
						fmt.Printf("✓ Pushed to git repository\n")
						// Suppress duplicate banner on re-run
						wf.SetQuiet(true)
					}
					
					// Re-run the deploy workflow to continue with pre-flight checks
					// If user committed, it will pass; if not, it will fail at dirty-tree check
					if err := wf.Run(); err != nil {
						return err
					}
					// After re-run, check if manifest is still uncommitted
					// This means user chose No and pre-flight was bypassed again
					if manifestStillDirty := wf.ManifestGenerated(); manifestStillDirty != "" {
						return fmt.Errorf("working tree is not clean. Commit changes first")
					}
				}

				// Set the results from the workflow
				deploySucceeded = true
				deployVerified = wf.ReadyVerified()
				predictionVerified = wf.PredictionVerified()
				deployNoOp = !wf.Deployed()
				deployNewCommit = deployNewCommit || wf.NewCommit()
			}
			fmt.Println()
		} else if !skipDeploy {
			fmt.Println(infoStyle.Render("No cluster connected; GitOps admission and policy enforcement simulated."))
		}

		if !skipDeploy {
			fmt.Println()
			ctxModel.SetStep(6)
			displayInteractiveContext(ctxModel, "Press Enter to continue to infrastructure and resource orchestration.")
			fmt.Println()

			fmt.Println(stepStyle.Render("Step 6: Infrastructure & Resource Orchestration"))
			
			// Read declared requirements from manifest's CNCF AI annotations
			manifestPath := filepath.Join(modelPath, "manifest.json")
			var annotations map[string]string
			if manifest, err := workflow.ReadUnifiedOCIManifest(manifestPath); err == nil && manifest.Annotations != nil {
				annotations = manifest.Annotations
			}
			
			// Query actual nodes
			nodesOut, nodesErr := exec.Command("kubectl", "get", "nodes", "-o", "json").CombinedOutput()
			if nodesErr != nil {
				fmt.Printf("⚠ Node query failed: %v\n", nodesErr)
			} else {
				// Parse node JSON
				var nodesList struct {
					Items []struct {
						Metadata struct {
							Name   string            `json:"name"`
							Labels map[string]string `json:"labels"`
						} `json:"metadata"`
						Status struct {
							Allocatable map[string]string `json:"allocatable"`
							Capacity   map[string]string `json:"capacity"`
						} `json:"status"`
					} `json:"items"`
				}
				if err := json.Unmarshal(nodesOut, &nodesList); err == nil {
					// Check each declared requirement
					// org.cncf.ai.accelerator
						if accel, ok := annotations[workflow.AnnotationAccelerator]; ok && accel != "" {
							if accel == "cpu" {
								// CPU is satisfied by any Ready node
								fmt.Printf("✓ Accelerator: %s (satisfied by any node)\n", accel)
							} else {
								// Check for GPU labels
								found := false
								for _, node := range nodesList.Items {
									if _, hasGPU := node.Metadata.Labels["nvidia.com/gpu.product"]; hasGPU {
										fmt.Printf("✓ Accelerator: %s (node %s has GPU)\n", accel, node.Metadata.Name)
										found = true
										break
									}
								}
								if !found {
									fmt.Printf("⚠ Accelerator: %s (no nodes with GPU labels found)\n", accel)
								}
							}
						}
						
						// org.cncf.ai.accelerator.cuda.min
						if cudaMin, ok := annotations[workflow.AnnotationCUDAVersionMin]; ok && cudaMin != "" {
							// Check node labels for CUDA version
							// Note: This is informational - we don't fail the step
							fmt.Printf("→ CUDA min: %s\n", cudaMin)
						}
						
						// org.cncf.ai.resource.memory.min
						if memMin, ok := annotations[workflow.AnnotationMemoryMin]; ok && memMin != "" {
							memMinKi := parseMemory(memMin)
							if memMinKi > 0 {
								found := false
								for _, node := range nodesList.Items {
									if memStr, ok := node.Status.Allocatable["memory"]; ok {
										nodeMemKi := parseMemory(memStr)
										if nodeMemKi >= memMinKi {
											fmt.Printf("✓ Memory min: %s (node %s has %s allocatable)\n", memMin, node.Metadata.Name, memStr)
											found = true
										}
									}
								}
								if !found {
									fmt.Printf("⚠ Memory min: %s (no nodes satisfy requirement)\n", memMin)
								}
							}
						}
						
						// ai.node.gpu.type
						if gpuType, ok := annotations[workflow.AnnotationGPUType]; ok && gpuType != "" {
							found := false
							for _, node := range nodesList.Items {
								if nodeGPUType, ok := node.Metadata.Labels["nvidia.com/gpu.product"]; ok && nodeGPUType == gpuType {
									fmt.Printf("✓ GPU type: %s (node %s has matching GPU)\n", gpuType, node.Metadata.Name)
									found = true
									break
								}
							}
							if !found {
								fmt.Printf("⚠ GPU type: %s (no nodes with matching GPU)\n", gpuType)
							}
						}
						
						// ai.node.vram.min
						if vramMin, ok := annotations[workflow.AnnotationVRAMMin]; ok && vramMin != "" {
							vramMinKi := parseMemory(vramMin)
							if vramMinKi > 0 {
								found := false
								for _, node := range nodesList.Items {
									// GPU VRAM is in node labels, not capacity
									if vramStr, ok := node.Metadata.Labels["nvidia.com/gpu.memory"]; ok {
										vramNodeKi := parseMemory(vramStr)
										if vramNodeKi >= vramMinKi {
											fmt.Printf("✓ vRAM min: %s (node %s has %s)\n", vramMin, node.Metadata.Name, vramStr)
											found = true
										}
									}
								}
								if !found {
									fmt.Printf("⚠ vRAM min: %s (no nodes satisfy requirement)\n", vramMin)
								}
							}
						}
						
						// ai.node.gpu.topology
						if gpuTopo, ok := annotations[workflow.AnnotationGPUTopology]; ok && gpuTopo != "" {
							fmt.Printf("→ GPU topology: %s\n", gpuTopo)
						}
					}
				}

			ctxModel.SetStep(7)
			displayInteractiveContext(ctxModel, "Press Enter to continue to deployment.")
			fmt.Println()

			fmt.Println()
			fmt.Println(stepStyle.Render("Step 7: Runtime Execution & Optimization"))

			// Step 7: Real post-deploy runtime audit
			// Only run if we actually deployed
			if !skipDeploy && deploySucceeded {
				// Get the InferenceService
				isvcOut, isvcErr := exec.Command("kubectl", "get", "inferenceservice", modelName, "-n", "models", "-o", "json").CombinedOutput()
				if isvcErr != nil {
					fmt.Printf("⚠ InferenceService query failed: %v\n", isvcErr)
				} else {
					// Parse InferenceService JSON
					var isvc struct {
						Spec struct {
							Predictor struct {
								Model struct {
									Runtime    string `json:"runtime"`
									Resources struct {
										Requests map[string]string `json:"requests"`
										Limits   map[string]string `json:"limits"`
									} `json:"resources"`
								} `json:"model"`
							} `json:"predictor"`
						} `json:"spec"`
						Status struct {
							Conditions []struct {
								Type   string `json:"type"`
								Status string `json:"status"`
							} `json:"conditions"`
						} `json:"status"`
					}
					if err := json.Unmarshal(isvcOut, &isvc); err == nil {
						// Compare deployed runtime vs declared annotation
						if deployedRuntime := isvc.Spec.Predictor.Model.Runtime; deployedRuntime != "" {
							if declaredRuntime, ok := annotations[workflow.AnnotationRuntime]; ok && declaredRuntime != "" {
								if deployedRuntime == declaredRuntime {
									fmt.Printf("✓ Runtime: %s (matches declared %s)\n", deployedRuntime, declaredRuntime)
								} else {
									fmt.Printf("⚠ Runtime: deployed=%s, declared=%s (mismatch)\n", deployedRuntime, declaredRuntime)
								}
							} else {
								fmt.Printf("→ Runtime: %s (no declared runtime annotation)\n", deployedRuntime)
							}
						}
						
						// Report resource requests/limits vs declared memory min
						if memMin, ok := annotations[workflow.AnnotationMemoryMin]; ok && memMin != "" {
							memMinKi := parseMemory(memMin)
							if memMinKi > 0 {
								if reqMem, ok := isvc.Spec.Predictor.Model.Resources.Requests["memory"]; ok {
									reqMemKi := parseMemory(reqMem)
									if reqMemKi >= memMinKi {
										fmt.Printf("✓ Memory request: %s (satisfies min %s)\n", reqMem, memMin)
									} else {
										fmt.Printf("⚠ Memory request: %s (below min %s)\n", reqMem, memMin)
									}
								} else {
									fmt.Printf("→ Memory: no request declared\n")
								}
								
								if limitMem, ok := isvc.Spec.Predictor.Model.Resources.Limits["memory"]; ok {
									limitMemKi := parseMemory(limitMem)
									if limitMemKi >= memMinKi {
										fmt.Printf("✓ Memory limit: %s (satisfies min %s)\n", limitMem, memMin)
									} else {
										fmt.Printf("⚠ Memory limit: %s (below min %s)\n", limitMem, memMin)
									}
								} else {
									fmt.Printf("→ Memory: no limit declared\n")
								}
							}
						}
						
						// Report InferenceService Ready status
						ready := false
						for _, cond := range isvc.Status.Conditions {
							if cond.Type == "Ready" && cond.Status == "True" {
								ready = true
								break
							}
						}
						if ready {
							fmt.Printf("✓ InferenceService %s is Ready\n", modelName)
						} else {
							fmt.Printf("⚠ InferenceService %s not Ready\n", modelName)
						}
					}
				}
			} else {
				fmt.Println("→ Skipped runtime audit: no cluster deployment")
			}
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
			signSimulated:      signSimulated,
			verifySucceeded:    verifySucceeded,
			verifySimulated:    verifySimulated,
			publishSucceeded:   publishDestination != "",
			publishDestination: publishDestination,
			deploySucceeded:    deploySucceeded,
			deployVerified:     deployVerified,
			predictionVerified: predictionVerified,
			deployNoOp:         deployNoOp,
			deployNewCommit:    deployNewCommit,
			modelName:          modelName,
			artifactName:       artifactName,
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
		fmt.Println("Inspect the manifest:")
		for _, instruction := range wizardManifestInstructions(modelPath, result) {
			fmt.Printf("  %s\n", instruction)
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
		fmt.Println("Inspecting local manifest...")
	} else {
		provider, err := workflow.GetRegistryProvider(registry)
		if err != nil {
			return err
		}
		annotations, err = provider.FetchManifestAnnotations(destination + "/" + artifact)
		if err != nil {
			return fmt.Errorf("failed to fetch published manifest: %w", err)
		}
		fmt.Printf("Inspecting published manifest at %s/%s...\n", destination, artifact)
	}
	fmt.Printf("  %s\n", summarizeManifest(annotations))
	if destination == "" {
		fmt.Printf("  Inspect details: cat %s\n", filepath.Join(modelPath, "manifest.json"))
	} else {
		fmt.Printf("  Inspect details: oras manifest fetch %s/%s\n", destination, artifact)
	}
	return nil
}

func summarizeManifest(annotations map[string]string) string {
	parts := []string{"Manifest includes"}
	if artifactType := annotations[workflow.AnnotationArtifactType]; artifactType != "" {
		parts = append(parts, artifactType+" artifact metadata")
	}
	if format := annotations[workflow.AnnotationPackagingFormat]; format != "" {
		parts = append(parts, format+" packaging")
	}
	if sbom := annotations[workflow.AnnotationSBOMFormat]; sbom != "" {
		parts = append(parts, sbom+" SBOM metadata")
	}
	if mofClass := annotations[workflow.AnnotationMOFClass]; mofClass != "" {
		parts = append(parts, "MOF Class "+mofClass)
	}
	if runtime := annotations[workflow.AnnotationRuntime]; runtime != "" {
		parts = append(parts, runtime+" runtime requirements")
	}
	if len(parts) == 1 {
		return "Manifest retrieved; no recognized Model CLI annotations found."
	}
	return strings.Join(parts, ", ") + "."
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
