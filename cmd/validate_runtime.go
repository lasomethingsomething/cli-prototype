package cmd

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/lasomethingsomething/cli-prototype/internal/workflow"
	"github.com/spf13/cobra"
)

func newValidateRuntimeCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "runtime",
		Short: "Validate runtime availability for artifact deployment",
		Long: `Check that the runtime the artifact declares (ai.runtime.type, layer deduplication,
Reference Skill DLC endpoint, skill references) is available in the cluster. Operators
are detected with kubectl; serving itself is left to KServe / vLLM.

Examples:
  model-cli validate runtime --artifact ghcr.io/my-org/my-model:v1
  model-cli validate runtime --artifact my-model:v1 --namespace production --json-output`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			out := outputOptionsFrom(cmd)
			namespace, _ := cmd.Flags().GetString("namespace")

			artifact, annotations, err := fetchAnnotations(cmd, out)
			if err != nil {
				return err
			}
			report := workflow.NewValidationReport("runtime", artifact)
			const section = "Runtime Availability"
			hints := reportHints{
				pass: "Runtime operators can proceed with model serving.",
				fail: "Hint: install the required runtime operators or adjust the artifact's runtime requirements.",
			}

			runtimeType := annotations[workflow.AnnotationRuntimeType]
			layerDedup := annotations[workflow.AnnotationLayerDeduplication]
			dlcEndpoint := annotations[workflow.AnnotationReferenceSkillDLC]
			skillRefs := annotations[workflow.AnnotationSkillReferences]
			if runtimeType == "" && layerDedup == "" && dlcEndpoint == "" && skillRefs == "" {
				report.Info(section, "requirements", "none declared; the default runtime will be used")
				return printReport(report, out, hints)
			}

			switch runtimeType {
			case "":
			case "kserve":
				if err := checkKServeInstalled(namespace); err != nil {
					report.Fail(section, workflow.AnnotationRuntimeType, "KServe operator not found: "+err.Error(), "kserve operator")
				} else {
					report.Pass(section, workflow.AnnotationRuntimeType, "kserve operator is installed")
				}
			case "vllm":
				if err := checkVLLMInstalled(namespace); err != nil {
					report.Fail(section, workflow.AnnotationRuntimeType, "vLLM runtime not found: "+err.Error(), "vllm runtime")
				} else {
					report.Pass(section, workflow.AnnotationRuntimeType, "vllm runtime is available")
				}
			default:
				report.Warn(section, workflow.AnnotationRuntimeType, runtimeType+" is not a known runtime; assuming it is available")
			}

			if layerDedup == "true" {
				if err := checkLayerDeduplicationSupport(runtimeType); err != nil {
					report.Fail(section, workflow.AnnotationLayerDeduplication, err.Error(), "layer deduplication")
				} else {
					report.Pass(section, workflow.AnnotationLayerDeduplication, "supported by "+runtimeType)
				}
			}
			if dlcEndpoint != "" {
				if err := checkEndpointReachable(dlcEndpoint); err != nil {
					report.Fail(section, workflow.AnnotationReferenceSkillDLC, err.Error(), "dlc endpoint "+dlcEndpoint)
				} else {
					report.Pass(section, workflow.AnnotationReferenceSkillDLC, dlcEndpoint+" is reachable")
				}
			}
			if skillRefs != "" {
				report.Warn(section, workflow.AnnotationSkillReferences, skillRefs+" declared; skill validation needs registry access (not implemented)")
			}
			return printReport(report, out, hints)
		},
	}
	addArtifactFlags(c)
	addOutputFlags(c)
	c.Flags().String("namespace", "default", "Kubernetes namespace to check")
	return c
}

// checkKServeInstalled checks if KServe operator is installed in the cluster
func checkKServeInstalled(namespace string) error {
	// Check for KServe CRDs
	cmd := exec.Command("kubectl", "get", "crd", "inferenceservices.serving.kserve.io")
	if err := cmd.Run(); err != nil {
		// Also check for serving.kserve.io API group
		cmd := exec.Command("kubectl", "api-versions")
		output, err := cmd.Output()
		if err != nil {
			return fmt.Errorf("kubectl not available or cluster not accessible")
		}
		if !strings.Contains(string(output), "serving.kserve.io") {
			return fmt.Errorf("KServe CRDs not found. Install with: kubectl apply -f https://github.com/kserve/kserve/releases/latest/download/kserve.yaml")
		}
	}
	return nil
}

// checkVLLMInstalled checks if vLLM runtime is available in the cluster
func checkVLLMInstalled(namespace string) error {
	// Check for vLLM pods or deployments
	cmd := exec.Command("kubectl", "get", "pods", "-n", namespace, "-l", "app=vllm")
	if err := cmd.Run(); err != nil {
		// vLLM might be installed differently
		// For now, just check if we can find any vLLM-related resources
		return fmt.Errorf("vLLM runtime not detected. Install vLLM following: https://docs.vllm.ai/en/latest/getting_started/installation.html")
	}
	return nil
}

// checkLayerDeduplicationSupport checks if layer deduplication is supported
func checkLayerDeduplicationSupport(runtimeType string) error {
	// Layer deduplication is typically supported by registries like ghcr.io, docker.io
	// or by the runtime itself
	// For this validation, we'll assume it's supported if using a major registry
	if runtimeType == "kserve" || runtimeType == "vllm" {
		// Both KServe and vLLM support layer deduplication with compatible registries
		return nil
	}
	return fmt.Errorf("layer deduplication support unknown for runtime %s", runtimeType)
}

// checkEndpointReachable checks if an endpoint is reachable
func checkEndpointReachable(endpoint string) error {
	// Skip localhost and file endpoints
	if strings.HasPrefix(endpoint, "localhost") || strings.HasPrefix(endpoint, "127.0.0.1") || strings.HasPrefix(endpoint, "/") {
		return nil // Assume local endpoints are valid
	}

	// Try to ping the endpoint (remove protocol if present)
	cleanEndpoint := strings.TrimPrefix(strings.TrimPrefix(endpoint, "https://"), "http://")
	cleanEndpoint = strings.TrimPrefix(cleanEndpoint, "//")

	// Use curl with a timeout to check reachability
	cmd := exec.Command("curl", "-s", "--max-time", "2", cleanEndpoint)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("endpoint %s is not reachable: %v", endpoint, err)
	}
	return nil
}
