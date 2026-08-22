package workflow

import (
	"fmt"
	"os/exec"
)

// --- ArgoCD Provider ---

type ArgoCDProvider struct{}

func (a *ArgoCDProvider) Name() string {
	return "argocd"
}

func (a *ArgoCDProvider) IsInstalled() bool {
	return exec.Command("argocd", "version").Run() == nil
}

func (a *ArgoCDProvider) InstallInstructions() string {
	return "brew install argoproj/tap/argocd"
}

func (a *ArgoCDProvider) Deploy(modelName, repoURL, path string) error {
	cmd := exec.Command("argocd", "app", "create", modelName, "--repo", repoURL, "--path", path, "--dest-namespace", "default")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create ArgoCD application: %v", err)
	}
	fmt.Printf("Created ArgoCD application: %s\n", modelName)
	return nil
}

// --- Flux Provider ---

type FluxProvider struct{}

func (f *FluxProvider) Name() string {
	return "flux"
}

func (f *FluxProvider) IsInstalled() bool {
	return exec.Command("flux", "version").Run() == nil
}

func (f *FluxProvider) InstallInstructions() string {
	return "brew install fluxcd/tap/flux"
}

func (f *FluxProvider) Deploy(modelName, repoURL, path string) error {
	cmd := exec.Command("flux", "create", "source", "git", modelName+"-"+"git", "--url", repoURL, "--branch", "main")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create Flux git source: %v", err)
	}
	cmd = exec.Command("flux", "create", "kustomization", modelName, "--source", modelName+"-"+"git", "--path", path)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create Flux kustomization: %v", err)
	}
	fmt.Printf("Created Flux deployment: %s\n", modelName)
	return nil
}

// GetGitOpsProvider returns the appropriate GitOps provider by name
func GetGitOpsProvider(name string) (GitOpsProvider, error) {
	switch name {
	case "argo", "argocd":
		return &ArgoCDProvider{}, nil
	case "flux":
		return &FluxProvider{}, nil
	default:
		return nil, fmt.Errorf("unknown GitOps provider: %s (supported: argo, flux)", name)
	}
}
