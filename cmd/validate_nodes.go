package cmd

import (
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/lasomethingsomething/cli-prototype/internal/workflow"
	"github.com/spf13/cobra"
)

func newValidateNodesCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "nodes",
		Short: "Validate cluster nodes match artifact hardware requirements",
		Long: `Check that the cluster has nodes satisfying the artifact's node requirement
annotations (ai.node.gpu.type, ai.node.vram.min, ai.node.gpu.topology). Nodes are read
with kubectl; scheduling itself is left to Kubernetes.

Examples:
  model-cli validate nodes --artifact ghcr.io/my-org/my-model:v1
  model-cli validate nodes --artifact my-model:v1 --namespace production --json-output`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			out := outputOptionsFrom(cmd)
			namespace, _ := cmd.Flags().GetString("namespace")

			artifact, annotations, err := fetchAnnotations(cmd, out)
			if err != nil {
				return err
			}
			report := workflow.NewValidationReport("nodes", artifact)
			const section = "Node Requirements"
			hints := reportHints{
				pass: "Kubernetes scheduler can proceed with deployment.",
				fail: "Hint: add nodes with matching hardware or adjust the artifact's node requirements.",
			}

			gpuType := annotations[workflow.AnnotationGPUType]
			vramMin := annotations[workflow.AnnotationVRAMMin]
			gpuTopology := annotations[workflow.AnnotationGPUTopology]
			if gpuType == "" && vramMin == "" && gpuTopology == "" {
				report.Info(section, "requirements", "none declared; Kubernetes will schedule to any available node")
				return printReport(report, out, hints)
			}

			nodes, err := getClusterNodes(namespace)
			if err != nil {
				return fmt.Errorf("failed to query cluster nodes: %v (ensure kubectl is configured and you have access to the cluster)", err)
			}
			report.Info(section, "cluster", fmt.Sprintf("%d node(s) checked in namespace %q", len(nodes), namespace))

			if gpuType != "" {
				if node := findNode(nodes, func(n ClusterNode) bool { return n.GPUType == gpuType }); node != nil {
					report.Pass(section, workflow.AnnotationGPUType, fmt.Sprintf("%s (node %s)", gpuType, node.Name))
				} else {
					report.Fail(section, workflow.AnnotationGPUType, "no node with GPU type "+gpuType, "gpu type "+gpuType)
				}
			}
			if vramMin != "" {
				// Simple string comparison for now; in production, parse and compare units.
				if node := findNode(nodes, func(n ClusterNode) bool { return n.VRAM >= vramMin }); node != nil {
					report.Pass(section, workflow.AnnotationVRAMMin, fmt.Sprintf("%s (node %s has %s)", vramMin, node.Name, node.VRAM))
				} else {
					report.Fail(section, workflow.AnnotationVRAMMin, "no node with vRAM >= "+vramMin, "vram >= "+vramMin)
				}
			}
			if gpuTopology != "" {
				if node := findNode(nodes, func(n ClusterNode) bool { return n.GPUTopology == gpuTopology }); node != nil {
					report.Pass(section, workflow.AnnotationGPUTopology, fmt.Sprintf("%s (node %s)", gpuTopology, node.Name))
				} else {
					report.Fail(section, workflow.AnnotationGPUTopology, "no node with GPU topology "+gpuTopology, "gpu topology "+gpuTopology)
				}
			}
			return printReport(report, out, hints)
		},
	}
	addArtifactFlags(c)
	addOutputFlags(c)
	c.Flags().String("namespace", "default", "Kubernetes namespace to check")
	return c
}

// findNode returns the first node matching pred, or nil.
func findNode(nodes []ClusterNode, pred func(ClusterNode) bool) *ClusterNode {
	for i := range nodes {
		if pred(nodes[i]) {
			return &nodes[i]
		}
	}
	return nil
}

// ClusterNode represents a Kubernetes node with GPU information
type ClusterNode struct {
	Name        string
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
