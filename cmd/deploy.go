package cmd

import (
	"fmt"

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
		artifactFlag, _ := cmd.Flags().GetString("artifact")

		// Interactive prompts if not set in config or via flags
		if err := askSelectIfEmpty(cmd, "gitops", &cfg.GitOps, "Select GitOps tool:", "Choose how you want to manage your deployments", []string{"argo", "flux"}); err != nil {
			return err
		}
		if err := askSelectIfEmpty(cmd, "registry", &cfg.Registry, "Select Model Registry:", "Choose how you want to package and distribute your models", []string{"oras", "modelpack"}); err != nil {
			return err
		}

		// Tour guide: Get model information interactively or via flags
		var modelName string
		var hasKubernetes bool
		var repoURL string
		var manifestPath string

		if err := askString(cmd, "model", &modelName, "Model name:", "What would you like to name your model deployment?"); err != nil {
			return err
		}

		if err := askConfirm(cmd, "has-k8s", &hasKubernetes, "Do you have a Kubernetes cluster available?", "This determines if we'll deploy to K8s or just package the model"); err != nil {
			return err
		}

		if cmd.Flags().Changed("repo") || hasKubernetes {
			if err := askString(cmd, "repo", &repoURL, "Git repository URL:", "Where are your Kubernetes manifests stored? (e.g., https://github.com/you/model-manifests)"); err != nil {
				return err
			}
		}

		if cmd.Flags().Changed("path") || (hasKubernetes && repoURL != "") {
			if err := askString(cmd, "path", &manifestPath, "Manifest path:", "Path to your Kubernetes manifests in the repo (e.g., ./manifests or ./k8s)"); err != nil {
				return err
			}
		} else if hasKubernetes {
			fmt.Println("Okay! We'll package the model but skip Kubernetes deployment.")
			fmt.Println("You can run 'model-cli deploy' again when you have a cluster ready.")
		}

		if err := requireValues("model", modelName); err != nil {
			return err
		}

		// All inputs collected: remember the tool choices for next time.
		warnIfSaveFails(config.Save(cfg))

		// Delegate to workflow
		wf, err := workflow.NewDeployWorkflow(cfg.GitOps, cfg.Registry)
		if err != nil {
			return err
		}

		if hasKubernetes && repoURL != "" {
			wf.SetModelInfo(modelName, repoURL, manifestPath)

			// Set artifact reference if provided (contains Trust Profile annotations)
			// This allows GitOps tools to access the annotations for admission (Story #63)
			if artifactFlag != "" {
				wf.SetArtifactRef(artifactFlag)
			} else if modelName != "" {
				// Use model name as artifact reference
				wf.SetArtifactRef(modelName)
			}
		}

		return wf.Run()
	},
}

func init() {
	rootCmd.AddCommand(deployCmd)
	deployCmd.Flags().String("gitops", "", "GitOps tool: argo or flux")
	deployCmd.Flags().String("registry", "", "Registry tool: oras or modelpack")
	deployCmd.Flags().String("model", "", "Model name for deployment")
	deployCmd.Flags().String("artifact", "", "Full artifact reference (e.g., ghcr.io/my-org/my-model:v1) with Trust Profile annotations")
	deployCmd.Flags().String("repo", "", "Git repository URL for Kubernetes manifests")
	deployCmd.Flags().String("path", "", "Path to Kubernetes manifests in repo")
	deployCmd.Flags().Bool("has-k8s", false, "Set to true if you have a Kubernetes cluster available")
}
