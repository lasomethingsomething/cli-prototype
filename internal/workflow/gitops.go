package workflow

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// GitOpsProvider defines the interface for GitOps tools like ArgoCD and Flux
type GitOpsProvider interface {
	Name() string
	IsInstalled() bool
	InstallInstructions() string
	Deploy(modelName, repoURL, path string) error
}

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
	// GitOps: the CLI never touches the cluster directly.
	// It commits the manifest into the Git repo and lets Flux reconcile.

	// 1. Verify the working tree is the target repo
	remote, err := exec.Command("git", "remote", "get-url", "origin").Output()
	if err != nil {
		return fmt.Errorf("not in a git repo: %v", err)
	}
	remoteURL := strings.TrimSpace(string(remote))
	if repoURL != "" && !strings.Contains(remoteURL, strings.TrimSuffix(repoURL, ".git")) &&
		!strings.Contains(repoURL, strings.TrimSuffix(remoteURL, ".git")) {
		return fmt.Errorf("current repo origin (%s) does not match target (%s)", remoteURL, repoURL)
	}

	// 2. Commit the manifest path
	if path == "" {
		return fmt.Errorf("manifest path is empty; nothing to commit")
	}
	if out, err := exec.Command("git", "add", path).CombinedOutput(); err != nil {
		return fmt.Errorf("git add %s failed: %s", path, out)
	}
	commit := exec.Command("git", "commit", "-m", "Deploy model "+modelName+" via wizard")
	commit.Env = append(os.Environ(), "GIT_EDITOR=true")
	if out, err := commit.CombinedOutput(); err != nil {
		// "nothing to commit" is fine - manifest already committed
		if !strings.Contains(string(out), "nothing to commit") &&
			!strings.Contains(string(out), "no changes added") {
			return fmt.Errorf("git commit failed: %s", out)
		}
	}
	if out, err := exec.Command("git", "pull", "--rebase", "--autostash").CombinedOutput(); err != nil {
		return fmt.Errorf("git pull --rebase failed: %s", out)
	}
	if out, err := exec.Command("git", "push").CombinedOutput(); err != nil {
		return fmt.Errorf("git push failed: %s", out)
	}

	// 3. Ask Flux to pick it up now (optional; interval would do it anyway)
	// Reconcile the source which will trigger dependent Kustomizations automatically
	reconcile := exec.Command("flux", "reconcile", "source", "flux-system")
	if err := reconcile.Run(); err != nil {
		// If source reconcile fails, Flux will still pick up changes on its next interval
		fmt.Printf("Pushed to Git. Flux will reconcile on its next interval (couldn't trigger immediately: %v)\n", err)
	} else {
		fmt.Printf("Pushed to Git and reconciled Flux source. Kustomizations will update automatically.\n")
	}
	return nil
}

// Ingredient describes a cluster component that the Git repo provides.
type Ingredient struct {
	Name        string
	Description string
	Present     bool
}

// GetGitOpsProvider returns the GitOps provider for an option name from
// GitOpsOptions. "argo" is an accepted alias for "argocd".
func GetGitOpsProvider(name string) (GitOpsProvider, error) {
	switch name {
	case "argo", "argocd":
		return &ArgoCDProvider{}, nil
	case "flux":
		return &FluxProvider{}, nil
	default:
		return nil, fmt.Errorf("unknown GitOps provider: %s (supported: argocd, flux)", name)
	}
}

// DetectIngredients checks which infrastructure components the cluster
// already runs, so the wizard can select instead of install.
func DetectIngredients() []Ingredient {
	type spec struct {
		name, desc, deployment string
	}
	known := []spec{
		{"kserve", "model serving operator", "kserve-controller-manager"},
		{"cert-manager", "TLS certificates", "cert-manager"},
		{"metrics-server", "resource metrics / HPA", "metrics-server"},
	}

	// Deployments tell us what actually runs, regardless of which
	// Kustomization installed it.
	out, err := exec.Command("kubectl", "get", "deployments", "-A", "-o", "name").CombinedOutput()
	text := strings.ToLower(string(out))

	ings := make([]Ingredient, 0, len(known)+1)
	for _, k := range known {
		ings = append(ings, Ingredient{
			Name:        k.name,
			Description: k.desc,
			Present:     err == nil && strings.Contains(text, k.deployment),
		})
	}

	// Flux itself: parse 'flux get kustomization' rows and require Ready=True.
	fluxOut, ferr := exec.Command("flux", "get", "kustomization", "--all-namespaces").CombinedOutput()
	fluxPresent := false
	if ferr == nil {
		for _, line := range strings.Split(string(fluxOut), "\n") {
			fields := strings.Fields(line)
			if len(fields) >= 5 && fields[1] == "flux-system" && fields[4] == "True" {
				fluxPresent = true
			}
		}
	}
	ings = append(ings, Ingredient{Name: "flux", Description: "GitOps agent", Present: fluxPresent})
	return ings
}
