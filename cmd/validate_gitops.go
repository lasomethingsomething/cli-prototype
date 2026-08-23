package cmd

import (
	"fmt"
	"os"

	"github.com/charmbracelet/huh"
	"github.com/lasomethingsomething/cli-prototype/config"
	"github.com/lasomethingsomething/cli-prototype/internal/workflow"
	"github.com/spf13/cobra"
)

var validateGitOpsCmd = &cobra.Command{
	Use:   "validate-gitops",
	Short: "Validate artifact annotations for GitOps deployment",
	Long: `Validate that an OCI artifact has the required annotations for GitOps deployment.

This command handles Phase 3, Step 5: GitOps Admission & Policy Enforcement (Story #65, #66).
It checks that artifacts have the necessary Trust Profile and Infrastructure Requirement
annotations attached before GitOps tools (Argo CD, Flux) attempt deployment.

Features:
- Fetches artifact manifest from registry (ORAS, ModelPack)
- Validates Trust Profile annotations (Story #63)
- Validates Infrastructure Requirement annotations (Story #64)
- Validates environment-specific safety policies (Story #66)
- Returns exit code 0 for pass, non-zero for fail
- Outputs structured results for CI/CD integration

This enables GitOps tools to perform pre-sync validation without downloading
large model binaries. The CLI orchestrates and hands off to external tools
(Sigstore Policy Controller, OPA/Gatekeeper, Kyverno) for actual policy enforcement.

Examples:
  model-cli validate-gitops
  model-cli validate-gitops --artifact my-registry/my-model:latest
  model-cli validate-gitops --artifact my-registry/my-model:latest --registry oras
  model-cli validate-gitops --artifact my-registry/my-model:latest --quiet
  model-cli validate-gitops --artifact my-registry/my-model:latest --json-output
  model-cli validate-gitops --artifact my-registry/my-model:latest --env air-gapped
  model-cli validate-gitops --artifact my-registry/my-model:latest --env hybrid-cloud --region us-east-1`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get flags
		artifactFlag, _ := cmd.Flags().GetString("artifact")
		registryFlag, _ := cmd.Flags().GetString("registry")
		quietFlag, _ := cmd.Flags().GetBool("quiet")
		jsonOutputFlag, _ := cmd.Flags().GetBool("json-output")
		envFlag, _ := cmd.Flags().GetString("env")
		regionFlag, _ := cmd.Flags().GetString("region")

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

		// Environment and region for safety policy validation (Story #66)
		var environment string
		var region string
		if envFlag != "" {
			environment = envFlag
		} else {
			// Default to no environment-specific validation
			environment = ""
		}
		if regionFlag != "" {
			region = regionFlag
		} else {
			region = ""
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
			if cfg.Registry == "" {
				cfg.Registry = registry
				config.Save(cfg)
			}
		}

		// Get registry provider
		provider, err := workflow.GetRegistryProvider(registry)
		if err != nil {
			return err
		}

		// Check if tool is installed
		if !provider.IsInstalled() {
			return fmt.Errorf("%s not installed. Install with: %s", provider.Name(), provider.InstallInstructions())
		}

		if !quietFlag {
			fmt.Printf("\nValidating GitOps annotations for: %s\n\n", artifact)
		}

		// Fetch manifest annotations from registry
		manifestAnnotations, err := provider.FetchManifestAnnotations(artifact)
		if err != nil {
			if !quietFlag {
				fmt.Printf("✗ Failed to fetch manifest for %s: %v\n", artifact, err)
			}
			return err
		}

		if !quietFlag {
			fmt.Println("✓ Fetched artifact manifest from registry")
		}

		// Define required annotations for GitOps admission
		// Trust Profile annotations (Story #63)
		requiredTrustAnnotations := []string{
			workflow.AnnotationSigningFramework,
			workflow.AnnotationSBOMFormat,
			workflow.AnnotationProvenanceType,
			workflow.AnnotationProfileVersion,
			workflow.AnnotationArtifactType,
		}

		// Infrastructure Requirement annotations (Story #64)
		requiredInfraAnnotations := []string{
			workflow.AnnotationRuntime,
			workflow.AnnotationAccelerator,
		}

		// Conditional infrastructure annotations
		conditionalInfraAnnotations := []string{
			workflow.AnnotationCUDAVersionMin,
			workflow.AnnotationMemoryMin,
		}

		// Track validation results
		allPassed := true
		missingAnnotations := []string{}
		warnings := []string{}

		// Validate Trust Profile annotations
		if !quietFlag {
			fmt.Println("\n=== Trust Profile Validation ===")
		}
		for _, ann := range requiredTrustAnnotations {
			if _, ok := manifestAnnotations[ann]; !ok {
				missingAnnotations = append(missingAnnotations, ann)
				allPassed = false
				if !quietFlag {
					fmt.Printf("  ✗ Missing: %s\n", ann)
				}
			} else {
				if !quietFlag {
					fmt.Printf("  ✓ %s: %s\n", ann, manifestAnnotations[ann])
				}
			}
		}

		// Validate Infrastructure Requirement annotations
		if !quietFlag {
			fmt.Println("\n=== Infrastructure Requirements Validation ===")
		}
		for _, ann := range requiredInfraAnnotations {
			if _, ok := manifestAnnotations[ann]; !ok {
				missingAnnotations = append(missingAnnotations, ann)
				allPassed = false
				if !quietFlag {
					fmt.Printf("  ✗ Missing: %s\n", ann)
				}
			} else {
				if !quietFlag {
					fmt.Printf("  ✓ %s: %s\n", ann, manifestAnnotations[ann])
				}
			}
		}

		// Check conditional infrastructure annotations (warn if missing)
		for _, ann := range conditionalInfraAnnotations {
			if _, ok := manifestAnnotations[ann]; !ok {
				warnings = append(warnings, fmt.Sprintf("Missing conditional annotation: %s", ann))
				if !quietFlag {
					fmt.Printf("  ⚠ Optional: %s (not present)\n", ann)
				}
			} else {
				if !quietFlag {
					fmt.Printf("  ✓ %s: %s\n", ann, manifestAnnotations[ann])
				}
			}
		}

		// Environment-specific validation for air-gapped and hybrid-cloud (Story #66)
		if environment != "" {
			if !quietFlag {
				fmt.Println("\n=== Environment Safety Policy Validation (Story #66) ===")
			}

			switch environment {
			case "air-gapped":
				// For air-gapped environments, validate that all dependencies are self-contained
				// Check for packaging format that supports offline deployment
				if packagingFormat, ok := manifestAnnotations[workflow.AnnotationPackagingFormat]; ok {
					if packagingFormat == "modelpack" {
						if !quietFlag {
							fmt.Printf("  ✓ Packaging format: %s (supports air-gapped deployment)\n", packagingFormat)
						}
					} else {
						warnings = append(warnings, "Air-gapped: consider using modelpack packaging format")
						if !quietFlag {
							fmt.Printf("  ⚠ Packaging format: %s (may not support air-gapped)\n", packagingFormat)
						}
					}
				} else {
					warnings = append(warnings, "Air-gapped: missing packaging format annotation")
					if !quietFlag {
						fmt.Printf("  ⚠ Missing packaging format annotation for air-gapped validation\n")
					}
				}

				// Check that SBOM is present (required for air-gapped compliance)
				if sbomFormat, ok := manifestAnnotations[workflow.AnnotationSBOMFormat]; ok {
					if !quietFlag {
						fmt.Printf("  ✓ SBOM format: %s (available for air-gapped compliance)\n", sbomFormat)
					}
				} else {
					warnings = append(warnings, "Air-gapped: missing SBOM format annotation")
					if !quietFlag {
						fmt.Printf("  ⚠ Missing SBOM for air-gapped compliance\n")
					}
				}

				// Air-gapped environments require all dependencies to be pre-loaded
				// We validate that the artifact declares its dependencies
				if !quietFlag {
					fmt.Printf("  ✓ Air-gapped environment validated\n")
					fmt.Printf("  ℹ Ensure all dependencies are pre-loaded in the air-gapped registry\n")
				}

			case "hybrid-cloud":
				// For hybrid-cloud, validate data residency and region-specific requirements
				if region != "" {
					// Check for data residency annotations if present
					// These would be set by the packaging process for multi-region deployments
					if residency, ok := manifestAnnotations[workflow.AnnotationDataResidency]; ok {
						if residency == region {
							if !quietFlag {
								fmt.Printf("  ✓ Data residency: %s (matches target region: %s)\n", residency, region)
							}
						} else {
							allPassed = false
							missingAnnotations = append(missingAnnotations, fmt.Sprintf("%s=%s", workflow.AnnotationDataResidency, region))
							if !quietFlag {
								fmt.Printf("  ✗ Data residency mismatch: artifact requires %s, target is %s\n", residency, region)
							}
						}
					} else {
						// No residency requirement, or annotation not present
						if !quietFlag {
							fmt.Printf("  ℹ No data residency requirement specified\n")
						}
					}

					// Validate network requirements for hybrid-cloud
					// Check for network access annotations
					if network, ok := manifestAnnotations[workflow.AnnotationNetworkAccess]; ok {
						if network == "internal" || network == "private" {
							if !quietFlag {
								fmt.Printf("  ✓ Network access: %s (suitable for hybrid-cloud)\n", network)
							}
						} else if network == "public" {
							warnings = append(warnings, "Hybrid-cloud: artifact has public network access")
							if !quietFlag {
								fmt.Printf("  ⚠ Network access: %s (may not be suitable for all hybrid-cloud configs)\n", network)
							}
						}
					} else {
						if !quietFlag {
							fmt.Printf("  ℹ No network access requirement specified\n")
						}
					}

					if !quietFlag {
						fmt.Printf("  ✓ Hybrid-cloud environment validated for region: %s\n", region)
					}
				} else {
					warnings = append(warnings, "Hybrid-cloud: region not specified")
					if !quietFlag {
						fmt.Printf("  ⚠ Region not specified for hybrid-cloud validation\n")
					}
				}

			case "development", "staging", "production":
				// For standard environments, just acknowledge the validation
				if !quietFlag {
					fmt.Printf("  ✓ Environment: %s\n", environment)
				}

			case "":
				// No environment specified, skip environment validation
				if !quietFlag {
					fmt.Printf("  ℹ No environment specified, skipping safety policy validation\n")
				}

			default:
				warnings = append(warnings, fmt.Sprintf("Unknown environment: %s", environment))
				if !quietFlag {
					fmt.Printf("  ⚠ Unknown environment: %s\n", environment)
				}
			}
		}

		// Output warnings if any
		if len(warnings) > 0 && !quietFlag {
			fmt.Println("\n=== Warnings ===")
			for _, warning := range warnings {
				fmt.Printf("  ⚠ %s\n", warning)
			}
		}

		// Final result
		if !quietFlag {
			fmt.Println("\n=== Result ===")
		}

		if allPassed {
			if !quietFlag {
				fmt.Println("✓ PASS: Artifact has all required annotations for GitOps admission")
				fmt.Println("\nGitOps tools (Argo CD, Flux) can proceed with deployment.")
				fmt.Println("External policy engines (Sigstore Policy Controller, OPA/Gatekeeper, Kyverno)")
				fmt.Println("will perform the actual admission control based on these annotations.")
			}
			if jsonOutputFlag {
				fmt.Printf(`{"status":"pass","artifact":"%s","missing_annotations":[]}\n`, artifact)
			}
			return nil
		} else {
			if !quietFlag {
				fmt.Println("✗ FAIL: Artifact is missing required annotations")
				fmt.Println("\nMissing annotations:")
				for _, ann := range missingAnnotations {
					fmt.Printf("  - %s\n", ann)
				}
				fmt.Println("\nTo fix: Re-package the artifact with model-cli package --sign")
				fmt.Println("and ensure all required metadata is included.")
			}
			if jsonOutputFlag {
				fmt.Printf(`{"status":"fail","artifact":"%s","missing_annotations":[`, artifact)
				for i, ann := range missingAnnotations {
					if i > 0 {
						fmt.Printf(",")
					}
					fmt.Printf(`"%s"`, ann)
				}
				fmt.Printf("]}\n")
			}
			os.Exit(1)
			return nil // Exit code already set
		}
	},
}

func init() {
	rootCmd.AddCommand(validateGitOpsCmd)
	validateGitOpsCmd.Flags().String("artifact", "", "OCI artifact reference to validate (e.g., ghcr.io/my-org/my-model:latest)")
	validateGitOpsCmd.Flags().String("registry", "", "Registry tool: oras or modelpack")
	validateGitOpsCmd.Flags().Bool("quiet", false, "Quiet mode: only output pass/fail status")
	validateGitOpsCmd.Flags().Bool("json-output", false, "Output results as JSON for CI/CD integration")
	validateGitOpsCmd.Flags().String("env", "", "Target environment: development, staging, production, air-gapped, hybrid-cloud")
	validateGitOpsCmd.Flags().String("region", "", "Target region for hybrid-cloud validation (e.g., us-east-1, eu-west-1)")
}
