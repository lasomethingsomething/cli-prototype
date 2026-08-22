package cmd

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/lasomethingsomething/cli-prototype/config"
	"github.com/lasomethingsomething/cli-prototype/internal/workflow"
	"github.com/spf13/cobra"
)

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploy a model using GitOps",
	Long: `Deploy a model to your cluster using GitOps tools (Argo or Flux) and registry tools (ORAS or ModelPack).

This command guides you through the deployment process with interactive prompts.
Example:
  model-cli deploy
  model-cli deploy --gitops argo --registry oras --model my-model --repo https://github.com/you/model-manifests`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Load()

		// Interactive prompts if not set in config or via flags
		if cfg.GitOps == "" {
			var gitOpsTool string
			if err := huh.NewSelect[string]().
				Title("Select GitOps tool:").
				Description("Choose how you want to manage your deployments").
				Options(huh.NewOptions("argo", "flux")...).
				Value(&gitOpsTool).
				Run(); err != nil {
				return err
			}
			cfg.GitOps = gitOpsTool
		}

		if cfg.Registry == "" {
			var registryTool string
			if err := huh.NewSelect[string]().
				Title("Select Model Registry:").
				Description("Choose how you want to package and distribute your models").
				Options(huh.NewOptions("oras", "modelpack")...).
				Value(&registryTool).
				Run(); err != nil {
				return err
			}
			cfg.Registry = registryTool
		}

		// Save config for future use
		config.Save(cfg)

		// Tour guide: Get model information interactively
		var modelName string
		if err := huh.NewInput().
			Title("Model name:").
			Description("What would you like to name your model deployment?").
			Value(&modelName).
			Run(); err != nil {
			return err
		}

		var hasKubernetes bool
		if err := huh.NewConfirm().
			Title("Do you have a Kubernetes cluster available?").
			Description("This determines if we'll deploy to K8s or just package the model").
			Value(&hasKubernetes).
			Run(); err != nil {
			return err
		}

		var repoURL string
		var manifestPath string
		
		if hasKubernetes {
			if err := huh.NewInput().
				Title("Git repository URL:").
				Description("Where are your Kubernetes manifests stored? (e.g., https://github.com/you/model-manifests)").
				Value(&repoURL).
				Run(); err != nil {
				return err
			}

			if err := huh.NewInput().
				Title("Manifest path:").
				Description("Path to your Kubernetes manifests in the repo (e.g., ./manifests or ./k8s)").
				Value(&manifestPath).
				Run(); err != nil {
				return err
			}
		} else {
			fmt.Println("Okay! We'll package the model but skip Kubernetes deployment.")
			fmt.Println("You can run 'model-cli deploy' again when you have a cluster ready.")
		}

		// Delegate to workflow
		wf, err := workflow.NewDeployWorkflow(cfg.GitOps, cfg.Registry)
		if err != nil {
			return err
		}
		
		if hasKubernetes && repoURL != "" {
			wf.SetModelInfo(modelName, repoURL, manifestPath)
		}
		
		return wf.Run()
	},
}

func init() {
	rootCmd.AddCommand(deployCmd)
}
