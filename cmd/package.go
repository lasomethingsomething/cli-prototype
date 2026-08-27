package cmd

import (
	"fmt"

	"github.com/lasomethingsomething/cli-prototype/config"
	"github.com/lasomethingsomething/cli-prototype/internal/workflow"
	"github.com/spf13/cobra"
)

var packageCmd = &cobra.Command{
	Use:   "package",
	Short: "Package a model or agentic skill as an OCI artifact",
	Long: `Package your model or agentic skill as an OCI artifact for distribution and deployment.

This command helps you package models, prompts, RAG context, or agentic skills
(conforming to agentskills.io standard format) into a single OCI artifact that can be
stored in registries and deployed anywhere. No specific tool requirement - uses OCI Image Spec standard.

Supported registry tools (mutually exclusive):
` + workflow.RegistryOptions().Bullets() + `

Harbor and other OCI registries are used through oras:
--registry oras --registry-url <harbor-host>/<project>

SBOM generation and MOF classification are a separate step: run
'model-cli harden' on the same --model-path after packaging.

Signing and SLSA provenance are also a separate step that runs afterwards
(Phase 1, Step 3: Supply Chain Check): see 'model-cli sign'.

Examples:
  # Package a model
  model-cli package
  model-cli package --model phi-4-mini --registry oras --output my-model:latest

  # Package an agentic skill
  model-cli package --model my-skill --model-path ./skills/my-skill --skill
  model-cli package --skill --model-path ./my-skill --artifact my-org/my-skill:v1

  # Then sign and attest the packaged artifact
  model-cli sign --artifact my-model:latest --signer cosign`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Load()

		// Get flags
		isSkillFlag, _ := cmd.Flags().GetBool("skill")
		// Node requirement flags for infrastructure orchestration (Story #68)
		// Runtime execution flags for Story #69

		// Interactive prompts if not set in config or via flags
		if err := askSelectToolIfEmpty(cmd, "registry", &cfg.Registry, "Select Registry tool:", "Choose how you want to package your model", workflow.RegistryOptions()); err != nil {
			return err
		}

		// Get packaging information
		var modelName string
		if err := askString(cmd, "model", &modelName, "Model name:", "What is the name of your model? (e.g., phi-4-mini)"); err != nil {
			return err
		}

		var modelPath string
		if err := askString(cmd, "model-path", &modelPath, "Model path:", "Path to your model files or directory"); err != nil {
			return err
		}

		var artifactName string
		if err := askString(cmd, "artifact", &artifactName, "Artifact name:", "What would you like to name the OCI artifact? (e.g., my-org/my-model:latest)"); err != nil {
			return err
		}

		var registryURL string
		if err := askString(cmd, "registry-url", &registryURL, "Registry URL:", "Where should we push the artifact? (e.g., ghcr.io or leave empty for local)"); err != nil {
			return err
		}

		var includeRAG bool
		if err := askConfirm(cmd, "include-rag", &includeRAG, "Include RAG context?", "Do you want to package RAG (Retrieval-Augmented Generation) context with your model?"); err != nil {
			return err
		}

		var ragPath string
		if includeRAG {
			if err := askString(cmd, "rag-path", &ragPath, "RAG context path:", "Path to your RAG context files"); err != nil {
				return err
			}
		}

		// Collect CNCF AI Interoperability Profile annotations
		annotations := workflow.NewAnnotationSet()

		// Runtime annotations
		if err := askString(cmd, "runtime", &annotations.Runtime, "Runtime:", "Serving runtime (e.g., vllm, kserve)"); err != nil {
			return err
		}

		if err := askSelect(cmd, "accelerator", &annotations.Accelerator, "Accelerator:", "Hardware accelerator requirement", []string{"nvidia-gpu", "amd-gpu", "intel-gpu", "cpu", "none"}); err != nil {
			return err
		}

		if err := askCUDAMin(cmd, annotations); err != nil {
			return err
		}

		if err := askString(cmd, "memory-min", &annotations.MemoryMin, "Minimum memory:", "Minimum memory required (e.g., 24GiB)"); err != nil {
			return err
		}

		// Node requirement annotations for infrastructure orchestration (Story #68)
		if err := askString(cmd, "gpu-type", &annotations.GPUType, "GPU type:", "Specific GPU type required (e.g., nvidia-a100, nvidia-h100, leave empty if not applicable)"); err != nil {
			return err
		}

		if err := askString(cmd, "vram-min", &annotations.VRAMMin, "Minimum vRAM per GPU:", "Minimum vRAM required per GPU (e.g., 40GiB, 80GiB, leave empty if not applicable)"); err != nil {
			return err
		}

		if err := askString(cmd, "gpu-topology", &annotations.GPUTopology, "GPU topology:", "GPU topology requirement (e.g., 8xH100, 4xA100, leave empty if not applicable)"); err != nil {
			return err
		}

		// Runtime execution annotations for Story #69
		if err := askString(cmd, "runtime-type", &annotations.RuntimeType, "Runtime type:", "Specific runtime for serving (e.g., vllm, kserve, leave empty to use default)"); err != nil {
			return err
		}

		if err := askSelect(cmd, "layer-dedup", &annotations.LayerDeduplication, "Layer deduplication:", "Enable layer deduplication optimization for large models", []string{"true", "false", ""}); err != nil {
			return err
		}

		// Skill-only annotations: prompt only when packaging a skill.
		if cmd.Flags().Changed("dlc-endpoint") || isSkillFlag {
			if err := askString(cmd, "dlc-endpoint", &annotations.ReferenceSkillDLC, "Reference Skill DLC endpoint:", "Endpoint for dynamic skill loading (e.g., https://dlc.example.com, leave empty if not applicable)"); err != nil {
				return err
			}
		}
		if cmd.Flags().Changed("skill-refs") || isSkillFlag {
			if err := askString(cmd, "skill-refs", &annotations.SkillReferences, "Skill references:", "Comma-separated list of skill references (e.g., skill:sha256:abc,skill:sha256:def)"); err != nil {
				return err
			}
		}

		if err := requireValues("model", modelName, "model-path", modelPath, "artifact", artifactName); err != nil {
			return err
		}

		// All inputs collected: remember the tool choice for next time.
		warnIfSaveFails(config.Save(cfg))

		// Create packaging workflow
		pf, err := workflow.NewPackageWorkflow(cfg.Registry)
		if err != nil {
			return err
		}

		// Set packaging info
		pf.SetPackageInfo(modelName, modelPath, artifactName, registryURL, includeRAG, ragPath)
		pf.SetAnnotations(annotations)
		pf.SetIsSkill(isSkillFlag)

		artifactType := "model"
		if isSkillFlag {
			artifactType = "skill"
		}
		fmt.Printf("\nPackaging your %s...", artifactType)
		fmt.Println()
		return pf.Run()
	},
}

// askCUDAMin collects the minimum CUDA version. CUDA only exists on NVIDIA
// hardware, so for any other accelerator (cpu, amd-gpu, ...) the default is
// dropped and the question is skipped: a stray cuda.min annotation would make
// 'validate nodes' and 'schedule' reject perfectly suitable nodes (Issue #103).
// An explicit --cuda-min is always respected.
func askCUDAMin(cmd *cobra.Command, annotations *workflow.AnnotationSet) error {
	if annotations.Accelerator != "nvidia-gpu" && !cmd.Flags().Changed("cuda-min") {
		annotations.CUDAMin = ""
		return nil
	}
	return askString(cmd, "cuda-min", &annotations.CUDAMin, "Minimum CUDA version:", "Minimum CUDA version required (e.g., 12.1, leave empty if not applicable)")
}

func init() {
	rootCmd.AddCommand(packageCmd)
	packageCmd.Flags().String("registry", "", "Registry tool: "+workflow.RegistryOptions().Summary())
	packageCmd.Flags().String("model", "", "Model name (e.g., phi-4-mini)")
	packageCmd.Flags().String("model-path", "", "Path to model files or directory")
	packageCmd.Flags().String("artifact", "", "OCI artifact name (e.g., my-org/my-model:latest)")
	packageCmd.Flags().String("registry-url", "", "Registry URL (e.g., ghcr.io)")
	packageCmd.Flags().Bool("include-rag", false, "Include RAG context")
	packageCmd.Flags().String("rag-path", "", "Path to RAG context files")
	packageCmd.Flags().String("runtime", "", "Serving runtime: vllm or kserve")
	packageCmd.Flags().String("accelerator", "", "Hardware accelerator: nvidia-gpu, amd-gpu, intel-gpu, cpu, or none")
	packageCmd.Flags().String("cuda-min", "", "Minimum CUDA version (e.g., 12.1)")
	packageCmd.Flags().String("memory-min", "", "Minimum memory (e.g., 24GiB)")
	// Node requirement flags for infrastructure orchestration (Story #68)
	packageCmd.Flags().String("gpu-type", "", "Specific GPU type (e.g., nvidia-a100, nvidia-h100)")
	packageCmd.Flags().String("vram-min", "", "Minimum vRAM per GPU (e.g., 40GiB, 80GiB)")
	packageCmd.Flags().String("gpu-topology", "", "GPU topology requirement (e.g., 8xH100, 4xA100)")
	// Runtime execution flags for Story #69
	packageCmd.Flags().String("runtime-type", "", "Specific runtime for serving (e.g., vllm, kserve)")
	packageCmd.Flags().String("layer-dedup", "", "Enable layer deduplication optimization (true/false)")
	packageCmd.Flags().String("dlc-endpoint", "", "Reference Skill DLC endpoint URL")
	packageCmd.Flags().String("skill-refs", "", "Comma-separated list of skill references")
	packageCmd.Flags().Bool("skill", false, "Package as an agentic skill (agentskills.io format)")
}
