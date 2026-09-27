package workflow

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// DeployResult contains the result of a GitOps deployment
type DeployResult struct {
	Error      error
	Deployed   bool // True if a new commit was created and pushed
	Ready      bool // True if InferenceService reached Ready
}

// GitOpsProvider defines the interface for GitOps tools like ArgoCD and Flux
type GitOpsProvider interface {
	Name() string
	IsInstalled() bool
	InstallInstructions() string
	Deploy(modelName, repoURL, path, modelPath string) DeployResult
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

func (a *ArgoCDProvider) Deploy(modelName, repoURL, path, modelPath string) DeployResult {
	cmd := exec.Command("argocd", "app", "create", modelName, "--repo", repoURL, "--path", path, "--dest-namespace", "default")
	if err := cmd.Run(); err != nil {
		return DeployResult{Error: fmt.Errorf("failed to create ArgoCD application: %v", err)}
	}
	fmt.Printf("Created ArgoCD application: %s\n", modelName)
	// ArgoCD deploy doesn't have the same no-op detection as Flux, so assume deployed
	return DeployResult{Error: nil, Deployed: true, Ready: false}
}

// --- Flux Provider ---

type FluxProvider struct{}

func (f *FluxProvider) Name() string {
	return "flux"
}

func (f *FluxProvider) IsInstalled() bool {
	_, err := exec.LookPath("flux")
	return err == nil
}

func (f *FluxProvider) InstallInstructions() string {
	return "brew install fluxcd/tap/flux"
}

func (f *FluxProvider) Deploy(modelName, repoURL, path, modelPath string) DeployResult {
	// GitOps: the CLI never touches the cluster directly.
	// It commits the manifest into the Git repo and lets Flux reconcile.

	// 1. Verify the working tree is the target repo
	remote, err := exec.Command("git", "remote", "get-url", "origin").Output()
	if err != nil {
		return DeployResult{Error: fmt.Errorf("not in a git repo: %v", err)}
	}
	remoteURL := strings.TrimSpace(string(remote))
	
	// Normalize URLs for comparison
	if repoURL != "" {
		if !URLsAreEqual(remoteURL, repoURL) {
			return DeployResult{Error: fmt.Errorf("current repo origin (%s) does not match target (%s)", remoteURL, repoURL)}
		}
	}

	// Pre-flight: verify the git remote and working tree state
	fmt.Println("Pre-flight checks:")
	
	// Get current branch for pre-flight checks
	branch, err := GetCurrentGitBranch()
	if err != nil {
		branch = "main"
	}

	// Check 1: Verify user can push to origin with dry-run
	fmt.Print("  - Verifying push permissions... ")
	if err := verifyGitPushable(remoteURL, branch); err != nil {
		return DeployResult{Error: err}
	}
	fmt.Println("✓")

	// Check 2: Verify working tree is clean
	fmt.Print("  - Verifying clean working tree... ")
	if err := verifyGitWorkingTreeClean(); err != nil {
		return DeployResult{Error: err}
	}
	fmt.Println("✓")

	// Check 3: Verify local branch is not behind origin
	fmt.Print("  - Verifying branch is not behind origin... ")
	if err := verifyGitNotBehindOrigin(remoteURL, branch); err != nil {
		return DeployResult{Error: err}
	}
	fmt.Println("✓")
	fmt.Println()

	// Check cluster reachability (Flux needs a working cluster to reconcile)
	// Use kubectl as a simple reachability probe
	kubectlCmd := exec.Command("kubectl", "get", "nodes", "-o", "name")
	if err := kubectlCmd.Run(); err != nil {
		return DeployResult{Error: fmt.Errorf("cluster unreachable — is minikube running? (flux installed, cluster unreachable)")}
	}

	// 2. Generate and write the InferenceService manifest
	// branch was already retrieved in pre-flight checks

	// Create inference service config and generate manifest
	inferenceConfig, err := CreateInferenceServiceConfig(
		modelName,
		modelPath,
		repoURL,
		branch,
	)
	if err != nil {
		return DeployResult{Error: fmt.Errorf("failed to create inference service config: %v", err)}
	}

	// Write manifest to the specified path
	manifestPath := filepath.Join(path, modelName+".yaml")
	if err := WriteInferenceServiceManifest(inferenceConfig, manifestPath); err != nil {
		return DeployResult{Error: fmt.Errorf("failed to write inference service manifest: %v", err)}
	}
	fmt.Printf("✓ InferenceService manifest generated: %s\n", manifestPath)

	// 3. Commit the manifest path
	if path == "" {
		return DeployResult{Error: fmt.Errorf("manifest path is empty; nothing to commit")}
	}
	if out, err := exec.Command("git", "add", path).CombinedOutput(); err != nil {
		return DeployResult{Error: fmt.Errorf("git add %s failed: %s", path, out)}
	}
	commit := exec.Command("git", "commit", "-m", "Deploy model "+modelName+" via wizard")
	commit.Env = append(os.Environ(), "GIT_EDITOR=true")
	commitOut, err := commit.CombinedOutput()
	if err != nil {
		// "nothing to commit" is fine - manifest already committed
		outStr := string(commitOut)
		if !strings.Contains(outStr, "nothing to commit") &&
			!strings.Contains(outStr, "no changes added") {
			return DeployResult{Error: fmt.Errorf("git commit failed: %s", outStr)}
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
		return DeployResult{Error: fmt.Errorf("git push failed: %s", pushOut)}
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

		// 7. Reconcile the kustomization for models namespace
		// The kustomization "models" is in the flux-system namespace and watches clusters/minikube/apps/
		reconcileKustomization := exec.Command("flux", "reconcile", "kustomization", "models", "-n", "flux-system", "--with-source")
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
	ready := true
	if err := WaitForInferenceServiceReady(modelName, inferenceConfig.Namespace); err != nil {
		// Don't return error - deployment may have happened but Ready not reached yet
		fmt.Printf("⚠ InferenceService did not reach Ready state: %v\n", err)
		ready = false
	} else {
		fmt.Printf("✓ InferenceService '%s' in namespace '%s' is Ready\n", modelName, inferenceConfig.Namespace)
	}

	return DeployResult{
		Error:    nil,
		Deployed: hasNewCommit,
		Ready:    ready,
	}
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

	// Flux itself: first check if binary is installed (on PATH)
	// This distinguishes between "flux not installed" vs "flux installed but cluster unreachable"
	_, fluxBinaryErr := exec.LookPath("flux")
	fluxPresent := false
	
	// If flux binary is not installed, don't try to query the cluster
	if fluxBinaryErr == nil {
		// Flux binary is installed, now check if it's actually running in the cluster
		fluxOut, ferr := exec.Command("flux", "get", "kustomization", "--all-namespaces").CombinedOutput()
		if ferr == nil {
			for _, line := range strings.Split(string(fluxOut), "\n") {
				fields := strings.Fields(line)
				if len(fields) >= 5 && fields[1] == "flux-system" && fields[4] == "True" {
					fluxPresent = true
				}
			}
		}
		// If flux binary is installed but cluster query failed, flux is "installed but cluster unreachable"
		// We still mark it as not Present in the cluster, but the binary is on PATH
	}
	// If flux binary is not on PATH, fluxPresent remains false
	
	ings = append(ings, Ingredient{Name: "flux", Description: "GitOps agent", Present: fluxPresent})
	return ings
}

// verifyGitPushable performs a dry-run push to verify the user can push to the remote
func verifyGitPushable(remoteURL, branch string) error {
	// Normalize the remote URL for display
	displayURL := remoteURL
	if strings.HasPrefix(displayURL, "git@") {
		displayURL = strings.Replace(displayURL, "git@", "ssh://git@", 1)
	}

	// Try dry-run push to check permissions
	dryRunCmd := exec.Command("git", "push", "--dry-run", "origin", branch)
	output, err := dryRunCmd.CombinedOutput()
	if err != nil {
		outputStr := string(output)
		// Check for common error messages
		if strings.Contains(outputStr, "permission denied") ||
			strings.Contains(outputStr, "authentication failed") ||
			strings.Contains(outputStr, "no push access") {
			// Provide actionable hints based on the error
			// If it looks like an auth issue, point to SSH key check
			if strings.Contains(outputStr, "authentication failed") || strings.Contains(outputStr, "Permission denied (publickey)") {
				return fmt.Errorf("no push permission to %s. Check SSH keys with: model-cli doctor. Git error: %s", displayURL, outputStr)
			}
			// For permission denied, suggest it might be an upstream repo that needs forking
			return fmt.Errorf("cannot push to %s. If this is an upstream repository, fork it first: gh repo fork --remote, or manually fork and add your fork as 'origin'. Check SSH keys with: model-cli doctor. Git error: %s", displayURL, outputStr)
		}
		return fmt.Errorf("git push dry-run failed for %s: %s", displayURL, outputStr)
	}
	return nil
}


// verifyGitWorkingTreeClean checks if the working tree is clean
func verifyGitWorkingTreeClean() error {
	// Check for uncommitted changes
	statusCmd := exec.Command("git", "status", "--porcelain")
	output, err := statusCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to check git status: %v", err)
	}
	
	// If there's any output, the working tree is not clean
	if len(output) > 0 {
		// Parse the output to get file names
		lines := strings.Split(strings.TrimSpace(string(output)), "\n")
		var dirtyFiles []string
		for _, line := range lines {
			if len(line) > 0 {
				dirtyFiles = append(dirtyFiles, line)
			}
		}
		if len(dirtyFiles) > 0 {
			return fmt.Errorf("working tree is not clean. Commit or stash changes first:\n  %s", strings.Join(dirtyFiles, "\n  "))
		}
	}
	return nil
}

// verifyGitNotBehindOrigin checks if the local branch is behind origin
func verifyGitNotBehindOrigin(remoteURL, branch string) error {
	// Fetch from origin first
	fetchCmd := exec.Command("git", "fetch", "origin")
	if err := fetchCmd.Run(); err != nil {
		return fmt.Errorf("failed to fetch from origin: %v", err)
	}
	
	// Check if local branch is behind origin
	// git rev-list HEAD..origin/branch --count will return >0 if behind
	localBranch := branch
	if localBranch == "" {
		localBranch = "HEAD"
	}
	
	// Use merge-base to check if we're behind
	mergeBaseCmd := exec.Command("git", "merge-base", localBranch, "origin/"+branch)
	mergeBaseOut, err := mergeBaseCmd.Output()
	if err != nil {
		// If origin/branch doesn't exist, try without origin/
		mergeBaseCmd = exec.Command("git", "merge-base", localBranch, branch)
		mergeBaseOut, err = mergeBaseCmd.Output()
		if err != nil {
			return fmt.Errorf("failed to determine branch relationship: %v", err)
		}
	}
	
	// Check if HEAD is an ancestor of origin/branch (meaning we're behind)
	// git rev-list mergeBase..origin/branch --count
	originRef := "origin/" + branch
	revListCmd := exec.Command("git", "rev-list", string(mergeBaseOut)+".."+originRef, "--count")
	countOut, err := revListCmd.Output()
	if err != nil {
		// Try without origin/ prefix
		revListCmd = exec.Command("git", "rev-list", string(mergeBaseOut)+".."+branch, "--count")
		countOut, err = revListCmd.Output()
		if err != nil {
			// If we can't determine, assume it's ok (might be first push)
			return nil
		}
	}
	
	count := strings.TrimSpace(string(countOut))
	if count != "0" {
		return fmt.Errorf("local branch is behind origin/%s. Run 'git pull' first to fast-forward", branch)
	}
	
	return nil
}
