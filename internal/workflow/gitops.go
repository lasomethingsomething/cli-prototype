package workflow

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// GitOpsProvider defines the interface for GitOps tools like ArgoCD and Flux
type GitOpsProvider interface {
	Name() string
	IsInstalled() bool
	InstallInstructions() string
	Deploy(modelName, repoURL, path, modelPath string) error
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

func (a *ArgoCDProvider) Deploy(modelName, repoURL, path, modelPath string) error {
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

func (f *FluxProvider) Deploy(modelName, repoURL, path, modelPath string) error {
	// GitOps: the CLI never touches the cluster directly.
	// It commits the manifest into the Git repo and lets Flux reconcile.

	// 1. Verify the working tree is the target repo
	remote, err := exec.Command("git", "remote", "get-url", "origin").Output()
	if err != nil {
		return fmt.Errorf("not in a git repo: %v", err)
	}
	remoteURL := strings.TrimSpace(string(remote))
	
	// Normalize URLs for comparison
	if repoURL != "" {
		if !URLsAreEqual(remoteURL, repoURL) {
			return fmt.Errorf("current repo origin (%s) does not match target (%s)", remoteURL, repoURL)
		}
	}

	// 2. Generate and write the InferenceService manifest
	// Get current branch
	branch, err := GetCurrentGitBranch()
	if err != nil {
		fmt.Printf("Warning: could not determine git branch: %v, defaulting to 'main'\n", err)
		branch = "main"
	}

	// Create inference service config and generate manifest
	inferenceConfig, err := CreateInferenceServiceConfig(
		modelName,
		modelPath,
		repoURL,
		branch,
	)
	if err != nil {
		return fmt.Errorf("failed to create inference service config: %v", err)
	}

	// Write manifest to the specified path
	manifestPath := filepath.Join(path, modelName+".yaml")
	if err := WriteInferenceServiceManifest(inferenceConfig, manifestPath); err != nil {
		return fmt.Errorf("failed to write inference service manifest: %v", err)
	}
	fmt.Printf("✓ InferenceService manifest generated: %s\n", manifestPath)

	// 3. Commit the manifest path
	if path == "" {
		return fmt.Errorf("manifest path is empty; nothing to commit")
	}
	if out, err := exec.Command("git", "add", path).CombinedOutput(); err != nil {
		return fmt.Errorf("git add %s failed: %s", path, out)
	}
	commit := exec.Command("git", "commit", "-m", "Deploy model "+modelName+" via wizard")
	commit.Env = append(os.Environ(), "GIT_EDITOR=true")
	commitOut, err := commit.CombinedOutput()
	if err != nil {
		// "nothing to commit" is fine - manifest already committed
		outStr := string(commitOut)
		if !strings.Contains(outStr, "nothing to commit") &&
			!strings.Contains(outStr, "no changes added") {
			return fmt.Errorf("git commit failed: %s", outStr)
		}
	}
	
	// Check if commit actually created a new commit
	hasNewCommit := !strings.Contains(string(commitOut), "nothing to commit") &&
		!strings.Contains(string(commitOut), "no changes added")
	
	if hasNewCommit {
		fmt.Printf("✓ Committed manifest to git\n")
	} else {
		fmt.Printf("⚠ Manifest already committed (no new commit)\n")
	}

	// 4. Push to remote
	pushOut, err := exec.Command("git", "push").CombinedOutput()
	if err != nil {
		return fmt.Errorf("git push failed: %s", pushOut)
	}
	if hasNewCommit {
		fmt.Printf("✓ Pushed to git repository\n")
	} else {
		fmt.Printf("⚠ Already up to date, nothing to push\n")
	}

	// 5. Verify push was successful by checking exit code (already done above)
	// We already checked push exit code - if we're here, push succeeded

	// 6. Trigger Flux reconciliation
	// Only reconcile if there was a new commit
	if hasNewCommit {
		// First, reconcile the source
		reconcileSource := exec.Command("flux", "reconcile", "source", "flux-system")
		reconcileOut, err := reconcileSource.CombinedOutput()
		if err != nil {
			fmt.Printf("Warning: failed to reconcile Flux source: %v\n%s\n", err, reconcileOut)
			// Continue - Flux will still pick up changes on its next interval
		} else {
			fmt.Printf("✓ Flux source reconciled\n")
			// Check for "applied revision" in output
			if strings.Contains(string(reconcileOut), "applied revision") {
				fmt.Printf("  ✓ New revision applied\n")
			}
		}

		// 7. Reconcile the kustomization for test-model namespace
		reconcileKustomization := exec.Command("flux", "reconcile", "kustomization", "test-model", "--with-source")
		kustOut, err := reconcileKustomization.CombinedOutput()
		if err != nil {
			fmt.Printf("Warning: failed to reconcile kustomization: %v\n%s\n", err, kustOut)
			// Continue - Flux will still reconcile on its next interval
		} else {
			fmt.Printf("✓ Flux kustomization reconciled\n")
			if strings.Contains(string(kustOut), "applied revision") {
				fmt.Printf("  ✓ New revision applied to kustomization\n")
			}
		}
	} else {
		fmt.Printf("⚠ Skipping Flux reconciliation (no new commit to deploy)\n")
	}

	// 8. Poll for InferenceService to reach READY=True
	fmt.Printf("Waiting for InferenceService to become ready...\n")
	if err := WaitForInferenceServiceReady(modelName, inferenceConfig.Namespace); err != nil {
		// Return error with details
		return fmt.Errorf("InferenceService did not reach Ready state: %v", err)
	}
	fmt.Printf("✓ InferenceService '%s' in namespace '%s' is Ready\n", modelName, inferenceConfig.Namespace)

	return nil
}

// WaitForInferenceServiceReady polls kubectl until the InferenceService is ready or timeout
func WaitForInferenceServiceReady(name, namespace string) error {
	const maxAttempts = 30
	const waitInterval = 5 * time.Second
	
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		cmd := exec.Command("kubectl", "get", "inferenceservice", name, "-n", namespace, "-o", "jsonpath={.status.conditions[?(@.type==\"Ready\")].status}")
		output, err := cmd.Output()
		if err != nil {
			// Check if it's just not found yet
			// If error is exit code 1, it might just not be created yet
			fmt.Printf("  Attempt %d/%d: InferenceService not found yet...\n", attempt, maxAttempts)
		} else {
			status := strings.TrimSpace(string(output))
			if status == "True" {
				return nil
			}
			fmt.Printf("  Attempt %d/%d: InferenceService status: %s\n", attempt, maxAttempts, status)
		}
		
		// Wait before next attempt
		if attempt < maxAttempts {
			time.Sleep(waitInterval)
		}
	}
	
	return fmt.Errorf("timeout waiting for InferenceService '%s' in namespace '%s' to reach Ready", name, namespace)
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
