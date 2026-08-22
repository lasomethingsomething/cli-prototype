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
	Short: "Package a model as an OCI artifact",
	Long: `Package your model as an OCI artifact for distribution and deployment.

This command helps you package models, prompts, and RAG context into a single
OCI artifact that can be stored in registries and deployed anywhere.

Examples:
  model-cli package
  model-cli package --model phi-4-mini --registry oras --output my-model:latest`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Load()

		// Interactive prompts if not set in config
		if cfg.Registry == "" {
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
		}

		// Save config for future use
		config.Save(cfg)

		// Tour guide: Get packaging information
		var modelName string
		if err := huh.NewInput().
			Title("Model name:").
			Description("What is the name of your model? (e.g., phi-4-mini)").
			Value(&modelName).
			Run(); err != nil {
			return err
		}

		var modelPath string
		if err := huh.NewInput().
			Title("Model path:").
			Description("Path to your model files or directory").
			Value(&modelPath).
			Run(); err != nil {
			return err
		}

		var artifactName string
		if err := huh.NewInput().
			Title("Artifact name:").
			Description("What would you like to name the OCI artifact? (e.g., my-org/my-model:latest)").
			Value(&artifactName).
			Run(); err != nil {
			return err
		}

		var registryURL string
		if err := huh.NewInput().
			Title("Registry URL:").
			Description("Where should we push the artifact? (e.g., ghcr.io or leave empty for local)").
			Value(&registryURL).
			Run(); err != nil {
			return err
		}

		var includeRAG bool
		if err := huh.NewConfirm().
			Title("Include RAG context?").
			Description("Do you want to package RAG (Retrieval-Augmented Generation) context with your model?").
			Value(&includeRAG).
			Run(); err != nil {
			return err
		}

		var ragPath string
		if includeRAG {
			if err := huh.NewInput().
				Title("RAG context path:").
				Description("Path to your RAG context files").
				Value(&ragPath).
				Run(); err != nil {
				return err
			}
		}

		// Collect CNCF AI Interoperability Profile annotations
		annotations := workflow.NewAnnotationSet()

		// Runtime annotations
		if err := huh.NewInput().
			Title("Runtime:").
			Description("Serving runtime (e.g., vllm, kserve)").
			Value(&annotations.Runtime).
			Run(); err != nil {
			return err
		}

		if err := huh.NewSelect[string]().
			Title("Accelerator:").
			Description("Hardware accelerator requirement").
			Options(huh.NewOptions("nvidia-gpu", "amd-gpu", "intel-gpu", "cpu", "none")...).
			Value(&annotations.Accelerator).
			Run(); err != nil {
			return err
		}

		if err := huh.NewInput().
			Title("Minimum CUDA version:").
			Description("Minimum CUDA version required (e.g., 12.1, leave empty if not applicable)").
			Value(&annotations.CUDAMin).
			Run(); err != nil {
			return err
		}

		if err := huh.NewInput().
			Title("Minimum memory:").
			Description("Minimum memory required (e.g., 24GiB)").
			Value(&annotations.MemoryMin).
			Run(); err != nil {
			return err
		}

		// MOF classification
		if err := huh.NewSelect[string]().
			Title("MOF Class:").
			Description("Model Openness Framework classification").
			Options(huh.NewOptions("I", "II", "III")...).
			Value(&annotations.MOFClass).
			Run(); err != nil {
			return err
		}

		if err := huh.NewInput().
			Title("MOF Components:").
			Description("Comma-separated list of MOF components (e.g., weights,training-data,code)").
			Value(&annotations.MOFComponents).
			Run(); err != nil {
			return err
		}

		// Create packaging workflow
		pf, err := workflow.NewPackageWorkflow(cfg.Registry)
		if err != nil {
			return err
		}

		// Set packaging info
		pf.SetPackageInfo(modelName, modelPath, artifactName, registryURL, includeRAG, ragPath)
		pf.SetAnnotations(annotations)

		fmt.Println("\nPackaging your model...")
		return pf.Run()
	},
}

func init() {
	rootCmd.AddCommand(packageCmd)
}
