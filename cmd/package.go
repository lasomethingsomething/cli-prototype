package cmd

import (
	"fmt"

	"github.com/charmbracelet/huh"
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
stored in registries and deployed anywhere.

The command also supports automatic signing at the point of creation using
Sigstore (cosign) or Notary v2 (notation) for supply chain security, and
generates SLSA provenance attestations by default for immutable provenance tracking.

Examples:
  # Package a model
  model-cli package
  model-cli package --model phi-4-mini --registry oras --output my-model:latest

  # Package an agentic skill
  model-cli package --model my-skill --model-path ./skills/my-skill --skill
  model-cli package --skill --model-path ./my-skill --artifact my-org/my-skill:v1

  # Package and sign automatically
  model-cli package --model phi-4-mini --artifact my-model:latest --sign --signer sigstore

  # Package without provenance (disable with flag)
  model-cli package --model phi-4-mini --artifact my-model:latest --generate-provenance=false`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Load()

		// Get flags
		registryFlag, _ := cmd.Flags().GetString("registry")
		modelNameFlag, _ := cmd.Flags().GetString("model")
		modelPathFlag, _ := cmd.Flags().GetString("model-path")
		artifactNameFlag, _ := cmd.Flags().GetString("artifact")
		registryURLFlag, _ := cmd.Flags().GetString("registry-url")
		includeRAGFlag, _ := cmd.Flags().GetBool("include-rag")
		ragPathFlag, _ := cmd.Flags().GetString("rag-path")
		runtimeFlag, _ := cmd.Flags().GetString("runtime")
		acceleratorFlag, _ := cmd.Flags().GetString("accelerator")
		cudaMinFlag, _ := cmd.Flags().GetString("cuda-min")
		memoryMinFlag, _ := cmd.Flags().GetString("memory-min")
		mofClassFlag, _ := cmd.Flags().GetString("mof-class")
		mofComponentsFlag, _ := cmd.Flags().GetString("mof-components")
		isSkillFlag, _ := cmd.Flags().GetBool("skill")
		signFlag, _ := cmd.Flags().GetBool("sign")
		signerFlag, _ := cmd.Flags().GetString("signer")
		provenanceFlag, _ := cmd.Flags().GetBool("generate-provenance")
		// Node requirement flags for infrastructure orchestration (Story #68)
		gpuTypeFlag, _ := cmd.Flags().GetString("gpu-type")
		vramMinFlag, _ := cmd.Flags().GetString("vram-min")
		gpuTopologyFlag, _ := cmd.Flags().GetString("gpu-topology")
		// Runtime execution flags for Story #69
		runtimeTypeFlag, _ := cmd.Flags().GetString("runtime-type")
		layerDedupFlag, _ := cmd.Flags().GetString("layer-dedup")
		dlcEndpointFlag, _ := cmd.Flags().GetString("dlc-endpoint")
		skillRefsFlag, _ := cmd.Flags().GetString("skill-refs")

		// Interactive prompts if not set in config or via flags
		if cfg.Registry == "" && registryFlag == "" {
			var registryTool string
			if err := huh.NewSelect[string]().
				Title("Select Registry tool:").
				Description("Choose how you want to package your model").
				Options(huh.NewOptions("oras", "modelpack")...).
				Value(&registryTool).
				Run(); err != nil {
				return err
			}
			cfg.Registry = registryTool
		} else if registryFlag != "" {
			cfg.Registry = registryFlag
		}

		// Save config for future use
		if err := config.Save(cfg); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		// Get packaging information
		var modelName string
		if modelNameFlag != "" {
			modelName = modelNameFlag
		} else {
			if err := huh.NewInput().
				Title("Model name:").
				Description("What is the name of your model? (e.g., phi-4-mini)").
				Value(&modelName).
				Run(); err != nil {
				return err
			}
		}

		var modelPath string
		if modelPathFlag != "" {
			modelPath = modelPathFlag
		} else {
			if err := huh.NewInput().
				Title("Model path:").
				Description("Path to your model files or directory").
				Value(&modelPath).
				Run(); err != nil {
				return err
			}
		}

		var artifactName string
		if artifactNameFlag != "" {
			artifactName = artifactNameFlag
		} else {
			if err := huh.NewInput().
				Title("Artifact name:").
				Description("What would you like to name the OCI artifact? (e.g., my-org/my-model:latest)").
				Value(&artifactName).
				Run(); err != nil {
				return err
			}
		}

		var registryURL string
		if registryURLFlag != "" {
			registryURL = registryURLFlag
		} else {
			if err := huh.NewInput().
				Title("Registry URL:").
				Description("Where should we push the artifact? (e.g., ghcr.io or leave empty for local)").
				Value(&registryURL).
				Run(); err != nil {
				return err
			}
		}

		var includeRAG bool
		if includeRAGFlag {
			includeRAG = true
		} else {
			if err := huh.NewConfirm().
				Title("Include RAG context?").
				Description("Do you want to package RAG (Retrieval-Augmented Generation) context with your model?").
				Value(&includeRAG).
				Run(); err != nil {
				return err
			}
		}

		var ragPath string
		if includeRAG {
			if ragPathFlag != "" {
				ragPath = ragPathFlag
			} else {
				if err := huh.NewInput().
					Title("RAG context path:").
					Description("Path to your RAG context files").
					Value(&ragPath).
					Run(); err != nil {
					return err
				}
			}
		}

		// Collect CNCF AI Interoperability Profile annotations
		annotations := workflow.NewAnnotationSet()

		// Runtime annotations
		if runtimeFlag != "" {
			annotations.Runtime = runtimeFlag
		} else {
			if err := huh.NewInput().
				Title("Runtime:").
				Description("Serving runtime (e.g., vllm, kserve)").
				Value(&annotations.Runtime).
				Run(); err != nil {
				return err
			}
		}

		if acceleratorFlag != "" {
			annotations.Accelerator = acceleratorFlag
		} else {
			if err := huh.NewSelect[string]().
				Title("Accelerator:").
				Description("Hardware accelerator requirement").
				Options(huh.NewOptions("nvidia-gpu", "amd-gpu", "intel-gpu", "cpu", "none")...).
				Value(&annotations.Accelerator).
				Run(); err != nil {
				return err
			}
		}

		if cudaMinFlag != "" {
			annotations.CUDAMin = cudaMinFlag
		} else {
			if err := huh.NewInput().
				Title("Minimum CUDA version:").
				Description("Minimum CUDA version required (e.g., 12.1, leave empty if not applicable)").
				Value(&annotations.CUDAMin).
				Run(); err != nil {
				return err
			}
		}

		if memoryMinFlag != "" {
			annotations.MemoryMin = memoryMinFlag
		} else {
			if err := huh.NewInput().
				Title("Minimum memory:").
				Description("Minimum memory required (e.g., 24GiB)").
				Value(&annotations.MemoryMin).
				Run(); err != nil {
				return err
			}
		}

		// Node requirement annotations for infrastructure orchestration (Story #68)
		if gpuTypeFlag != "" {
			annotations.GPUType = gpuTypeFlag
		} else {
			if err := huh.NewInput().
				Title("GPU type:").
				Description("Specific GPU type required (e.g., nvidia-a100, nvidia-h100, leave empty if not applicable)").
				Value(&annotations.GPUType).
				Run(); err != nil {
				return err
			}
		}

		if vramMinFlag != "" {
			annotations.VRAMMin = vramMinFlag
		} else {
			if err := huh.NewInput().
				Title("Minimum vRAM per GPU:").
				Description("Minimum vRAM required per GPU (e.g., 40GiB, 80GiB, leave empty if not applicable)").
				Value(&annotations.VRAMMin).
				Run(); err != nil {
				return err
			}
		}

		if gpuTopologyFlag != "" {
			annotations.GPUTopology = gpuTopologyFlag
		} else {
			if err := huh.NewInput().
				Title("GPU topology:").
				Description("GPU topology requirement (e.g., 8xH100, 4xA100, leave empty if not applicable)").
				Value(&annotations.GPUTopology).
				Run(); err != nil {
				return err
			}
		}

		// Runtime execution annotations for Story #69
		if runtimeTypeFlag != "" {
			annotations.RuntimeType = runtimeTypeFlag
		} else {
			if err := huh.NewInput().
				Title("Runtime type:").
				Description("Specific runtime for serving (e.g., vllm, kserve, leave empty to use default)").
				Value(&annotations.RuntimeType).
				Run(); err != nil {
				return err
			}
		}

		if layerDedupFlag != "" {
			annotations.LayerDeduplication = layerDedupFlag
		} else {
			if err := huh.NewSelect[string]().
				Title("Layer deduplication:").
				Description("Enable layer deduplication optimization for large models").
				Options(huh.NewOptions("true", "false", "")...).
				Value(&annotations.LayerDeduplication).
				Run(); err != nil {
				return err
			}
		}

		if dlcEndpointFlag != "" {
			annotations.ReferenceSkillDLC = dlcEndpointFlag
		} else if isSkillFlag {
			// Only prompt for DLC endpoint if this is a skill
			if err := huh.NewInput().
				Title("Reference Skill DLC endpoint:").
				Description("Endpoint for dynamic skill loading (e.g., https://dlc.example.com, leave empty if not applicable)").
				Value(&annotations.ReferenceSkillDLC).
				Run(); err != nil {
				return err
			}
		}

		if skillRefsFlag != "" {
			annotations.SkillReferences = skillRefsFlag
		} else if isSkillFlag {
			// Only prompt for skill references if this is a skill
			if err := huh.NewInput().
				Title("Skill references:").
				Description("Comma-separated list of skill references (e.g., skill:sha256:abc,skill:sha256:def)").
				Value(&annotations.SkillReferences).
				Run(); err != nil {
				return err
			}
		}

		// MOF classification
		if mofClassFlag != "" {
			annotations.MOFClass = mofClassFlag
		} else {
			if err := huh.NewSelect[string]().
				Title("MOF Class:").
				Description("Model Openness Framework classification").
				Options(huh.NewOptions("I", "II", "III")...).
				Value(&annotations.MOFClass).
				Run(); err != nil {
				return err
			}
		}

		if mofComponentsFlag != "" {
			annotations.MOFComponents = mofComponentsFlag
		} else {
			if err := huh.NewInput().
				Title("MOF Components:").
				Description("Comma-separated list of MOF components (e.g., weights,training-data,code)").
				Value(&annotations.MOFComponents).
				Run(); err != nil {
				return err
			}
		}

		// Create packaging workflow
		pf, err := workflow.NewPackageWorkflow(cfg.Registry)
		if err != nil {
			return err
		}

		// Set packaging info
		pf.SetPackageInfo(modelName, modelPath, artifactName, registryURL, includeRAG, ragPath)
		pf.SetAnnotations(annotations)
		pf.SetIsSkill(isSkillFlag)

		// Set signing options
		// Determine signer: use flag, then config, then default to empty (will default to sigstore in workflow)
		signerToUse := signerFlag
		if signerToUse == "" {
			signerToUse = cfg.Signer
		}
		pf.SetSigningOptions(signFlag, signerToUse)

		// Set provenance options (enabled by default, can be disabled with --generate-provenance=false)
		pf.SetProvenanceOptions(provenanceFlag)

		artifactType := "model"
		if isSkillFlag {
			artifactType = "skill"
		}
		fmt.Printf("\nPackaging your %s...", artifactType)
		fmt.Println()
		return pf.Run()
	},
}

func init() {
	rootCmd.AddCommand(packageCmd)
	packageCmd.Flags().String("registry", "", "Registry tool: oras or modelpack")
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
	packageCmd.Flags().String("mof-class", "", "MOF Class: I, II, or III")
	packageCmd.Flags().String("mof-components", "", "MOF components (comma-separated)")
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
	packageCmd.Flags().Bool("sign", false, "Sign the artifact automatically after packaging")
	packageCmd.Flags().String("signer", "", "Signing tool: sigstore or notary (default: sigstore)")
	packageCmd.Flags().Bool("generate-provenance", true, "Generate SLSA provenance attestation (default: true)")
}
