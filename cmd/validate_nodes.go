package cmd

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/lasomethingsomething/cli-prototype/config"
	"github.com/lasomethingsomething/cli-prototype/internal/workflow"
	"github.com/spf13/cobra"
)

var validateNodesCmd = &cobra.Command{
	Use:   "validate-nodes",
	Short: "Validate cluster nodes match artifact hardware requirements",
	Long: `Validate that cluster nodes have the required hardware to run an OCI artifact.

This command handles Phase 3, Step 6: Infrastructure & Resource Orchestration (Story #68).
It fetches the artifact manifest from the registry, reads node requirement annotations
(GPU type, vRAM minimum, GPU topology), and checks cluster node labels to ensure
compatibility before Kubernetes schedules the workload.

Features:
- Fetches artifact manifest from registry (ORAS, ModelPack)
- Extracts node requirement annotations (ai.node.gpu.type, ai.node.vram.min, ai.node.gpu.topology)
- Queries cluster nodes and checks labels against requirements
- Returns exit code 0 for pass, non-zero for fail
- Outputs structured results for CI/CD integration

Note: This command performs validation only. Actual pod scheduling is delegated to
Kubernetes scheduler. The CLI orchestrates and hands off to external tools.

Examples:
  model-cli validate-nodes
  model-cli validate-nodes --artifact my-registry/my-model:latest
  model-cli validate-nodes --artifact my-registry/my-model:latest --registry oras
  model-cli validate-nodes --artifact my-registry/my-model:latest --namespace production
  model-cli validate-nodes --artifact my-registry/my-model:latest --quiet
  model-cli validate-nodes --artifact my-registry/my-model:latest --json-output`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get flags
		artifactFlag, _ := cmd.Flags().GetString("artifact")
		registryFlag, _ := cmd.Flags().GetString("registry")
		namespaceFlag, _ := cmd.Flags().GetString("namespace")
		quietFlag, _ := cmd.Flags().GetBool("quiet")
		jsonOutputFlag, _ := cmd.Flags().GetBool("json-output")

		// Interactive prompts if not provided via flags
		var artifact string
		if artifactFlag != "" {
			artifact = artifactFlag
		} else {
			if err := huh.NewInput().
				Title("Artifact to validate:").
				Description("The OCI artifact reference (e.g., ghcr.io/my-org/my-model:latest)").
				Value(&artifact).
				Run(); err != nil {
				return err
			}
		}

		var registry string
		if registryFlag != "" {
			registry = registryFlag
		} else {
			cfg := config.Load()
			if cfg.Registry != "" {
				registry = cfg.Registry
			} else {
				if err := huh.NewSelect[string]().
					Title("Registry tool:").
					Description("Choose how to fetch the artifact manifest").
					Options(huh.NewOptions("oras", "modelpack")...).
					Value(&registry).
					Run(); err != nil {
					return err
				}
			}
		}
		// Save config for future use
		if registry != "" {
			cfg := config.Load()
			cfg.Registry = registry
			config.Save(cfg)
		}

		var namespace string
		if namespaceFlag != "" {
			namespace = namespaceFlag
		} else {
			// Default to current kubectl namespace or "default"
			namespace = "default"
		}

		// Get registry provider
		registryProvider, err := workflow.GetRegistryProvider(registry)
		if err != nil {
			return fmt.Errorf("failed to get registry provider: %v", err)
		}

		if !quietFlag {
			fmt.Printf("\nValidating node requirements for: %s\n\n", artifact)
		}

		// Fetch manifest annotations from registry
		if !quietFlag {
			fmt.Println("→ Fetching artifact manifest...")
		}

		fullArtifact := artifact
		// If artifact doesn't include registry, prepend the configured registry URL
		if !strings.Contains(artifact, "/") && !strings.Contains(artifact, "::") {
			// For now, we'll try to fetch as-is; if it fails, we'll provide guidance
			fullArtifact = artifact
		}

		annotations, err := registryProvider.FetchManifestAnnotations(fullArtifact)
		if err != nil {
			if !quietFlag {
				fmt.Printf("⚠ Warning: Could not fetch manifest from registry: %v\n", err)
				fmt.Println("  This might be because the artifact hasn't been pushed yet.")
				fmt.Println("  For local validation, use a local registry or push first.")
			}
			// For Story #68, we'll check if we can validate from a local manifest
			// For now, return an error
			return fmt.Errorf("failed to fetch manifest: %v. Hint: Push artifact first or use a local registry", err)
		}

		if !quietFlag {
			fmt.Println("✓ Fetched artifact manifest")
		}

		// Extract node requirement annotations
		gpuType := annotations[workflow.AnnotationGPUType]
		vramMin := annotations[workflow.AnnotationVRAMMin]
		gpuTopology := annotations[workflow.AnnotationGPUTopology]

		if !quietFlag {
			fmt.Println("\n=== Node Requirements from Manifest ===")
			if gpuType != "" {
				fmt.Printf("  GPU Type: %s\n", gpuType)
			} else {
				fmt.Println("  GPU Type: (not specified)")
			}
			if vramMin != "" {
				fmt.Printf("  vRAM Minimum: %s\n", vramMin)
			} else {
				fmt.Println("  vRAM Minimum: (not specified)")
			}
			if gpuTopology != "" {
				fmt.Printf("  GPU Topology: %s\n", gpuTopology)
			} else {
				fmt.Println("  GPU Topology: (not specified)")
			}
		}

		// If no node requirements are specified, consider it a pass
		if gpuType == "" && vramMin == "" && gpuTopology == "" {
			if !quietFlag {
				fmt.Println("\n✓ PASS: No specific node requirements declared")
				fmt.Println("  Kubernetes will schedule to any available node")
			}
			return nil
		}

		// Query cluster nodes
		if !quietFlag {
			fmt.Println("\n=== Checking Cluster Nodes ===")
		}

		nodes, err := getClusterNodes(namespace)
		if err != nil {
			if !quietFlag {
				fmt.Printf("⚠ Warning: Could not query cluster nodes: %v\n", err)
				fmt.Println("  Ensure kubectl is configured and you have access to the cluster.")
			}
			return fmt.Errorf("failed to query cluster nodes: %v", err)
		}

		if !quietFlag {
			fmt.Printf("  Found %d node(s) in namespace '%s'\n", len(nodes), namespace)
		}

		// Validate each requirement
		allPass := true
		var validationErrors []string

		// Check GPU Type
		if gpuType != "" {
			if !quietFlag {
				fmt.Printf("\n  Checking GPU Type: %s\n", gpuType)
			}
			gpuTypePass := false
			for _, node := range nodes {
				if node.GPUType == gpuType {
					gpuTypePass = true
					if !quietFlag {
						fmt.Printf("    ✓ Node %s has GPU type %s\n", node.Name, node.GPUType)
					}
					break
				}
			}
			if !gpuTypePass {
				allPass = false
				validationErrors = append(validationErrors, fmt.Sprintf("no nodes with GPU type %s", gpuType))
				if !quietFlag {
					fmt.Printf("    ✗ No nodes with GPU type %s\n", gpuType)
				}
			}
		}

		// Check vRAM Minimum
		if vramMin != "" {
			if !quietFlag {
				fmt.Printf("\n  Checking vRAM Minimum: %s\n", vramMin)
			}
			vramPass := false
			for _, node := range nodes {
				// Simple string comparison for now; in production, parse and compare
				if node.VRAM >= vramMin {
					vramPass = true
					if !quietFlag {
						fmt.Printf("    ✓ Node %s has vRAM %s (>= %s)\n", node.Name, node.VRAM, vramMin)
					}
					break
				}
			}
			if !vramPass {
				allPass = false
				validationErrors = append(validationErrors, fmt.Sprintf("no nodes with vRAM >= %s", vramMin))
				if !quietFlag {
					fmt.Printf("    ✗ No nodes with vRAM >= %s\n", vramMin)
				}
			}
		}

		// Check GPU Topology
		if gpuTopology != "" {
			if !quietFlag {
				fmt.Printf("\n  Checking GPU Topology: %s\n", gpuTopology)
			}
			topologyPass := false
			for _, node := range nodes {
				// Check if node's GPU topology matches requirement
				// For now, simple string match; in production, parse topology (e.g., 8xH100 = 8 GPUs of type H100)
				if node.GPUTopology == gpuTopology {
					topologyPass = true
					if !quietFlag {
						fmt.Printf("    ✓ Node %s has GPU topology %s\n", node.Name, node.GPUTopology)
					}
					break
				}
			}
			if !topologyPass {
				allPass = false
				validationErrors = append(validationErrors, fmt.Sprintf("no nodes with GPU topology %s", gpuTopology))
				if !quietFlag {
					fmt.Printf("    ✗ No nodes with GPU topology %s\n", gpuTopology)
				}
			}
		}

		// Output results
		if jsonOutputFlag {
			result := map[string]interface{}{
				"artifact":       artifact,
				"requirements":   map[string]string{"gpu_type": gpuType, "vram_min": vramMin, "gpu_topology": gpuTopology},
				"nodes_checked":  len(nodes),
				"pass":           allPass,
			}
			if !allPass {
				result["errors"] = validationErrors
			}
			jsonOutput, _ := json.MarshalIndent(result, "", "  ")
			fmt.Println(string(jsonOutput))
		} else if !quietFlag {
			fmt.Println("\n=== Result ===")
			if allPass {
				fmt.Println("✓ PASS: All node requirements can be satisfied")
				fmt.Println("  Kubernetes scheduler can proceed with deployment")
			} else {
				fmt.Println("✗ FAIL: Node requirements cannot be satisfied")
				for _, err := range validationErrors {
					fmt.Printf("  - %s\n", err)
				}
				fmt.Println("\nHint: Add nodes with matching hardware or adjust artifact requirements")
			}
		}

		if !allPass {
			return fmt.Errorf("node validation failed: %v", validationErrors)
		}

		return nil
	},
}

// ClusterNode represents a Kubernetes node with GPU information
type ClusterNode struct {
	Name         string
	GPUType     string
	VRAM        string
	GPUTopology string
}

// getClusterNodes queries the Kubernetes cluster for nodes and extracts GPU information
func getClusterNodes(namespace string) ([]ClusterNode, error) {
	// Use kubectl to get nodes with labels
	// kubectl get nodes -o jsonpath='{.items[*].metadata.name}'
	// kubectl get nodes -o json to get all node details

	cmd := exec.Command("kubectl", "get", "nodes", "-o", "json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run kubectl: %v. Ensure kubectl is installed and configured", err)
	}

	var nodes []ClusterNode

	// Parse JSON output from kubectl
	var nodeList struct {
		Items []struct {
			Metadata struct {
				Name string `json:"name"`
			} `json:"metadata"`
			Status struct {
				Capacity struct {
					GPU string `json:"nvidia.com/gpu"`
				} `json:"capacity"`
				Allocatable struct {
					GPU string `json:"nvidia.com/gpu"`
				} `json:"allocatable"`
				NodeInfo struct {
					KubeletVersion string `json:"kubeletVersion"`
				} `json:"nodeInfo"`
			} `json:"status"`
			Labels map[string]string `json:"metadata.labels"`
		} `json:"items"`
	}

	if err := json.Unmarshal(output, &nodeList); err != nil {
		return nil, fmt.Errorf("failed to parse kubectl output: %v", err)
	}

	// Extract GPU information from node labels and capacity
	for _, item := range nodeList.Items {
		node := ClusterNode{
			Name: item.Metadata.Name,
		}

		// Extract GPU type from labels
		// Common label: nvidia.com/gpu.product
		if gpuProduct, ok := item.Labels["nvidia.com/gpu.product"]; ok {
			node.GPUType = gpuProduct
		} else if gpuType, ok := item.Labels["cloud.google.com/gke-accelerator"]; ok {
			node.GPUType = gpuType
		} else if gpuType, ok := item.Labels["alpha.kubernetes.io/node-gpu-type"]; ok {
			node.GPUType = gpuType
		} else if item.Status.Capacity.GPU != "" {
			// If we have GPU capacity but no specific type, use generic
			node.GPUType = "nvidia-gpu"
		}

		// Extract vRAM from labels or annotations
		// Common: nvidia.com/gpu.memory
		if vram, ok := item.Labels["nvidia.com/gpu.memory"]; ok {
			node.VRAM = vram
		} else if vram, ok := item.Labels["cloud.google.com/gke-accelerator-memory"]; ok {
			node.VRAM = vram
		}

		// Extract GPU topology from labels
		// Common: nvidia.com/gpu.count, topology.k8s.io/zone
		if gpuCount, ok := item.Labels["nvidia.com/gpu.count"]; ok {
			node.GPUTopology = gpuCount + "x" + node.GPUType
		}

		nodes = append(nodes, node)
	}

	return nodes, nil
}

func init() {
	rootCmd.AddCommand(validateNodesCmd)
	validateNodesCmd.Flags().String("artifact", "", "OCI artifact reference to validate")
	validateNodesCmd.Flags().String("registry", "", "Registry tool: oras or modelpack")
	validateNodesCmd.Flags().String("namespace", "default", "Kubernetes namespace to check")
	validateNodesCmd.Flags().Bool("quiet", false, "Quiet mode: only output pass/fail status")
	validateNodesCmd.Flags().Bool("json-output", false, "Output results as JSON for CI/CD integration")
}
