package workflow

import (
    "fmt"
)

// DeployWorkflow orchestrates the deployment using pluggable providers
type DeployWorkflow struct {
    gitOps      string
    registry    string
    gitOpsProvider GitOpsProvider
    registryProvider RegistryProvider
    modelName   string
    repoURL     string
    manifestPath string
}

// NewDeployWorkflow creates a new deployment workflow with the given providers
func NewDeployWorkflow(gitOps, registry string) (*DeployWorkflow, error) {
    gitOpsProvider, err := GetGitOpsProvider(gitOps)
    if err != nil {
        return nil, err
    }
    
    registryProvider, err := GetRegistryProvider(registry)
    if err != nil {
        return nil, err
    }
    
    return &DeployWorkflow{
        gitOps:           gitOps,
        registry:        registry,
        gitOpsProvider:  gitOpsProvider,
        registryProvider: registryProvider,
    }, nil
}

// SetModelInfo sets the model deployment details
func (w *DeployWorkflow) SetModelInfo(modelName, repoURL, manifestPath string) {
    w.modelName = modelName
    w.repoURL = repoURL
    w.manifestPath = manifestPath
}

// Run executes the deployment workflow
func (w *DeployWorkflow) Run() error {
	if w.modelName == "" || w.repoURL == "" {
		return fmt.Errorf("model name and repository URL are required: call SetModelInfo before Run")
	}

	fmt.Printf("Starting deployment with GitOps: %s, Registry: %s\n", w.gitOps, w.registry)
    
    // Check if GitOps provider is available
    if !w.gitOpsProvider.IsInstalled() {
        return fmt.Errorf("%s not installed. Install with: %s", w.gitOpsProvider.Name(), w.gitOpsProvider.InstallInstructions())
    }
    
    // Check if registry provider is available
    if !w.registryProvider.IsInstalled() {
        return fmt.Errorf("%s not installed. Install with: %s", w.registryProvider.Name(), w.registryProvider.InstallInstructions())
    }
    
	// Deploy using GitOps provider
	fmt.Printf("Deploying model '%s' from repository '%s' with %s...\n",
		w.modelName, w.repoURL, w.gitOps)
	if err := w.gitOpsProvider.Deploy(w.modelName, w.repoURL, w.manifestPath); err != nil {
		return err
	}
    
    // Use registry provider
    if w.registry == "oras" || w.registry == "modelpack" {
        fmt.Printf("Using %s for registry operations\n", w.registry)
        // For now, just indicate the provider is ready
        // Actual push/pull will be added when we integrate with packaging commands
    }
    
    fmt.Println("Deployment workflow complete!")
    return nil
}