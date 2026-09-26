package workflow

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ToolCategory represents the category of a tool for the doctor command
type ToolCategory string

const (
	// CategoryBrew means the tool can be installed via brew
	CategoryBrew ToolCategory = "brew"
	// CategoryEnvironment means the tool is an environment prerequisite
	CategoryEnvironment ToolCategory = "environment"
	// CategoryCluster means the tool is a cluster-side dependency
	CategoryCluster ToolCategory = "cluster"
)

// Tool defines the interface for tools that the doctor command checks
type Tool interface {
	Name() string
	Category() ToolCategory
	IsInstalled() bool
	InstallInstructions() string
	Description() string
}

// ToolStatus represents the status of a tool check
type ToolStatus string

const (
	StatusMissing          ToolStatus = "missing"
	StatusInstalledNotRunning ToolStatus = "installed-not-running"
	StatusNotDeployed      ToolStatus = "not-deployed"
	StatusReady            ToolStatus = "ready"
)

// ToolResult represents the check result for a single tool
type ToolResult struct {
	Tool      Tool
	Installed bool
	Status    ToolStatus
	Hint      string
	Error     error
}

// DoctorReport represents the complete report from the doctor command
type DoctorReport struct {
	Results []ToolResult
}

// AllTools returns all tools that the doctor command checks
func AllTools() []Tool {
	return []Tool{
		// Brew-installable tools
		&orasTool{},
		&syftTool{},
		&cosignTool{},
		&fluxTool{},
		&podmanTool{},
		&kubectlTool{},
		&minikubeTool{},
		&notationTool{},
		
		// Environment prerequisites
		&xcodeCLTTool{},
		&sshKeyTool{},
		
		// Cluster-side tools (verify only)
		&kserveTool{},
		&certManagerTool{},
		&metricsServerTool{},
	}
}

// BrewInstallableTools returns only the tools that can be installed via brew
func BrewInstallableTools() []Tool {
	var tools []Tool
	for _, t := range AllTools() {
		if t.Category() == CategoryBrew {
			tools = append(tools, t)
		}
	}
	return tools
}

// CheckAll runs IsInstalled() on all tools and returns a report
func CheckAll() *DoctorReport {
	report := &DoctorReport{
		Results: make([]ToolResult, 0, len(AllTools())),
	}
	
	for _, tool := range AllTools() {
		result := checkToolStatus(tool)
		report.Results = append(report.Results, *result)
	}
	
	return report
}

// checkToolStatus checks a tool and returns its status and hint
func checkToolStatus(tool Tool) *ToolResult {
	status := StatusMissing
	hint := ""
	installed := tool.IsInstalled()
	
	// Special handling for tools with running state beyond just being installed
	if tool.Name() == "podman" {
		// podman: check binary first, then machine state
		_, err := exec.LookPath("podman")
		if err != nil {
			status = StatusMissing
		} else {
			// Binary installed, check if machine is running
			cmd := exec.Command("podman", "machine", "inspect")
			if err := cmd.Run(); err != nil {
				status = StatusInstalledNotRunning
				hint = "podman machine start"
			} else {
				status = StatusReady
			}
		}
	} else if tool.Name() == "minikube" {
		// minikube: check binary first, then cluster state
		_, err := exec.LookPath("minikube")
		if err != nil {
			status = StatusMissing
			hint = "brew install minikube"
		} else {
			// Binary installed, check if cluster is running
			cmd := exec.Command("minikube", "status", "-o", "json")
			if err := cmd.Run(); err != nil {
				// Check if cluster is reachable via kubectl instead
				kubectlCmd := exec.Command("kubectl", "get", "nodes", "-o", "name")
				if kubectlErr := kubectlCmd.Run(); kubectlErr != nil {
					status = StatusInstalledNotRunning
					hint = "minikube start"
				} else {
					// Cluster is reachable, minikube is working
					status = StatusReady
				}
			} else {
				status = StatusReady
			}
		}
	} else if tool.Category() == CategoryCluster {
		// Cluster tools: check kubectl reachability, then deployment presence
		cmd := exec.Command("kubectl", "get", "deployments", "-A", "-o", "name")
		if err := cmd.Run(); err != nil {
			// kubectl cannot reach cluster
			status = StatusMissing
			hint = "minikube start / see README bootstrap"
		} else {
			// kubectl works, check if the specific deployment exists
			if installed {
				status = StatusReady
			} else {
				// Cluster is reachable but deployment not yet created (e.g., Flux still reconciling)
				status = StatusNotDeployed
				hint = "Wait for Flux reconciliation or check Flux logs with 'flux get kustomizations -A'"
			}
		}
	} else {
		// Regular tools: just use IsInstalled()
		if installed {
			status = StatusReady
		} else {
			status = StatusMissing
		}
	}
	
	return &ToolResult{
		Tool:      tool,
		Installed: installed,
		Status:    status,
		Hint:      hint,
		Error:     nil,
	}
}

// CheckTool checks a specific tool
func CheckTool(name string) (*ToolResult, error) {
	for _, tool := range AllTools() {
		if tool.Name() == name {
			return checkToolStatus(tool), nil
		}
	}
	return nil, fmt.Errorf("unknown tool: %s", name)
}

// InstallToolResult contains the result of an install attempt
type InstallToolResult struct {
	Tool    Tool
	Error   error
	Stdout  string
	Stderr  string
}

// InstallTool installs a specific tool using brew.
// It returns the result including any error and stderr output.
// If the tool is already installed but unlinked, it attempts to link it.
func InstallTool(tool Tool) (*InstallToolResult, error) {
	if tool.Category() != CategoryBrew {
		return nil, fmt.Errorf("tool %s is not brew-installable", tool.Name())
	}
	
	installCmd := tool.InstallInstructions()
	if !strings.Contains(installCmd, "brew install") {
		return nil, fmt.Errorf("tool %s does not have a brew install command", tool.Name())
	}
	
	// Extract the package name from the install command
	parts := strings.Fields(installCmd)
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid brew install command: %s", installCmd)
	}
	
	packageName := strings.Join(parts[2:], " ")
	binaryName := tool.Name()
	
	// First, check if brew has the formula but it's unlinked
	checkCmd := exec.Command("brew", "list", packageName)
	if err := checkCmd.Run(); err == nil {
		// Package is installed by brew, check if binary is on PATH
		if _, err := exec.LookPath(binaryName); err != nil {
			// Binary not on PATH, try to link it
			fmt.Printf("Linking %s via brew...\n", binaryName)
			linkCmd := exec.Command("brew", "link", packageName)
			linkOutput, err := linkCmd.CombinedOutput()
			if err != nil {
				return &InstallToolResult{
					Tool:   tool,
					Error:  fmt.Errorf("failed to link %s: %v", packageName, err),
					Stderr: string(linkOutput),
				}, err
			}
			// Verify the link worked
			if _, err := exec.LookPath(binaryName); err != nil {
				return &InstallToolResult{
					Tool:   tool,
					Error:  fmt.Errorf("link succeeded but %s still not on PATH", binaryName),
					Stderr: string(linkOutput),
				}, fmt.Errorf("%s not on PATH after link", binaryName)
			}
			return &InstallToolResult{Tool: tool}, nil
		}
		// Binary is on PATH, already installed
		return &InstallToolResult{Tool: tool}, nil
	}
	
	// Package not installed, install it
	fmt.Printf("Installing %s via brew...\n", tool.Name())
	cmd := exec.Command("brew", "install", packageName)
	output, err := cmd.CombinedOutput()
	
	if err != nil {
		return &InstallToolResult{
			Tool:    tool,
			Error:   fmt.Errorf("failed to install %s: %v", tool.Name(), err),
			Stdout:  string(output),
			Stderr:  string(output),
		}, err
	}
	
	// Verify the binary is on PATH after install
	if _, err := exec.LookPath(binaryName); err != nil {
		// Try linking
		fmt.Printf("Linking %s via brew...\n", binaryName)
		linkCmd := exec.Command("brew", "link", packageName)
		linkOutput, linkErr := linkCmd.CombinedOutput()
		if linkErr != nil {
			return &InstallToolResult{
				Tool:   tool,
				Error:  fmt.Errorf("installed but not on PATH, failed to link %s: %v", packageName, linkErr),
				Stdout: string(output),
				Stderr: string(linkOutput),
			}, fmt.Errorf("installed but not on PATH and link failed: %v", linkErr)
		}
		// Verify the link worked
		if _, err := exec.LookPath(binaryName); err != nil {
			return &InstallToolResult{
				Tool:   tool,
				Error:  fmt.Errorf("installed and linked but %s still not on PATH", binaryName),
				Stdout: string(output),
				Stderr: string(linkOutput),
			}, fmt.Errorf("installed and linked but not on PATH")
		}
	}
	
	return &InstallToolResult{Tool: tool}, nil
}

// EnsureToolInstalled checks if a tool is installed and optionally installs it.
// It returns true if the tool is installed (either was already installed or was just installed),
// false if the user declined installation or the tool cannot be installed.
// This function handles the prompt and installation for brew-installable tools.
// For non-brew tools, it just prints the instructions.
// Note: This function uses fmt for output, so callers should handle TUI consistency.
func EnsureToolInstalled(toolName string, purpose string, interactive bool) (bool, error) {
	tool, err := GetTool(toolName)
	if err != nil {
		return false, fmt.Errorf("unknown tool: %s", toolName)
	}
	
	if tool.IsInstalled() {
		return true, nil
	}
	
	// Tool is not installed
	if !interactive {
		return false, nil
	}
	
	// Check if it's brew-installable
	if tool.Category() == CategoryBrew {
		fmt.Printf("%s is not installed — needed for %s. Install now? [Y/n] ", tool.Name(), purpose)
		
		// Read user input
		var response string
		_, err := fmt.Scanln(&response)
		if err != nil {
			// If there's an error reading input (e.g., non-interactive), treat as no
			return false, nil
		}
		
		response = strings.ToLower(strings.TrimSpace(response))
		if response == "y" || response == "" {
			_, err := InstallTool(tool)
			if err != nil {
				return false, err
			}
			return true, nil
		}
		return false, nil
	}
	
	// For non-brew tools, just explain
	fmt.Printf("%s is not installed — needed for %s.\n", tool.Name(), purpose)
	fmt.Printf("Install it with: %s\n", tool.InstallInstructions())
	return false, nil
}

// ToolInfo returns the Tool interface and whether it's installed for a given tool name
func ToolInfo(name string) (Tool, bool, error) {
	tool, err := GetTool(name)
	if err != nil {
		return nil, false, err
	}
	return tool, tool.IsInstalled(), nil
}

// =============================================================================
// Tool implementations
// =============================================================================

// --- Brew-installable tools ---

type orasTool struct{}

func (o *orasTool) Name() string              { return "oras" }
func (o *orasTool) Category() ToolCategory    { return CategoryBrew }
func (o *orasTool) IsInstalled() bool          { return exec.Command("oras", "version").Run() == nil }
func (o *orasTool) InstallInstructions() string { return "brew install oras" }
func (o *orasTool) Description() string         { return "OCI artifact registry client" }

type syftTool struct{}

func (s *syftTool) Name() string              { return "syft" }
func (s *syftTool) Category() ToolCategory    { return CategoryBrew }
func (s *syftTool) IsInstalled() bool          { return exec.Command("syft", "version").Run() == nil }
func (s *syftTool) InstallInstructions() string { return "brew install anchore/syft/syft" }
func (s *syftTool) Description() string         { return "SBOM generation tool" }

type cosignTool struct{}

func (c *cosignTool) Name() string              { return "cosign" }
func (c *cosignTool) Category() ToolCategory    { return CategoryBrew }
func (c *cosignTool) IsInstalled() bool          { _, err := exec.LookPath("cosign"); return err == nil }
func (c *cosignTool) InstallInstructions() string { return "brew install sigstore/tap/cosign" }
func (c *cosignTool) Description() string         { return "Sigstore container signing and verification" }

type fluxTool struct{}

func (f *fluxTool) Name() string              { return "flux" }
func (f *fluxTool) Category() ToolCategory    { return CategoryBrew }
func (f *fluxTool) IsInstalled() bool          { _, err := exec.LookPath("flux"); return err == nil }
func (f *fluxTool) InstallInstructions() string { return "brew install fluxcd/tap/flux" }
func (f *fluxTool) Description() string         { return "GitOps continuous delivery tool" }

type podmanTool struct{}

func (p *podmanTool) Name() string              { return "podman" }
func (p *podmanTool) Category() ToolCategory    { return CategoryBrew }
func (p *podmanTool) IsInstalled() bool          { _, err := exec.LookPath("podman"); return err == nil }
func (p *podmanTool) InstallInstructions() string { 
	// Detect architecture to provide correct podman version guidance
	arch := getSystemArch()
	if arch == "x86_64" {
		// Intel Mac: podman 6+ doesn't work, need 5.x
		return "brew install podman@5 || (brew extract podman /opt/homebrew/Cellar/podman@5 5.1.2 && brew link podman@5)"
	}
	// Apple Silicon: latest podman works
	return "brew install podman"
}
func (p *podmanTool) Description() string         { return "Container engine (Docker alternative)" }

// getSystemArch returns the system architecture
func getSystemArch() string {
	// Try uname -m
	cmd := exec.Command("uname", "-m")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	arch := strings.TrimSpace(string(output))
	// Map x86_64 to amd64 for consistency
	if arch == "x86_64" {
		return "amd64"
	}
	return arch
}

type kubectlTool struct{}

func (k *kubectlTool) Name() string              { return "kubectl" }
func (k *kubectlTool) Category() ToolCategory    { return CategoryBrew }
func (k *kubectlTool) IsInstalled() bool          { return exec.Command("kubectl", "version", "--client").Run() == nil }
func (k *kubectlTool) InstallInstructions() string { return "brew install kubectl" }
func (k *kubectlTool) Description() string         { return "Kubernetes command-line tool" }

type minikubeTool struct{}

func (m *minikubeTool) Name() string              { return "minikube" }
func (m *minikubeTool) Category() ToolCategory    { return CategoryBrew }
func (m *minikubeTool) IsInstalled() bool          { return exec.Command("minikube", "version").Run() == nil }
func (m *minikubeTool) InstallInstructions() string { return "brew install minikube" }
func (m *minikubeTool) Description() string         { return "Local Kubernetes cluster" }

type notationTool struct{}

func (n *notationTool) Name() string              { return "notation" }
func (n *notationTool) Category() ToolCategory    { return CategoryBrew }
func (n *notationTool) IsInstalled() bool          { _, err := exec.LookPath("notation"); return err == nil }
func (n *notationTool) InstallInstructions() string { return "brew install notation" }
func (n *notationTool) Description() string         { return "Notary v2 container signing and verification (notation)" }

// --- Environment prerequisites ---

type xcodeCLTTool struct{}

func (x *xcodeCLTTool) Name() string           { return "xcode-clt" }
func (x *xcodeCLTTool) Category() ToolCategory { return CategoryEnvironment }
func (x *xcodeCLTTool) IsInstalled() bool {
	// Check if xcode-select is available and has a valid installation
	cmd := exec.Command("xcode-select", "-p")
	return cmd.Run() == nil
}
func (x *xcodeCLTTool) InstallInstructions() string { return "xcode-select --install" }
func (x *xcodeCLTTool) Description() string         { return "Xcode Command Line Tools" }

type sshKeyTool struct{}

func (s *sshKeyTool) Name() string           { return "ssh-key" }
func (s *sshKeyTool) Category() ToolCategory { return CategoryEnvironment }
func (s *sshKeyTool) IsInstalled() bool {
	// Check if there's at least one SSH key in the default location
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	sshDir := filepath.Join(homeDir, ".ssh")
	entries, err := os.ReadDir(sshDir)
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) == ".pub" {
			return true
		}
	}
	return false
}
func (s *sshKeyTool) InstallInstructions() string { 
	return "ssh-keygen -t ed25519 -C \"your_email@example.com\"" 
}
func (s *sshKeyTool) Description() string { return "SSH key for GitHub/GitLab access" }

// --- Cluster-side tools (verify only) ---

type kserveTool struct{}

func (k *kserveTool) Name() string              { return "kserve" }
func (k *kserveTool) Category() ToolCategory    { return CategoryCluster }
func (k *kserveTool) IsInstalled() bool          {
	// Check if kserve is running in the cluster
	cmd := exec.Command("kubectl", "get", "deployments", "-A", "-o", "name")
	if cmd.Run() != nil {
		return false
	}
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(out), "kserve-controller-manager")
}
func (k *kserveTool) InstallInstructions() string { 
	return "See README for cluster bootstrap instructions (Flux installation)" 
}
func (k *kserveTool) Description() string { return "Model serving operator for Kubernetes" }

type certManagerTool struct{}

func (c *certManagerTool) Name() string              { return "cert-manager" }
func (c *certManagerTool) Category() ToolCategory    { return CategoryCluster }
func (c *certManagerTool) IsInstalled() bool          {
	cmd := exec.Command("kubectl", "get", "deployments", "-A", "-o", "name")
	if cmd.Run() != nil {
		return false
	}
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(out), "cert-manager")
}
func (c *certManagerTool) InstallInstructions() string { 
	return "See README for cluster bootstrap instructions (Flux installation)" 
}
func (c *certManagerTool) Description() string { return "TLS certificates manager for Kubernetes" }

type metricsServerTool struct{}

func (m *metricsServerTool) Name() string              { return "metrics-server" }
func (m *metricsServerTool) Category() ToolCategory    { return CategoryCluster }
func (m *metricsServerTool) IsInstalled() bool          {
	cmd := exec.Command("kubectl", "get", "deployments", "-A", "-o", "name")
	if cmd.Run() != nil {
		return false
	}
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(out), "metrics-server")
}
func (m *metricsServerTool) InstallInstructions() string { 
	return "See README for cluster bootstrap instructions (Flux installation)" 
}
func (m *metricsServerTool) Description() string { return "Resource metrics server for HPA" }

// GetTool returns a specific tool by name
func GetTool(name string) (Tool, error) {
	for _, tool := range AllTools() {
		if tool.Name() == name {
			return tool, nil
		}
	}
	return nil, fmt.Errorf("unknown tool: %s", name)
}

// DoctorSummary returns a summary of the doctor report
func (r *DoctorReport) DoctorSummary() (int, int, int) {
	brewMissing := 0
	brewTotal := 0
	otherMissing := 0
	
	for _, result := range r.Results {
		if result.Tool.Category() == CategoryBrew {
			brewTotal++
			if !result.Installed {
				brewMissing++
			}
		} else if !result.Installed {
			otherMissing++
		}
	}
	
	return brewMissing, brewTotal, otherMissing
}

// AllInstalled returns true if all tools are installed
func (r *DoctorReport) AllInstalled() bool {
	for _, result := range r.Results {
		if !result.Installed {
			return false
		}
	}
	return true
}

// MissingTools returns a list of tools that are not installed
func (r *DoctorReport) MissingTools() []Tool {
	var missing []Tool
	for _, result := range r.Results {
		if !result.Installed {
			missing = append(missing, result.Tool)
		}
	}
	return missing
}

// MissingBrewTools returns a list of brew-installable tools that are not installed
func (r *DoctorReport) MissingBrewTools() []Tool {
	var missing []Tool
	for _, result := range r.Results {
		if !result.Installed && result.Tool.Category() == CategoryBrew {
			missing = append(missing, result.Tool)
		}
	}
	return missing
}
