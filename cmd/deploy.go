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

Examples:
  model-cli deploy
  model-cli deploy --gitops argo --registry oras --model my-model --repo https://github.com/you/model-manifests
  model-cli deploy --gitops flux --registry modelpack --model phi-4-mini --repo https://github.com/org/manifests --path ./k8s`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Load()

		// Get flags
		gitOpsFlag, _ := cmd.Flags().GetString("gitops")
		registryFlag, _ := cmd.Flags().GetString("registry")
		modelNameFlag, _ := cmd.Flags().GetString("model")
		repoURLFlag, _ := cmd.Flags().GetString("repo")
		manifestPathFlag, _ := cmd.Flags().GetString("path")
		hasKubernetesFlag, _ := cmd.Flags().GetBool("has-k8s")

		// Interactive prompts if not set in config or via flags
		if cfg.GitOps == "" && gitOpsFlag == "" {
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
		} else if gitOpsFlag != "" {
			cfg.GitOps = gitOpsFlag
		}

		if cfg.Registry == "" && registryFlag == "" {
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
		} else if registryFlag != "" {
			cfg.Registry = registryFlag
		}

		// Save config for future use
		config.Save(cfg)

		// Tour guide: Get model information interactively or via flags
		var modelName string
		var hasKubernetes bool
		var repoURL string
		var manifestPath string

		if modelNameFlag != "" {
			modelName = modelNameFlag
		} else {
			if err := huh.NewInput().
				Title("Model name:").
				Description("What would you like to name your model deployment?").
				Value(&modelName).
				Run(); err != nil {
				return err
			}
		}

		if hasKubernetesFlag {
			hasKubernetes = true
		} else {
			if err := huh.NewConfirm().
				Title("Do you have a Kubernetes cluster available?").
				Description("This determines if we'll deploy to K8s or just package the model").
				Value(&hasKubernetes).
				Run(); err != nil {
				return err
			}
		}

		if repoURLFlag != "" {
			repoURL = repoURLFlag
		} else if hasKubernetes {
			if err := huh.NewInput().
				Title("Git repository URL:").
				Description("Where are your Kubernetes manifests stored? (e.g., https://github.com/you/model-manifests)").
				Value(&repoURL).
				Run(); err != nil {
				return err
			}
		}

		if manifestPathFlag != "" {
			manifestPath = manifestPathFlag
		} else if hasKubernetes && repoURL != "" {
			if err := huh.NewInput().
				Title("Manifest path:").
				Description("Path to your Kubernetes manifests in the repo (e.g., ./manifests or ./k8s)").
				Value(&manifestPath).
				Run(); err != nil {
				return err
			}
		} else if hasKubernetes {
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
	deployCmd.Flags().String("gitops", "", "GitOps tool: argo or flux")
	deployCmd.Flags().String("registry", "", "Registry tool: oras or modelpack")
	deployCmd.Flags().String("model", "", "Model name for deployment")
	deployCmd.Flags().String("repo", "", "Git repository URL for Kubernetes manifests")
	deployCmd.Flags().String("path", "", "Path to Kubernetes manifests in repo")
	deployCmd.Flags().Bool("has-k8s", false, "Set to true if you have a Kubernetes cluster available")
}
