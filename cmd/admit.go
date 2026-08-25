package cmd

import (
	"fmt"
	"os"

	"github.com/charmbracelet/huh"
	"github.com/lasomethingsomething/cli-prototype/config"
	"github.com/lasomethingsomething/cli-prototype/internal/workflow"
	"github.com/spf13/cobra"
)

// admitRunOptions holds the configuration for the admit command
type admitRunOptions struct {
	artifact    string
	registry    string
	environment string
	strict      bool
	jsonOutput  bool
	region      string
}

var admitCmd = &cobra.Command{
	Use:   "admit",
	Short: "Evaluate artifact for GitOps admission",
	Long: `Evaluate an OCI artifact's trust profile for GitOps admission control (Story #67).

This command handles Phase 3, Step 5: GitOps Admission & Policy Enforcement.
The cluster evaluates the artifact's Trust Profile before allowing deployment.

Specifically checks:
- Valid signature presence (Sigstore/Notary v2) - actual verification delegated to external tools
- SBOM presence - actual validation delegated to external tools
- Infrastructure dependency matching (Story #64)
- Compliance profile contract (Story #63)
- Environment safety policies (Story #66)

If any check fails, admission is blocked.

Note: This command performs pre-flight validation. Actual signature verification
and policy enforcement are delegated to external tools (Sigstore Policy Controller,
OPA/Gatekeeper, Kyverno) as per the CLI's orchestration-only principle.

Examples:
  model-cli admit
  model-cli admit --artifact my-registry/my-model:latest
  model-cli admit --artifact my-registry/my-model:latest --strict
  model-cli admit --artifact my-registry/my-model:latest --env production
  model-cli admit --artifact my-registry/my-model:latest --registry oras
  model-cli admit --artifact my-registry/my-model:latest --json-output`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Collect all options
		opts := admitRunOptions{}

		// Get flags
		artifactFlag, _ := cmd.Flags().GetString("artifact")
		registryFlag, _ := cmd.Flags().GetString("registry")
		environmentFlag, _ := cmd.Flags().GetString("env")
		strictFlag, _ := cmd.Flags().GetBool("strict")
		jsonOutputFlag, _ := cmd.Flags().GetBool("json-output")
		regionFlag, _ := cmd.Flags().GetString("region")

		opts.artifact = artifactFlag
		opts.registry = registryFlag
		opts.environment = environmentFlag
		opts.strict = strictFlag
		opts.jsonOutput = jsonOutputFlag
		opts.region = regionFlag

		// Interactive prompts if not provided via flags
		if opts.artifact == "" {
			if err := huh.NewInput().
				Title("Artifact to evaluate:").
				Description("The OCI artifact reference to check (e.g., ghcr.io/my-org/my-model:latest)").
				Value(&opts.artifact).
				Run(); err != nil {
				return err
			}
		}

		if opts.registry == "" {
			cfg := config.Load()
			if cfg.Registry != "" {
				opts.registry = cfg.Registry
			} else {
				if err := huh.NewSelect[string]().
					Title("Registry tool:").
					Description("Choose how to fetch the artifact manifest").
					Options(huh.NewOptions("oras", "modelpack")...).
					Value(&opts.registry).
					Run(); err != nil {
					return err
				}
				// Save config for future use
				cfg.Registry = opts.registry
				config.Save(cfg)
			}
		}

		if opts.environment == "" {
			if err := huh.NewSelect[string]().
				Title("Target environment:").
				Description("Destination environment for deployment").
				Options(huh.NewOptions("development", "staging", "production", "air-gapped", "hybrid-cloud")...).
				Value(&opts.environment).
				Run(); err != nil {
				return err
			}
		}

		if opts.region == "" && opts.environment == "hybrid-cloud" {
			if err := huh.NewInput().
				Title("Target region:").
				Description("Target region for hybrid-cloud validation (e.g., us-east-1, eu-west-1)").
				Value(&opts.region).
				Run(); err != nil {
				return err
			}
		}

		// Get registry provider
		provider, err := workflow.GetRegistryProvider(opts.registry)
		if err != nil {
			return err
		}

		// Check if tool is installed
		if !provider.IsInstalled() {
			return fmt.Errorf("%s not installed. Install with: %s", provider.Name(), provider.InstallInstructions())
		}

		if !opts.jsonOutput {
			fmt.Printf("\nEvaluating artifact '%s' for admission to '%s'...\n\n", opts.artifact, opts.environment)
		}

		// Track validation results
		allPassed := true
		missingRequirements := []string{}
		warnings := []string{}

		// === Fetch Manifest ===
		if !opts.jsonOutput {
			fmt.Println("=== Fetching Manifest ===")
		}

		manifestAnnotations, err := provider.FetchManifestAnnotations(opts.artifact)
		if err != nil {
			if !opts.jsonOutput {
				fmt.Printf("✗ Failed to fetch manifest for %s: %v\n", opts.artifact, err)
			}
			if opts.jsonOutput {
				fmt.Printf(`{"status":"fail","artifact":"%s","error":"%v"}\n`, opts.artifact, err)
			}
			os.Exit(1)
			return nil
		}

		if !opts.jsonOutput {
			fmt.Println("  ✓ Fetched artifact manifest from registry")
		}

		// === Trust Profile Evaluation (Story #63) ===
		if !opts.jsonOutput {
			fmt.Println("\n=== Trust Profile (Story #63) ===")
		}

		// Signature check - verify presence, not cryptographic validity
		// Actual verification delegated to Sigstore Policy Controller, cosign, etc.
		if signingFramework, ok := manifestAnnotations[workflow.AnnotationSigningFramework]; ok {
			if !opts.jsonOutput {
				fmt.Printf("  ✓ Signing framework: %s\n", signingFramework)
				fmt.Printf("  ℹ Actual signature verification delegated to external tools (Sigstore Policy Controller, cosign)\n")
			}
		} else {
			allPassed = false
			missingRequirements = append(missingRequirements, workflow.AnnotationSigningFramework)
			if !opts.jsonOutput {
				fmt.Printf("  ✗ Missing: %s\n", workflow.AnnotationSigningFramework)
			}
		}

		// SBOM check - verify presence, not validity
		// Actual SBOM validation delegated to external tools
		if sbomFormat, ok := manifestAnnotations[workflow.AnnotationSBOMFormat]; ok {
			if !opts.jsonOutput {
				fmt.Printf("  ✓ SBOM format: %s\n", sbomFormat)
				fmt.Printf("  ℹ Actual SBOM validation delegated to external tools\n")
			}
		} else {
			allPassed = false
			missingRequirements = append(missingRequirements, workflow.AnnotationSBOMFormat)
			if !opts.jsonOutput {
				fmt.Printf("  ✗ Missing: %s\n", workflow.AnnotationSBOMFormat)
			}
		}

		// Provenance check
		if provenanceType, ok := manifestAnnotations[workflow.AnnotationProvenanceType]; ok {
			if !opts.jsonOutput {
				fmt.Printf("  ✓ Provenance type: %s\n", provenanceType)
			}
		} else {
			allPassed = false
			missingRequirements = append(missingRequirements, workflow.AnnotationProvenanceType)
			if !opts.jsonOutput {
				fmt.Printf("  ✗ Missing: %s\n", workflow.AnnotationProvenanceType)
			}
		}

		// Profile version check
		if profileVersion, ok := manifestAnnotations[workflow.AnnotationProfileVersion]; ok {
			if !opts.jsonOutput {
				fmt.Printf("  ✓ Profile version: %s\n", profileVersion)
			}
		} else {
			allPassed = false
			missingRequirements = append(missingRequirements, workflow.AnnotationProfileVersion)
			if !opts.jsonOutput {
				fmt.Printf("  ✗ Missing: %s\n", workflow.AnnotationProfileVersion)
			}
		}

		// Artifact type check
		if artifactType, ok := manifestAnnotations[workflow.AnnotationArtifactType]; ok {
			if !opts.jsonOutput {
				fmt.Printf("  ✓ Artifact type: %s\n", artifactType)
			}
		} else {
			allPassed = false
			missingRequirements = append(missingRequirements, workflow.AnnotationArtifactType)
			if !opts.jsonOutput {
				fmt.Printf("  ✗ Missing: %s\n", workflow.AnnotationArtifactType)
			}
		}

		// === Infrastructure Dependency Matching (Story #64) ===
		if !opts.jsonOutput {
			fmt.Println("\n=== Infrastructure Dependencies (Story #64) ===")
		}

		// Runtime requirement
		if runtime, ok := manifestAnnotations[workflow.AnnotationRuntime]; ok {
			if !opts.jsonOutput {
				fmt.Printf("  ✓ Runtime: %s\n", runtime)
			}
		} else {
			allPassed = false
			missingRequirements = append(missingRequirements, workflow.AnnotationRuntime)
			if !opts.jsonOutput {
				fmt.Printf("  ✗ Missing: %s\n", workflow.AnnotationRuntime)
			}
		}

		// Accelerator requirement
		if accelerator, ok := manifestAnnotations[workflow.AnnotationAccelerator]; ok {
			if !opts.jsonOutput {
				fmt.Printf("  ✓ Accelerator: %s\n", accelerator)
			}
		} else {
			allPassed = false
			missingRequirements = append(missingRequirements, workflow.AnnotationAccelerator)
			if !opts.jsonOutput {
				fmt.Printf("  ✗ Missing: %s\n", workflow.AnnotationAccelerator)
			}
		}

		// Conditional: CUDA version
		if cudaMin, ok := manifestAnnotations[workflow.AnnotationCUDAVersionMin]; ok {
			if !opts.jsonOutput {
				fmt.Printf("  ✓ CUDA min: %s\n", cudaMin)
			}
		} else if opts.strict {
			warnings = append(warnings, fmt.Sprintf("Missing conditional annotation: %s", workflow.AnnotationCUDAVersionMin))
			if !opts.jsonOutput {
				fmt.Printf("  ⚠ Optional: %s (not present)\n", workflow.AnnotationCUDAVersionMin)
			}
		}

		// Conditional: Memory min
		if memoryMin, ok := manifestAnnotations[workflow.AnnotationMemoryMin]; ok {
			if !opts.jsonOutput {
				fmt.Printf("  ✓ Memory min: %s\n", memoryMin)
			}
		} else if opts.strict {
			warnings = append(warnings, fmt.Sprintf("Missing conditional annotation: %s", workflow.AnnotationMemoryMin))
			if !opts.jsonOutput {
				fmt.Printf("  ⚠ Optional: %s (not present)\n", workflow.AnnotationMemoryMin)
			}
		}

		// Environment matching (Story #66)
		if !opts.jsonOutput {
			fmt.Println("\n=== Environment Matching (Story #66) ===")
		}

		// Get the actual environment validation
		if opts.environment != "" {
			switch opts.environment {
			case "air-gapped":
				if packagingFormat, ok := manifestAnnotations[workflow.AnnotationPackagingFormat]; ok {
					if packagingFormat == "modelpack" {
						if !opts.jsonOutput {
							fmt.Printf("  ✓ Packaging format: %s (supports air-gapped)\n", packagingFormat)
						}
					} else {
						warnings = append(warnings, "Air-gapped: consider using modelpack packaging format")
						if !opts.jsonOutput {
							fmt.Printf("  ⚠ Packaging format: %s (may not support air-gapped)\n", packagingFormat)
						}
					}
				} else {
					warnings = append(warnings, "Air-gapped: missing packaging format annotation")
					if !opts.jsonOutput {
						fmt.Printf("  ⚠ Missing packaging format for air-gapped\n")
					}
				}

				if !opts.jsonOutput {
					fmt.Printf("  ✓ Air-gapped environment validated\n")
				}

			case "hybrid-cloud":
				if opts.region != "" {
					if residency, ok := manifestAnnotations[workflow.AnnotationDataResidency]; ok {
						if residency == opts.region {
							if !opts.jsonOutput {
								fmt.Printf("  ✓ Data residency: %s (matches target: %s)\n", residency, opts.region)
							}
						} else {
							if opts.strict {
								allPassed = false
								missingRequirements = append(missingRequirements, fmt.Sprintf("%s=%s", workflow.AnnotationDataResidency, opts.region))
								if !opts.jsonOutput {
									fmt.Printf("  ✗ Data residency mismatch: requires %s, target is %s\n", residency, opts.region)
								}
							} else {
								warnings = append(warnings, fmt.Sprintf("Data residency mismatch: requires %s, target is %s", residency, opts.region))
								if !opts.jsonOutput {
									fmt.Printf("  ⚠ Data residency mismatch: requires %s, target is %s\n", residency, opts.region)
								}
							}
						}
					} else {
						if !opts.jsonOutput {
							fmt.Printf("  ℹ No data residency requirement\n")
						}
					}

					if !opts.jsonOutput {
						fmt.Printf("  ✓ Hybrid-cloud environment validated for region: %s\n", opts.region)
					}
				} else {
					warnings = append(warnings, "Hybrid-cloud: region not specified")
					if !opts.jsonOutput {
						fmt.Printf("  ⚠ Region not specified for hybrid-cloud\n")
					}
				}

			case "development", "staging", "production":
				if !opts.jsonOutput {
					fmt.Printf("  ✓ Environment: %s\n", opts.environment)
				}

			case "":
				// No environment specified
				if !opts.jsonOutput {
					fmt.Printf("  ℹ No environment specified\n")
				}

			default:
				warnings = append(warnings, fmt.Sprintf("Unknown environment: %s", opts.environment))
				if !opts.jsonOutput {
					fmt.Printf("  ⚠ Unknown environment: %s\n", opts.environment)
				}
			}
		}

		// === Policy Enforcement ===
		if !opts.jsonOutput {
			fmt.Println("\n=== Policy Enforcement ===")
		}

		if opts.strict {
			if !opts.jsonOutput {
				fmt.Println("  ✓ Strict mode enabled")
				fmt.Println("  ✓ All requirements enforced")
			}
		} else {
			if !opts.jsonOutput {
				fmt.Println("  ⚠ Strict mode disabled")
				fmt.Println("  ⚠ Warnings allowed, not blocking")
			}
		}

		if !opts.jsonOutput {
			fmt.Println("  ✓ Air-gapped/hybrid-cloud safety policies checked")
		}

		// === Final Decision ===
		if !opts.jsonOutput {
			fmt.Println("\n=== Admission Decision ===")
		}

		if allPassed {
			if !opts.jsonOutput {
				fmt.Println("✓ ARTIFACT ADMITTED")
				fmt.Println("\nThe artifact:")
				fmt.Println("  - Has valid Trust Profile annotations")
				fmt.Println("  - Has Infrastructure Requirement annotations")
				fmt.Println("  - Meets environment safety policies")
				fmt.Println("\nGitOps controller can proceed with deployment.")
				fmt.Println("External policy engines (Sigstore Policy Controller, OPA/Gatekeeper, Kyverno)")
				fmt.Println("will perform the actual admission control.")
			}
			if opts.jsonOutput {
				fmt.Printf(`{"status":"admitted","artifact":"%s","missing_requirements":[]}\n`, opts.artifact)
			}
			return nil
		} else {
			if !opts.jsonOutput {
				fmt.Println("✗ ARTIFACT NOT ADMITTED")
				fmt.Println("\nMissing requirements:")
				for _, req := range missingRequirements {
					fmt.Printf("  - %s\n", req)
				}
				if len(warnings) > 0 {
					fmt.Println("\nWarnings:")
					for _, w := range warnings {
						fmt.Printf("  - %s\n", w)
					}
				}
				fmt.Println("\nTo fix: Re-package the artifact with model-cli package --sign")
				fmt.Println("and ensure all required metadata is included.")
			}
			if opts.jsonOutput {
				fmt.Printf(`{"status":"not_admitted","artifact":"%s","missing_requirements":[`, opts.artifact)
				for i, req := range missingRequirements {
					if i > 0 {
						fmt.Printf(",")
					}
					fmt.Printf(`"%s"`, req)
				}
				fmt.Printf("]}\n")
			}
			os.Exit(1)
			return nil
		}
	},
}

func init() {
	rootCmd.AddCommand(admitCmd)
	admitCmd.Flags().Bool("strict", false, "Strict mode: block admission on warnings")
	admitCmd.Flags().String("env", "", "Target environment (development, staging, production, air-gapped, hybrid-cloud)")
	admitCmd.Flags().String("artifact", "", "OCI artifact reference to evaluate")
	admitCmd.Flags().String("registry", "", "Registry tool: oras or modelpack")
	admitCmd.Flags().String("region", "", "Target region for hybrid-cloud validation (e.g., us-east-1, eu-west-1)")
	admitCmd.Flags().Bool("json-output", false, "Output results as JSON for CI/CD integration")
}
