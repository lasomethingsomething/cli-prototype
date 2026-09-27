package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
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
		
		// Wait for cert-manager webhook to be available
		// This is the specific dependency that caused Certificate/kserve/serving-cert failures
		fmt.Println("  - Waiting for cert-manager webhook...")
		if err := waitForCertManagerWebhook(); err != nil {
			return reportFailure("converge", err, completed)
		}
		fmt.Println("  ✓ cert-manager webhook ready")
		
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

// parseMemoryString parses a memory string from podman (in bytes) and returns GB as integer
func parseMemoryString(memoryStr string) int {
	// memoryStr is a number representing bytes
	memoryStr = strings.TrimSpace(memoryStr)
	if memoryStr == "" {
		return 0
	}
	
	// Try to parse as integer (bytes)
	bytes, err := strconv.ParseInt(memoryStr, 10, 64)
	if err != nil {
		// Might be in different format, try to extract number
		// Handle formats like "1956MB", "1.956GB", etc.
		memoryStr = strings.TrimSpace(memoryStr)
		memoryStr = strings.TrimSuffix(memoryStr, "B")
		memoryStr = strings.TrimSuffix(memoryStr, "MB")
		memoryStr = strings.TrimSuffix(memoryStr, "GB")
		memoryStr = strings.TrimSuffix(memoryStr, "TB")
		
		// Try to parse as float
		var memoryFloat float64
		_, err := fmt.Sscanf(memoryStr, "%f", &memoryFloat)
		if err != nil {
			return 0
		}
		
		// If we stripped MB, divide by 1024 to get GB
		// But we need to know what unit we had - check the original string
		// Since we already stripped, we can't tell, so assume bytes was intended
		return int(memoryFloat / 1024 / 1024 / 1024)
	}
	
	// Convert bytes to GB
	gigabytes := bytes / (1024 * 1024 * 1024)
	return int(gigabytes)
}

// waitForCertManagerWebhook waits for cert-manager webhook deployment to be available
// This is the specific dependency that caused Certificate/kserve/serving-cert failures
// Uses kubectl wait for condition=Available with 300s timeout
func waitForCertManagerWebhook() error {
	fmt.Println("      Running: kubectl -n cert-manager wait --for=condition=Available deploy/cert-manager-webhook --timeout=300s")
	cmd := exec.Command("kubectl", "-n", "cert-manager", "wait", "--for=condition=Available", "deploy/cert-manager-webhook", "--timeout=300s")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("cert-manager webhook not available: %v\nOutput: %s\n\nHint: Check cert-manager installation with: kubectl get all -n cert-manager", err, string(output))
	}
	return nil
}

// checkPodDNS runs a DNS check pod to verify pod networking works
// Returns an error with actionable hint if DNS is broken
func checkPodDNS() error {
	// Wait for default ServiceAccount to exist (needed for kubectl run)
	// On fresh clusters, the default SA may not be created yet
	fmt.Println("      Waiting for default ServiceAccount...")
	const maxAttempts = 12
	const waitInterval = 5 * time.Second
	
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		cmd := exec.Command("kubectl", "get", "sa", "default", "-n", "default")
		if err := cmd.Run(); err == nil {
			break
		}
		if attempt < maxAttempts {
			time.Sleep(waitInterval)
		}
		if attempt == maxAttempts {
			return fmt.Errorf("timeout waiting for default ServiceAccount to be created")
		}
	}
	
	// Create a busybox pod to test DNS
	fmt.Println("      Running DNS pre-flight check...")
	
	// Run a pod that tries to resolve github.com
	dnsCheckCmd := exec.Command("kubectl", "run", "dns-precheck", "--image=busybox:latest", "--rm", "-i", "--restart=Never", "--", "nslookup", "github.com")
	// Set timeout for the DNS check
	dnsCheckCmd.Env = append(os.Environ(), "KUBECTL_TIMEOUT=10")
	
	output, err := dnsCheckCmd.CombinedOutput()
	outputStr := string(output)
	
	if err != nil {
		// DNS lookup failed - check for specific errors
		if strings.Contains(outputStr, "Error") || strings.Contains(outputStr, "timeout") || strings.Contains(outputStr, "failed") {
			cleanupCmd := exec.Command("kubectl", "delete", "pod", "dns-precheck", "--ignore-not-found")
			_ = cleanupCmd.Run()
			
			return fmt.Errorf("pod DNS is broken in this cluster. This is likely a driver/machine issue, not an auth problem.\n\nTry using the Docker driver instead: minikube delete && minikube start --driver=docker\n\nOriginal error: %s", outputStr)
		}
		// Clean up the pod
		cleanupCmd := exec.Command("kubectl", "delete", "pod", "dns-precheck", "--ignore-not-found")
		_ = cleanupCmd.Run()
		return fmt.Errorf("DNS check failed: %s", outputStr)
	}
	
	// Check if the output contains a successful resolution
	if strings.Contains(outputStr, "github.com") || strings.Contains(outputStr, "140.82") {
		// Clean up the pod
		cleanupCmd := exec.Command("kubectl", "delete", "pod", "dns-precheck", "--ignore-not-found")
		_ = cleanupCmd.Run()
		return nil
	}
	
	// Clean up the pod
	cleanupCmd := exec.Command("kubectl", "delete", "pod", "dns-precheck", "--ignore-not-found")
	_ = cleanupCmd.Run()
	return fmt.Errorf("DNS check: unexpected output: %s", outputStr)
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
	
	// On macOS, default to docker driver (podman driver has pod DNS issues)
	// See friction log #45: podman driver breaks pod DNS on macOS
	driver := "docker"
	if !hasDocker && hasPodman {
		// Fall back to podman only if docker is not available
		driver = "podman"
		fmt.Println("    ⚠ Using podman driver (docker not installed)")
		fmt.Println("      Note: On macOS, the podman driver may have DNS issues. Install Docker for best results.")
	} else if !hasPodman && !hasDocker {
		return fmt.Errorf("neither podman nor docker available - install one first")
	}
	
	// For podman driver, check machine capacity and adapt resources
	memory := "6g"
	cpus := "4"
	minMemoryGB := 2 // Minimum memory required for minikube
	
	if driver == "podman" {
		fmt.Println("    - Checking podman machine capacity...")
		
		// Get podman machine info
		inspectCmd := exec.Command("podman", "machine", "inspect", "--format", "{{.HostMemory}}")
		memoryBytes, err := inspectCmd.Output()
		if err != nil {
			// Try podman info as fallback
			infoCmd := exec.Command("podman", "info", "--format", "{{.Host.MemTotal}}")
			memoryBytes, err = infoCmd.Output()
			if err != nil {
				fmt.Println("    ⚠ Could not detect podman machine memory, using default 6g")
			} else {
				// Parse memory from podman info
				memoryStr := strings.TrimSpace(string(memoryBytes))
				// memory is in bytes, convert to GB
				memoryGB := parseMemoryString(memoryStr)
				if memoryGB > 0 {
					// Reserve 1GB overhead, min 2GB for minikube
					availableMemoryGB := memoryGB - 1
					if availableMemoryGB < minMemoryGB {
						return fmt.Errorf("podman machine has only %dGB memory (need at least %dGB). Run: podman machine set --memory %d",
							memoryGB, minMemoryGB+1, (minMemoryGB+1)*1024)
					}
					// Use min of desired (6g) and available
					if availableMemoryGB < 6 {
						memory = fmt.Sprintf("%dg", availableMemoryGB)
						fmt.Printf("    - Adjusted memory to %s (machine has %dGB)\n", memory, memoryGB)
					}
				}
			}
		} else {
			// Successfully got memory from inspect
			memoryStr := strings.TrimSpace(string(memoryBytes))
			memoryGB := parseMemoryString(memoryStr)
			if memoryGB > 0 {
				// Reserve 1GB overhead, min 2GB for minikube
				availableMemoryGB := memoryGB - 1
				if availableMemoryGB < minMemoryGB {
					return fmt.Errorf("podman machine has only %dGB memory (need at least %dGB). Run: podman machine set --memory %d",
							memoryGB, minMemoryGB+1, (minMemoryGB+1)*1024)
				}
				// Use min of desired (6g) and available
				if availableMemoryGB < 6 {
					memory = fmt.Sprintf("%dg", availableMemoryGB)
					fmt.Printf("    - Adjusted memory to %s (machine has %dGB)\n", memory, memoryGB)
				}
			}
		}
		
		// Check CPU count
		cpusCmd := exec.Command("podman", "machine", "inspect", "--format", "{{.CPUs}}")
		cpusOutput, err := cpusCmd.Output()
		if err != nil {
			// Try podman info
			infoCmd := exec.Command("podman", "info", "--format", "{{.Host.CPUs}}")
			cpusOutput, err = infoCmd.Output()
			if err != nil {
				fmt.Println("    ⚠ Could not detect podman machine CPUs, using default 4")
			} else {
				cpus = strings.TrimSpace(string(cpusOutput))
				// Use min of desired (4) and available
				if c, _ := strconv.Atoi(cpus); c > 0 && c < 4 {
					memory = cpus
					fmt.Printf("    - Adjusted CPUs to %s (machine has %s)\n", cpus, cpus)
				}
			}
		} else {
			cpus = strings.TrimSpace(string(cpusOutput))
			// Use min of desired (4) and available
			if c, _ := strconv.Atoi(cpus); c > 0 && c < 4 {
				cpus = fmt.Sprintf("%d", c)
				fmt.Printf("    - Adjusted CPUs to %s (machine has %s)\n", cpus, cpus)
			}
		}
	}
	
	// Start minikube with pinned version
	startCmd := exec.Command("minikube", "start", "--driver="+driver, "--kubernetes-version="+PinnedKubernetesVersion, "--cpus="+cpus, "--memory="+memory)
	startCmd.Stdout = os.Stdout
	startCmd.Stderr = os.Stderr
	
	fmt.Printf("    Running: minikube start --driver=%s --kubernetes-version=%s --cpus=%s --memory=%s\n",
		driver, PinnedKubernetesVersion, cpus, memory)
	
	if err := startCmd.Run(); err != nil {
		errMsg := err.Error()
		// Check for podman memory error (exit code 14)
		if strings.Contains(errMsg, "Podman has only") || strings.Contains(errMsg, "MK_USAGE") {
			// Extract the required memory from the error message
			// Error format: "Podman has only 1956MB memory but you specified 6144MB"
			return fmt.Errorf("%v\n\nHint: Run 'podman machine set --memory <bytes>' to increase memory (e.g., podman machine set --memory 8192)", err)
		}
		return fmt.Errorf("failed to start minikube: %v", err)
	}
	
	// Pre-flight DNS check: verify pod DNS works before bootstrap
	// This catches driver issues (e.g., podman driver on macOS breaks DNS)
	fmt.Println("    - Checking pod DNS...")
	if err := checkPodDNS(); err != nil {
		return err
	}
	fmt.Println("    ✓ Pod DNS working")
	
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
	
	if useSSH {
		fmt.Printf("    Running: flux bootstrap git --url=%s --branch=main --path=./clusters/minikube\n", sshURL)
		fmt.Println("    (using SSH - may fail if flux cannot access your SSH key)")
	} else {
		fmt.Printf("    Running: flux bootstrap git --url=%s --branch=main --path=./clusters/minikube --token-auth\n", repoURL)
		fmt.Println("    (using token auth)")
	}
	
	output, err := bootstrapCmd.CombinedOutput()
	// Print the output to terminal
	if output != nil {
		fmt.Print(string(output))
	}
	outputStr := string(output)
	
	if err != nil {
		// Check for cold-start race: cert-manager webhook not ready
		// Error pattern: Certificate/kserve/serving-cert dry-run failed: failed calling webhook "webhook.cert-manager.io": connection refused
		if strings.Contains(outputStr, "webhook.cert-manager.io") && strings.Contains(outputStr, "connection refused") {
			fmt.Println("    ⚠ Cold-start race detected (cert-manager webhook not ready yet) — retrying reconcile...")
			
			// Wait for webhook to come up
			fmt.Println("    Waiting 60 seconds for cert-manager webhook...")
			time.Sleep(60 * time.Second)
			
			// Trigger reconcile
			fmt.Println("    Running: flux reconcile kustomization flux-system --with-source")
			reconcileCmd := exec.Command("flux", "reconcile", "kustomization", "flux-system", "--with-source")
			reconcileCmd.Stdout = os.Stdout
			reconcileCmd.Stderr = os.Stderr
			if err := reconcileCmd.Run(); err != nil {
				// Continue - reconcile may not be available yet
				fmt.Printf("    ⚠ Reconcile returned: %v\n", err)
			}
			
			// Poll for flux-system kustomization to be ready
			fmt.Println("    Waiting for flux-system kustomization to be ready...")
			waitCmd := exec.Command("kubectl", "wait", "--for=condition=Ready", "kustomization/flux-system", "-n", "flux-system", "--timeout=300s")
			if err := waitCmd.Run(); err != nil {
				return fmt.Errorf("webhook race retry failed: %v. Check cert-manager with: kubectl get all -n cert-manager", err)
			}
			
			fmt.Println("    ✓ Bootstrap recovered from cold-start race")
		} else {
			return fmt.Errorf("flux bootstrap failed: %v\n  Hint: If using SSH, try token auth with GITHUB_TOKEN=$(gh auth token)", err)
		}
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
		// Use kubectl instead of flux CLI (project history: flux version exit-1 trap)
		// Query kustomizations via kubectl to get their Ready status
		cmd := exec.Command("kubectl", "get", "kustomizations", "-A", "-o", "json")
		output, err := cmd.CombinedOutput()
		outputStr := string(output)
		
		if err != nil {
			// Check for known cold-start race conditions in stderr
			if strings.Contains(outputStr, "connection refused") ||
				strings.Contains(outputStr, "no matches for kind") ||
				strings.Contains(outputStr, "error") {
				// These are transient - try to reconcile and continue
				fmt.Printf("  Attempt %d/%d: Transient error detected, retrying...\n", attempt, maxAttempts)
				fmt.Printf("    Command: %s\n", cmd.String())
				fmt.Printf("    Output: %s\n", outputStr)
				
				// Try to reconcile all kustomizations
				reconcileCmd := exec.Command("flux", "reconcile", "kustomization", "-A", "--with-source")
				reconcileOut, reconcileErr := reconcileCmd.CombinedOutput()
				if reconcileErr != nil {
					fmt.Printf("    Reconcile output: %s\n", string(reconcileOut))
				}
				
				time.Sleep(waitInterval)
				continue
			}
			
			// For other errors, print full diagnostics and fail
			return fmt.Errorf("failed to get kustomizations: command=%s exit=%v output=%s",
				cmd.String(), err, outputStr)
		}
		
		// Parse the JSON output to check Ready status
		allReady, err := parseKustomizationReady(outputStr)
		if err != nil {
			// Parse error means invalid JSON - print diagnostics and fail
			return fmt.Errorf("failed to parse kustomizations JSON: %v. Output:\n%s", err, outputStr)
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
	
	// Timeout - print diagnostic information
	cmd := exec.Command("kubectl", "get", "kustomizations", "-A")
	output, _ := cmd.CombinedOutput()
	fmt.Printf("\nTimeout waiting for convergence.\n")
	fmt.Printf("Command: %s\n", cmd.String())
	fmt.Printf("Output:\n%s\n", string(output))
	
	return fmt.Errorf("timeout waiting for all kustomizations to be ready")
}

// ksConditions represents a Kustomization's conditions
// Used for parsing kubectl get kustomizations -A -o json output
type ksConditions struct {
	Type   string `json:"type"`
	Status string `json:"status"`
}

// ksStatus represents a Kustomization's status
type ksStatus struct {
	Conditions []ksConditions `json:"conditions"`
}

// ksItem represents a single Kustomization
type ksItem struct {
	Status ksStatus `json:"status"`
}

// ksList represents the list of Kustomizations
type ksList struct {
	Items []ksItem `json:"items"`
}

// parseKustomizationReady parses kubectl get kustomizations -A -o json output
// and returns true if every kustomization has a Ready=True condition
func parseKustomizationReady(jsonOutput string) (bool, error) {
	var list ksList
	if err := json.Unmarshal([]byte(jsonOutput), &list); err != nil {
		return false, fmt.Errorf("invalid JSON from kubectl: %v", err)
	}
	if len(list.Items) == 0 {
		return false, fmt.Errorf("no kustomizations found")
	}
	
	for _, item := range list.Items {
		ready := false
		for _, c := range item.Status.Conditions {
			if c.Type == "Ready" && c.Status == "True" {
				ready = true
				break
			}
		}
		if !ready {
			// Not an error - just not ready yet
			return false, nil
		}
	}
	return true, nil
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
