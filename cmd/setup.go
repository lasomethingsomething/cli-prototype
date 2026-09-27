package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/lasomethingsomething/cli-prototype/internal/workflow"
	"github.com/spf13/cobra"
)

// PinnedKubernetesVersion is the version of Kubernetes that works reliably with minikube
// on this environment. v1.37.0 was observed to fail binding apiserver port 8443.
const PinnedKubernetesVersion = "v1.33.0"

// Phase tracking for resume/report
var setupPhases = []string{"tools", "registry", "cluster", "bootstrap", "converge", "finish"}

// setupCmd handles the setup command
var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Set up everything needed for model-cli: tools, registry, cluster, and Flux",
	Long: `The setup command orchestrates the complete setup from a fresh Mac with Homebrew
to a flux-managed minikube cluster ready for the wizard. It:
  1. Installs all required brew tools (oras, syft, cosign, flux, kubectl, minikube, podman)
  2. Starts a local container registry at localhost:5000
  3. Starts a minikube cluster with pinned Kubernetes version
  4. Bootstraps Flux with token auth (preferred) or SSH
  5. Waits for all kustomizations to converge
  6. Prints warnings if Flux bootstrap created a commit

This is the recommended way to set up for the Test Drive.

Examples:
  model-cli setup              # Interactive: prompt for each step
  model-cli setup --yes        # Non-interactive: proceed with defaults`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Track completed phases
		completed := make(map[string]bool)
		
		// Get flags
		yesFlag, _ := cmd.Flags().GetBool("yes")
		interactiveMode := interactive() || !yesFlag
		
		// Phase 1: Tools
		fmt.Println(titleStyle.Render("Phase 1: Installing required tools"))
		fmt.Println()
		
		if err := installRequiredTools(interactiveMode); err != nil {
			return reportFailure("tools", err, completed)
		}
		completed["tools"] = true
		fmt.Println("✓ Phase 1: Tools installed")
		fmt.Println()

		// Phase 2: Registry
		fmt.Println(titleStyle.Render("Phase 2: Local registry"))
		fmt.Println()
		
		if err := ensureRegistry(interactiveMode); err != nil {
			return reportFailure("registry", err, completed)
		}
		completed["registry"] = true
		fmt.Println("✓ Phase 2: Registry ready")
		fmt.Println()

		// Phase 3: Cluster
		fmt.Println(titleStyle.Render("Phase 3: Kubernetes cluster"))
		fmt.Println()
		
		if err := ensureCluster(interactiveMode); err != nil {
			return reportFailure("cluster", err, completed)
		}
		completed["cluster"] = true
		fmt.Println("✓ Phase 3: Cluster ready")
		fmt.Println()

		// Phase 4: Bootstrap
		fmt.Println(titleStyle.Render("Phase 4: Flux bootstrap"))
		fmt.Println()
		
		bootstrapCommit := false
		if err := bootstrapFlux(interactiveMode, &bootstrapCommit); err != nil {
			return reportFailure("bootstrap", err, completed)
		}
		completed["bootstrap"] = true
		fmt.Println("✓ Phase 4: Flux bootstrapped")
		fmt.Println()

		// Phase 5: Converge
		fmt.Println(titleStyle.Render("Phase 5: Waiting for convergence"))
		fmt.Println()
		
		if err := waitForConvergence(); err != nil {
			return reportFailure("converge", err, completed)
		}
		completed["converge"] = true
		fmt.Println("✓ Phase 5: All kustomizations converged")
		fmt.Println()

		// Phase 6: Finish
		fmt.Println(titleStyle.Render("Phase 6: Final checks"))
		fmt.Println()
		
		if bootstrapCommit {
			fmt.Println("⚠ Flux bootstrap created a commit on origin/main")
			fmt.Println("  Run 'git pull' to update your local clone before running the wizard")
			fmt.Println()
		}
		
		fmt.Println("✓ Setup complete - ready for model-cli wizard")
		completed["finish"] = true

		return nil
	},
}

// installRequiredTools installs all required brew tools
func installRequiredTools(interactiveMode bool) error {
	// Tools that are required for setup
	requiredTools := []string{"oras", "syft", "cosign", "flux", "kubectl", "minikube", "podman"}
	
	fmt.Println("Checking required tools...")
	
	for _, toolName := range requiredTools {
		fmt.Printf("  - %s... ", toolName)
		
		tool, err := workflow.GetTool(toolName)
		if err != nil {
			fmt.Println("⚠ unknown tool")
			continue
		}
		
		if tool.IsInstalled() {
			fmt.Println("✓")
			continue
		}
		
		// Tool not installed - offer to install
		if interactiveMode {
			var install bool
			if err := huh.NewConfirm().
				Title(fmt.Sprintf("Install %s?", toolName)).
				Description(tool.InstallInstructions()).
				Value(&install).
				Run(); err != nil {
				return fmt.Errorf("failed to prompt for %s: %v", toolName, err)
			}
			
			if !install {
				return fmt.Errorf("user declined to install %s", toolName)
			}
			
			fmt.Printf("    Installing %s...\n", toolName)
			result, err := workflow.InstallTool(tool)
			if err != nil {
				return fmt.Errorf("failed to install %s: %v\n%s", toolName, err, result.Stderr)
			}
			fmt.Println("    ✓")
		} else {
			// Non-interactive mode - just report missing
			fmt.Println("✗ not installed")
			return fmt.Errorf("%s not installed and --non-interactive set", toolName)
		}
	}
	
	return nil
}

// ensureRegistry ensures a local registry is running
func ensureRegistry(interactiveMode bool) error {
	fmt.Println("Checking local registry (localhost:5000)...")
	
	// Try to connect to localhost:5000
	cmd := exec.Command("curl", "-s", "-o", "/dev/null", "http://localhost:5000/v2/")
	if err := cmd.Run(); err == nil {
		fmt.Println("  ✓ Local registry is serving")
		return nil
	}
	
	// Check if podman is installed
	podmanTool, err := workflow.GetTool("podman")
	if err != nil {
		return fmt.Errorf("podman tool not found")
	}
	
	if !podmanTool.IsInstalled() {
		// Try to install podman
		fmt.Println("  - Podman required for local registry")
		if interactiveMode {
			var install bool
			if err := huh.NewConfirm().
				Title("Install podman?").
				Value(&install).
				Run(); err != nil {
				return err
			}
			if !install {
				return fmt.Errorf("user declined podman install")
			}
			result, err := workflow.InstallTool(podmanTool)
			if err != nil {
				return fmt.Errorf("failed to install podman: %v\n%s", err, result.Stderr)
			}
		} else {
			return fmt.Errorf("podman not installed and --non-interactive set")
		}
	}
	
	// Start podman registry container
	fmt.Println("  - Starting podman registry container...")
	
	// Check if registry container already exists
	checkCmd := exec.Command("podman", "ps", "-a", "--filter", "name=registry", "--format", "{{.Names}}")
	output, _ := checkCmd.Output()
	containerExists := strings.TrimSpace(string(output)) == "registry"
	
	if containerExists {
		// Container exists, try to start it
		startCmd := exec.Command("podman", "start", "registry")
		if err := startCmd.Run(); err != nil {
			// Container might be in bad state, remove and recreate
			_ = exec.Command("podman", "rm", "-f", "registry").Run()
			containerExists = false
		}
	}
	
	if !containerExists {
		startCmd := exec.Command("podman", "run", "-d", "--name", "registry", "-p", "5000:5000", "docker.io/library/registry:2")
		if err := startCmd.Run(); err != nil {
			return fmt.Errorf("failed to start registry: %v", err)
		}
	}
	
	// Wait for registry to be ready
	fmt.Println("  - Waiting for registry to start...")
	const maxAttempts = 30
	const waitInterval = 1 * time.Second
	
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		cmd := exec.Command("curl", "-s", "-o", "/dev/null", "http://localhost:5000/v2/")
		if err := cmd.Run(); err == nil {
			fmt.Println("  ✓ Local registry is serving at localhost:5000")
			return nil
		}
		if attempt < maxAttempts {
			time.Sleep(waitInterval)
		}
	}
	
	return fmt.Errorf("timeout waiting for registry at localhost:5000")
}

// ensureCluster ensures a Kubernetes cluster is running
func ensureCluster(interactiveMode bool) error {
	fmt.Println("Checking Kubernetes cluster...")
	
	// Check if cluster is reachable
	kubectlCmd := exec.Command("kubectl", "get", "nodes", "-o", "name")
	if err := kubectlCmd.Run(); err == nil {
		fmt.Println("  ✓ Cluster is reachable")
		
		// Check if minikube
		minikubeCmd := exec.Command("minikube", "status", "-o", "json")
		if minikubeCmd.Run() == nil {
			fmt.Println("  ✓ Minikube cluster detected")
		}
		return nil
	}
	
	// Cluster not reachable - start minikube
	fmt.Println("  - Starting minikube cluster...")
	
	// Check for podman or docker
	hasPodman := exec.Command("podman", "--version").Run() == nil
	hasDocker := exec.Command("docker", "--version").Run() == nil
	
	driver := "podman"
	if !hasPodman && hasDocker {
		driver = "docker"
	} else if !hasPodman && !hasDocker {
		return fmt.Errorf("neither podman nor docker available - install one first")
	}
	
	// Start minikube with pinned version
	startCmd := exec.Command("minikube", "start", "--driver="+driver, "--kubernetes-version="+PinnedKubernetesVersion, "--cpus=4", "--memory=6g")
	startCmd.Stdout = os.Stdout
	startCmd.Stderr = os.Stderr
	
	fmt.Printf("    Running: minikube start --driver=%s --kubernetes-version=%s --cpus=4 --memory=6g\n",
		driver, PinnedKubernetesVersion)
	
	if err := startCmd.Run(); err != nil {
		return fmt.Errorf("failed to start minikube: %v", err)
	}
	
	// Wait for nodes to be ready
	fmt.Println("  - Waiting for nodes to be ready...")
	const maxAttempts = 30
	const waitInterval = 5 * time.Second
	
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		cmd := exec.Command("kubectl", "get", "nodes", "-o", "jsonpath={.items[*].status.conditions[?(@.type==\"Ready\")].status}")
		output, err := cmd.Output()
		if err == nil {
			statuses := strings.Fields(string(output))
			allReady := true
			for _, status := range statuses {
				if status != "True" {
					allReady = false
					break
				}
			}
			if allReady {
				fmt.Println("  ✓ Nodes are ready")
				return nil
			}
		}
		if attempt < maxAttempts {
			time.Sleep(waitInterval)
		}
	}
	
	return fmt.Errorf("timeout waiting for nodes to be ready")
}

// bootstrapFlux bootstraps Flux on the cluster
func bootstrapFlux(interactiveMode bool, bootstrapCommit *bool) error {
	fmt.Println("Checking Flux installation...")
	
	// Check if flux-system namespace exists
	cmd := exec.Command("kubectl", "get", "ns", "flux-system")
	if err := cmd.Run(); err == nil {
		fmt.Println("  ✓ Flux already bootstrapped")
		*bootstrapCommit = false
		return nil
	}
	
	// Flux not bootstrapped - need to bootstrap
	fmt.Println("  - Bootstrapping Flux...")
	
	// Get repo URL
	remoteCmd := exec.Command("git", "remote", "get-url", "origin")
	remoteOutput, err := remoteCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get git remote: %v", err)
	}
	repoURL := strings.TrimSpace(string(remoteOutput))
	
	// Get GitHub token
	token := ""
	
	// Try gh auth token first
	if cmd := exec.Command("gh", "auth", "token"); cmd.Run() == nil {
		output, err := cmd.Output()
		if err == nil {
			token = strings.TrimSpace(string(output))
		}
	}
	
	if token == "" {
		// Try GITHUB_TOKEN env var
		token = os.Getenv("GITHUB_TOKEN")
	}
	
	var useSSH bool
	if token == "" {
		// No token available - fall back to SSH with warning
		if interactiveMode {
			fmt.Println("  ⚠ No GitHub token found - falling back to SSH auth")
			fmt.Println("     Note: flux's Go SSH stack cannot access macOS Keychain")
			fmt.Println("     Consider using token auth: export GITHUB_TOKEN=$(gh auth token)")
			
			var useSSHConfirm bool
			if err := huh.NewConfirm().
				Title("Proceed with SSH auth (may fail)?").
				Value(&useSSHConfirm).
				Run(); err != nil {
				return err
			}
			useSSH = useSSHConfirm
		} else {
			// Non-interactive with no token - use SSH
			useSSH = true
			fmt.Println("  ⚠ No GitHub token found - using SSH auth (may fail)")
		}
	}
	
	// Normalize repo URL to HTTPS for token auth
	sshURL := repoURL
	repoURL = workflow.NormalizeGitURLForHTTPS(repoURL)
	
	// Build bootstrap command
	var bootstrapCmd *exec.Cmd
	if useSSH {
		bootstrapCmd = exec.Command("flux", "bootstrap", "git", "--url="+sshURL, "--branch=main", "--path=./clusters/minikube")
	} else {
		bootstrapCmd = exec.Command("flux", "bootstrap", "git", "--url="+repoURL, "--branch=main", "--path=./clusters/minikube", "--token-auth")
		bootstrapCmd.Env = append(os.Environ(), "GITHUB_TOKEN="+token)
	}
	
	bootstrapCmd.Stdout = os.Stdout
	bootstrapCmd.Stderr = os.Stderr
	
	if useSSH {
		fmt.Printf("    Running: flux bootstrap git --url=%s --branch=main --path=./clusters/minikube\n", sshURL)
		fmt.Println("    (using SSH - may fail if flux cannot access your SSH key)")
	} else {
		fmt.Printf("    Running: flux bootstrap git --url=%s --branch=main --path=./clusters/minikube --token-auth\n", repoURL)
		fmt.Println("    (using token auth)")
	}
	
	if err := bootstrapCmd.Run(); err != nil {
		return fmt.Errorf("flux bootstrap failed: %v\n  Hint: If using SSH, try token auth with GITHUB_TOKEN=$(gh auth token)", err)
	}
	
	*bootstrapCommit = true
	fmt.Println("  ✓ Flux bootstrapped")
	
	return nil
}

// waitForConvergence waits for all kustomizations to be ready
func waitForConvergence() error {
	fmt.Println("Waiting for all kustomizations to converge...")
	
	const maxAttempts = 60  // 10 minutes at 10s intervals
	const waitInterval = 10 * time.Second
	
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		cmd := exec.Command("flux", "get", "kustomizations", "-A", "-o", "jsonpath={.items[*].status.ready}")
		output, err := cmd.Output()
		if err != nil {
			// Check for known cold-start race conditions
			stderr := err.Error()
			if strings.Contains(stderr, "connection refused") ||
				strings.Contains(stderr, "no matches for kind") {
				// These are transient - try to reconcile and continue
				fmt.Printf("  Attempt %d/%d: Transient error detected, retrying...\n", attempt, maxAttempts)
				
				// Try to reconcile all kustomizations
				reconcileCmd := exec.Command("sh", "-c", "flux reconcile kustomization -A --with-source 2>/dev/null || true")
				_ = reconcileCmd.Run()
				
				time.Sleep(waitInterval)
				continue
			}
			
			return fmt.Errorf("failed to get kustomizations: %v", err)
		}
		
		// Parse the ready statuses
		statuses := strings.Fields(string(output))
		allReady := true
		for _, status := range statuses {
			if status != "True" {
				allReady = false
				break
			}
		}
		
		if allReady {
			fmt.Println("  ✓ All kustomizations are ready")
			return nil
		}
		
		fmt.Printf("  Attempt %d/%d: Waiting for kustomizations...\n", attempt, maxAttempts)
		
		if attempt < maxAttempts {
			time.Sleep(waitInterval)
		}
	}
	
	// Timeout - print which kustomizations are not ready
	cmd := exec.Command("flux", "get", "kustomizations", "-A")
	output, _ := cmd.Output()
	fmt.Printf("\nTimeout waiting for convergence.\nStuck kustomizations:\n%s\n", output)
	
	return fmt.Errorf("timeout waiting for all kustomizations to be ready")
}

// reportFailure prints a failure report and returns the error
func reportFailure(failedPhase string, err error, completed map[string]bool) error {
	fmt.Println()
	fmt.Println(errorStyle.Render(fmt.Sprintf("Phase %s failed: %v", failedPhase, err)))
	fmt.Println()
	
	fmt.Println("Completed phases:")
	for _, phase := range setupPhases {
		status := "✓"
		if !completed[phase] {
			status = "✗"
		}
		fmt.Printf("  [%s] %s\n", status, phase)
	}
	
	fmt.Println()
	fmt.Println("Re-run model-cli setup to resume from the failure point.")
	
	return err
}

// Styling for setup output - reuses styles from wizard.go
// errorStyle is defined here for setup-specific errors
var errorStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#FF5555")).
	Bold(true)

func init() {
	rootCmd.AddCommand(setupCmd)

	// Flags for setup command
	setupCmd.Flags().Bool("yes", false, "Non-interactive: install all missing tools and proceed with defaults")
}
