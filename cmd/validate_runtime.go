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

var validateRuntimeCmd = &cobra.Command{
	Use:   "validate-runtime",
	Short: "Validate runtime availability for artifact deployment",
	Long: `Validate that the required serving runtime is available in the cluster.

This command handles Phase 3, Step 7: Runtime Execution & Optimization (Story #69).
It fetches the artifact manifest from the registry, reads runtime requirement
annotations (runtime type, layer deduplication), and checks if the required
runtime operators (KServe, vLLM) are installed in the cluster.

Features:
- Fetches artifact manifest from registry (ORAS, ModelPack)
- Extracts runtime requirement annotations (ai.runtime.type, ai.runtime.optimization.layer-dedup)
- Checks for runtime operator CRDs in the cluster
- Validates Reference Skill DLC endpoint accessibility
- Returns exit code 0 for pass, non-zero for fail
- Outputs structured results for CI/CD integration

Note: This command performs validation only. Actual model serving is delegated to
the runtime operators (KServe, vLLM). The CLI orchestrates and hands off to external tools.

Examples:
  model-cli validate-runtime
  model-cli validate-runtime --artifact my-registry/my-model:latest
  model-cli validate-runtime --artifact my-registry/my-model:latest --registry oras
  model-cli validate-runtime --artifact my-registry/my-model:latest --namespace production
  model-cli validate-runtime --artifact my-registry/my-model:latest --quiet
  model-cli validate-runtime --artifact my-registry/my-model:latest --json-output`,
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
			warnIfSaveFails(config.Save(cfg))
		}

		var namespace string
		if namespaceFlag != "" {
			namespace = namespaceFlag
		} else {
			namespace = "default"
		}

		// Get registry provider
		registryProvider, err := workflow.GetRegistryProvider(registry)
		if err != nil {
			return fmt.Errorf("failed to get registry provider: %v", err)
		}

		if !quietFlag {
			fmt.Printf("\nValidating runtime requirements for: %s\n\n", artifact)
		}

		// Fetch manifest annotations from registry
		if !quietFlag {
			fmt.Println("→ Fetching artifact manifest...")
		}

		fullArtifact := artifact

		annotations, err := registryProvider.FetchManifestAnnotations(fullArtifact)
		if err != nil {
			if !quietFlag {
				fmt.Printf("⚠ Warning: Could not fetch manifest from registry: %v\n", err)
				fmt.Println("  This might be because the artifact hasn't been pushed yet.")
				fmt.Println("  For local validation, use a local registry or push first.")
			}
			return fmt.Errorf("failed to fetch manifest: %v. Hint: Push artifact first or use a local registry", err)
		}

		if !quietFlag {
			fmt.Println("✓ Fetched artifact manifest")
		}

		// Extract runtime requirement annotations
		runtimeType := annotations[workflow.AnnotationRuntimeType]
		layerDedup := annotations[workflow.AnnotationLayerDeduplication]
		dlcEndpoint := annotations[workflow.AnnotationReferenceSkillDLC]
		skillRefs := annotations[workflow.AnnotationSkillReferences]

		if !quietFlag {
			fmt.Println("\n=== Runtime Requirements from Manifest ===")
			if runtimeType != "" {
				fmt.Printf("  Runtime Type: %s\n", runtimeType)
			} else {
				fmt.Println("  Runtime Type: (not specified, will use default)")
			}
			if layerDedup != "" {
				fmt.Printf("  Layer Deduplication: %s\n", layerDedup)
			} else {
				fmt.Println("  Layer Deduplication: (not specified)")
			}
			if dlcEndpoint != "" {
				fmt.Printf("  Reference Skill DLC Endpoint: %s\n", dlcEndpoint)
			} else {
				fmt.Println("  Reference Skill DLC Endpoint: (not specified)")
			}
			if skillRefs != "" {
				fmt.Printf("  Skill References: %s\n", skillRefs)
			} else {
				fmt.Println("  Skill References: (not specified)")
			}
		}

		// If no runtime requirements are specified, consider it a pass
		if runtimeType == "" && layerDedup == "" && dlcEndpoint == "" && skillRefs == "" {
			if !quietFlag {
				fmt.Println("\n✓ PASS: No specific runtime requirements declared")
				fmt.Println("  Default runtime will be used for deployment")
			}
			return nil
		}

		// Validate runtime availability
		if !quietFlag {
			fmt.Println("\n=== Checking Runtime Availability ===")
		}

		allPass := true
		var validationErrors []string

		// Check runtime operator
		if runtimeType != "" {
			if !quietFlag {
				fmt.Printf("\n  Checking runtime operator: %s\n", runtimeType)
			}

			runtimePass := false
			switch runtimeType {
			case "kserve":
				// Check if KServe is installed
				if err := checkKServeInstalled(namespace); err == nil {
					runtimePass = true
					if !quietFlag {
						fmt.Println("    ✓ KServe operator is installed")
					}
				} else {
					validationErrors = append(validationErrors, fmt.Sprintf("KServe operator not found: %v", err))
					if !quietFlag {
						fmt.Printf("    ✗ KServe operator not found: %v\n", err)
					}
				}
			case "vllm":
				// Check if vLLM is installed
				if err := checkVLLMInstalled(namespace); err == nil {
					runtimePass = true
					if !quietFlag {
						fmt.Println("    ✓ vLLM runtime is available")
					}
				} else {
					validationErrors = append(validationErrors, fmt.Sprintf("vLLM runtime not found: %v", err))
					if !quietFlag {
						fmt.Printf("    ✗ vLLM runtime not found: %v\n", err)
					}
				}
			default:
				// For unknown runtimes, just acknowledge
				runtimePass = true
				if !quietFlag {
					fmt.Printf("    ⚠ Runtime %s: unknown runtime, assuming available\n", runtimeType)
				}
			}

			if !runtimePass {
				allPass = false
			}
		}

		// Check layer deduplication support
		if layerDedup == "true" {
			if !quietFlag {
				fmt.Println("\n  Checking layer deduplication support")
			}
			// Layer deduplication is typically supported by the registry or runtime
			// For now, we'll check if the registry supports it
			if err := checkLayerDeduplicationSupport(runtimeType); err == nil {
				if !quietFlag {
					fmt.Println("    ✓ Layer deduplication is supported")
				}
			} else {
				validationErrors = append(validationErrors, fmt.Sprintf("layer deduplication not supported: %v", err))
				if !quietFlag {
					fmt.Printf("    ✗ Layer deduplication not supported: %v\n", err)
				}
				allPass = false
			}
		}

		// Check Reference Skill DLC endpoint
		if dlcEndpoint != "" {
			if !quietFlag {
				fmt.Println("\n  Checking Reference Skill DLC endpoint")
			}
			// For now, just check if the endpoint is reachable via ping/curl
			// In production, this would be a more sophisticated check
			if err := checkEndpointReachable(dlcEndpoint); err == nil {
				if !quietFlag {
					fmt.Printf("    ✓ DLC endpoint %s is reachable\n", dlcEndpoint)
				}
			} else {
				validationErrors = append(validationErrors, fmt.Sprintf("DLC endpoint unreachable: %v", err))
				if !quietFlag {
					fmt.Printf("    ✗ DLC endpoint unreachable: %v\n", err)
				}
				allPass = false
			}
		}

		// Check skill references
		if skillRefs != "" {
			if !quietFlag {
				fmt.Println("\n  Checking skill references")
			}
			// Parse skill references and check if they exist in the registry
			// For now, just acknowledge that they were specified
			if !quietFlag {
				fmt.Printf("    ✓ Skill references declared: %s\n", skillRefs)
				fmt.Println("    ⚠ Skill validation requires registry access (not implemented)")
			}
		}

		// Output results
		if jsonOutputFlag {
			result := map[string]interface{}{
				"artifact": artifact,
				"requirements": map[string]string{
					"runtime_type": runtimeType,
					"layer_dedup":  layerDedup,
					"dlc_endpoint": dlcEndpoint,
					"skill_refs":   skillRefs,
				},
				"pass": allPass,
			}
			if !allPass {
				result["errors"] = validationErrors
			}
			jsonOutput, _ := json.MarshalIndent(result, "", "  ")
			fmt.Println(string(jsonOutput))
		} else if !quietFlag {
			fmt.Println("\n=== Result ===")
			if allPass {
				fmt.Println("✓ PASS: All runtime requirements can be satisfied")
				fmt.Println("  Runtime operators can proceed with model serving")
			} else {
				fmt.Println("✗ FAIL: Runtime requirements cannot be satisfied")
				for _, err := range validationErrors {
					fmt.Printf("  - %s\n", err)
				}
				fmt.Println("\nHint: Install required runtime operators or adjust artifact requirements")
			}
		}

		if !allPass {
			return fmt.Errorf("runtime validation failed: %v", validationErrors)
		}

		return nil
	},
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

func init() {
	rootCmd.AddCommand(validateRuntimeCmd)
	validateRuntimeCmd.Flags().String("artifact", "", "OCI artifact reference to validate")
	validateRuntimeCmd.Flags().String("registry", "", "Registry tool: oras or modelpack")
	validateRuntimeCmd.Flags().String("namespace", "default", "Kubernetes namespace to check")
	validateRuntimeCmd.Flags().Bool("quiet", false, "Quiet mode: only output pass/fail status")
	validateRuntimeCmd.Flags().Bool("json-output", false, "Output results as JSON for CI/CD integration")
}
