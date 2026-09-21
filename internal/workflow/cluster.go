package workflow

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// ClusterSetup handles setting up a local Kubernetes cluster with minikube
// and optional Flux bootstrap
func ClusterSetup(clusterCPUs, clusterMemory string) error {
	// Step 1: Ensure minikube and kubectl are installed
	fmt.Println("Setting up Kubernetes cluster...")
	
	// Check and install minikube
	minikubeTool, err := GetTool("minikube")
	if err != nil {
		return fmt.Errorf("failed to get minikube tool: %v", err)
	}
	
	if !minikubeTool.IsInstalled() {
		fmt.Println("Installing minikube...")
		_, err := InstallTool(minikubeTool)
		if err != nil {
			return fmt.Errorf("failed to install minikube: %v", err)
		}
	}
	
	// Check and install kubectl
	kubectlTool, err := GetTool("kubectl")
	if err != nil {
		return fmt.Errorf("failed to get kubectl tool: %v", err)
	}
	
	if !kubectlTool.IsInstalled() {
		fmt.Println("Installing kubectl...")
		_, err := InstallTool(kubectlTool)
		if err != nil {
			return fmt.Errorf("failed to install kubectl: %v", err)
		}
	}
	
	// Step 2: Start minikube cluster
	fmt.Println("Starting minikube cluster...")
	
	args := []string{"start", "--cpus=" + clusterCPUs, "--memory=" + clusterMemory}
	cmd := exec.Command("minikube", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to start minikube: %v\nOutput: %s", err, string(output))
	}
	
	fmt.Println("minikube output:", string(output))
	
	// Step 3: Wait for cluster to be ready
	fmt.Println("Waiting for cluster to be ready...")
	if err := WaitForClusterReady(300 * time.Second); err != nil {
		return fmt.Errorf("cluster did not become ready: %v", err)
	}
	
	fmt.Println("✓ Cluster is ready!")
	return nil
}

// WaitForClusterReady waits for the Kubernetes cluster to be ready
func WaitForClusterReady(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	
	for time.Now().Before(deadline) {
		cmd := exec.Command("kubectl", "get", "nodes", "-o", "jsonpath={.items[*].status.conditions[?(@.type==\"Ready\")].status}")
		output, err := cmd.Output()
		if err != nil {
			// kubectl might not be ready yet, wait and retry
			time.Sleep(5 * time.Second)
			continue
		}
		
		status := strings.TrimSpace(string(output))
		if status == "True" {
			return nil
		}
		
		time.Sleep(10 * time.Second)
	}
	
	return fmt.Errorf("timeout waiting for cluster to be ready")
}

// FluxBootstrap performs Flux bootstrap on the cluster using SSH transport
// SSH transport is used because it only requires an SSH key (checked by doctor)
// whereas HTTPS transport would require a GitHub PAT
func FluxBootstrap(repoURL, repoPath string) error {
	// Step 1: Ensure flux is installed
	fluxTool, err := GetTool("flux")
	if err != nil {
		return fmt.Errorf("failed to get flux tool: %v", err)
	}
	
	if !fluxTool.IsInstalled() {
		fmt.Println("Installing flux...")
		_, err := InstallTool(fluxTool)
		if err != nil {
			return fmt.Errorf("failed to install flux: %v", err)
		}
	}
	
	// Step 2: Convert HTTPS URLs to SSH if needed
	// Flux bootstrap git requires SSH transport
	sshURL := repoURL
	if strings.HasPrefix(repoURL, "https://github.com/") {
		// Convert https://github.com/owner/repo to ssh://git@github.com/owner/repo.git
		sshURL = "ssh://git@github.com/" + strings.TrimPrefix(repoURL, "https://github.com/") + ".git"
		fmt.Printf("Note: Using SSH transport for Flux bootstrap: %s\n", sshURL)
		fmt.Println("Ensure you have an SSH key registered with GitHub.")
	}
	
	// Step 3: Run flux bootstrap with SSH
	fmt.Println("Running flux bootstrap...")
	cmd := exec.Command("flux", "bootstrap", "git", "--url", sshURL, "--branch", "main", "--path", repoPath)
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to bootstrap flux: %v\nOutput: %s", err, string(output))
	}
	
	fmt.Println("flux bootstrap output:", string(output))
	
	// Step 4: Wait for Flux to be ready
	fmt.Println("Waiting for Flux to be ready...")
	if err := WaitForFluxReady(300 * time.Second); err != nil {
		return fmt.Errorf("Flux did not become ready: %v", err)
	}
	
	fmt.Println("✓ Flux is ready!")
	return nil
}

// extractOwner extracts the owner from a GitHub URL
func extractOwner(url string) string {
	// Handle both SSH and HTTPS URLs
	url = strings.TrimSuffix(url, ".git")
	parts := strings.Split(url, "/")
	if len(parts) >= 4 {
		return parts[3]
	}
	return ""
}

// extractRepo extracts the repository name from a GitHub URL
func extractRepo(url string) string {
	// Handle both SSH and HTTPS URLs
	url = strings.TrimSuffix(url, ".git")
	parts := strings.Split(url, "/")
	if len(parts) >= 5 {
		return parts[4]
	}
	return ""
}

// WaitForFluxReady waits for Flux to be ready in the cluster
func WaitForFluxReady(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	
	for time.Now().Before(deadline) {
		cmd := exec.Command("flux", "get", "kustomizations", "-n", "flux-system")
		output, err := cmd.Output()
		if err != nil {
			// Flux might not be ready yet, wait and retry
			time.Sleep(5 * time.Second)
			continue
		}
		
		// Check if flux-system kustomization is ready
		out := string(output)
		if strings.Contains(out, "flux-system") && strings.Contains(out, "True") {
			return nil
		}
		
		time.Sleep(10 * time.Second)
	}
	
	return fmt.Errorf("timeout waiting for Flux to be ready")
}

// RegistrySetup sets up a local OCI registry using Podman
func RegistrySetup() error {
	// Step 1: Ensure podman is installed
	podmanTool, err := GetTool("podman")
	if err != nil {
		return fmt.Errorf("failed to get podman tool: %v", err)
	}
	
	if !podmanTool.IsInstalled() {
		fmt.Println("Installing podman...")
		// Check if we're on Intel Mac and provide specific guidance
		if runtime.GOARCH == "amd64" {
			fmt.Println("Note: On Intel Macs, use podman 5.x (latest versions don't support the Apple hypervisor)")
			fmt.Println("Install with: brew install podman@5")
		}
		_, err := InstallTool(podmanTool)
		if err != nil {
			return fmt.Errorf("failed to install podman: %v\n\nFor Intel Macs, ensure you're using podman 5.x: brew install podman@5", err)
		}
	}
	
	// Step 2: Initialize podman machine if needed
	fmt.Println("Setting up Podman machine...")
	
	// Check if machine is already initialized
	checkCmd := exec.Command("podman", "machine", "inspect")
	if err := checkCmd.Run(); err != nil {
		// Machine not initialized, initialize it
		fmt.Println("Initializing Podman machine...")
		initCmd := exec.Command("podman", "machine", "init")
		output, err := initCmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to initialize podman machine: %v\nOutput: %s", err, string(output))
		}
		fmt.Println("Podman machine initialized")
	}
	
	// Step 3: Start podman machine
	fmt.Println("Starting Podman machine...")
	startCmd := exec.Command("podman", "machine", "start")
	output, err := startCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to start podman machine: %v\nOutput: %s", err, string(output))
	}
	fmt.Println("Podman machine started")
	
	// Give it a moment to start
	time.Sleep(5 * time.Second)
	
	// Step 4: Start the registry container
	fmt.Println("Starting local OCI registry...")
	registryCmd := exec.Command("podman", "run", "-d", "--rm", "--name", "model-cli-registry", "-p", "5000:5000", "registry:2")
	output, err = registryCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to start registry: %v\nOutput: %s", err, string(output))
	}
	fmt.Println("Registry container started")
	
	// Step 5: Wait for registry to be ready
	fmt.Println("Waiting for registry to be ready...")
	if err := WaitForRegistryReady(30 * time.Second); err != nil {
		return fmt.Errorf("registry did not become ready: %v", err)
	}
	
	fmt.Println("✓ Registry is ready at localhost:5000!")
	return nil
}

// WaitForRegistryReady waits for the local OCI registry to be ready
func WaitForRegistryReady(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	
	for time.Now().Before(deadline) {
		cmd := exec.Command("curl", "-s", "-f", "http://localhost:5000/v2/")
		err := cmd.Run()
		if err == nil {
			// curl succeeded and returned OK
			return nil
		}
		
		time.Sleep(2 * time.Second)
	}
	
	return fmt.Errorf("timeout waiting for registry to be ready")
}
