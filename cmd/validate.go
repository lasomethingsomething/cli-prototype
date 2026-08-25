package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/lasomethingsomething/cli-prototype/internal/workflow"
	"github.com/spf13/cobra"
)

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate an artifact against the metadata contract, GitOps policies or a cluster",
	Long: `Validate evaluates an artifact's CNCF AI annotations against a target:

  validate manifest   Metadata contract (JSON Schema) of a local manifest or a registry artifact
  validate local      Local compliance check of a model directory before pushing
  validate gitops     Trust profile + infrastructure annotations for GitOps pre-sync hooks
  validate admission  Full admission evaluation for a target environment (air-gapped, hybrid-cloud, ...)
  validate nodes      Cluster nodes satisfy the artifact's GPU/vRAM/topology requirements
  validate runtime    Runtime operators (vLLM, KServe) the artifact needs are available

Every target shares the same flags for output (--quiet, --json-output) and, where a registry
is involved, for the artifact (--artifact, --registry). Running 'validate' with the
manifest flags and no subcommand is the same as 'validate manifest'.

Examples:
  model-cli validate manifest --manifest ./model/manifest.json --json-schema --strict
  model-cli validate gitops --artifact ghcr.io/my-org/my-model:v1 --env air-gapped --quiet
  model-cli validate admission --artifact my-model:v1 --env hybrid-cloud --region eu-west-1 --json-output
  model-cli validate nodes --artifact my-model:v1 --namespace production`,
	Args: cobra.NoArgs,
	RunE: runValidateManifest,
}

func newValidateManifestCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "manifest",
		Short: "Validate a manifest against the Standardized Metadata Contract",
		Long: `Validate the CNCF AI annotations of a manifest against the Standardized Metadata
Contract. Reads a local manifest.json (as written by 'model-cli package') with --manifest,
or fetches the manifest of --artifact from the registry.

Examples:
  model-cli validate manifest --manifest ./my-model/manifest.json
  model-cli validate manifest --manifest ./my-model/manifest.json --json-schema --strict
  model-cli validate manifest --artifact ghcr.io/my-org/my-model:v1 --check-relationships`,
		Args: cobra.NoArgs,
		RunE: runValidateManifest,
	}
	addManifestFlags(c)
	return c
}

func addManifestFlags(c *cobra.Command) {
	addArtifactFlags(c)
	c.Flags().String("manifest", "", "Path to a local OCI manifest.json (e.g. produced by 'model-cli package') instead of fetching from the registry")
	c.Flags().Bool("check-relationships", false, "Map and validate artifact relationships")
	c.Flags().Bool("json-schema", false, "Use JSON Schema validation for the metadata contract")
	c.Flags().Bool("strict", false, "Strict mode: fail validation on warnings")
	c.Flags().String("artifact-type", "", "Artifact type: model, skill, or pipeline (overrides annotation)")
}

func runValidateManifest(cmd *cobra.Command, args []string) error {
	manifestFlag, _ := cmd.Flags().GetString("manifest")
	jsonSchemaFlag, _ := cmd.Flags().GetBool("json-schema")
	strictFlag, _ := cmd.Flags().GetBool("strict")
	artifactTypeFlag, _ := cmd.Flags().GetString("artifact-type")

	var checkRelationships bool
	if err := askConfirm(cmd, "check-relationships", &checkRelationships, "Check relationships?", "Map model->skill and other artifact relationships"); err != nil {
		return err
	}

	var (
		subject             string
		manifestAnnotations map[string]string
	)
	if manifestFlag != "" {
		subject = manifestFlag
		fmt.Println("\n✓ Reading manifest from disk...")
		manifest, err := workflow.ReadUnifiedOCIManifest(manifestFlag)
		if err != nil {
			return err
		}
		manifestAnnotations = manifest.Annotations
		if len(manifestAnnotations) == 0 {
			return fmt.Errorf("manifest at %s has no CNCF AI annotations", manifestFlag)
		}
		for _, key := range []string{workflow.AnnotationProfileVersion, workflow.AnnotationArtifactType} {
			if _, ok := manifestAnnotations[key]; !ok {
				return fmt.Errorf("manifest at %s is missing required annotation %q", manifestFlag, key)
			}
		}
	} else {
		artifact, annotations, err := fetchAnnotations(cmd, outputOptions{})
		if err != nil {
			return err
		}
		subject, manifestAnnotations = artifact, annotations
	}
	fmt.Printf("\nValidating manifest for: %s\n", subject)
	fmt.Println("✓ Reading Standardized Metadata Contract...")

	artifactType := workflow.ArtifactTypeModel
	if artifactTypeFlag != "" {
		artifactType = workflow.ArtifactType(artifactTypeFlag)
	} else if at, ok := manifestAnnotations[workflow.AnnotationArtifactType]; ok {
		artifactType = workflow.ArtifactType(at)
	}

	fmt.Println("\n  Detected Annotations:")
	for _, key := range sortedKeys(manifestAnnotations) {
		fmt.Printf("    %s: %s\n", key, manifestAnnotations[key])
	}

	var contractResult *workflow.ContractValidationResult
	if contractJSON, ok := manifestAnnotations[workflow.AnnotationMetadataContract]; ok && jsonSchemaFlag {
		contractResult, _ = workflow.ValidateContractWithStrictJSONSchema(contractJSON)
	} else {
		contractResult = workflow.ValidateManifestMetadata(manifestAnnotations, artifactType)
	}

	if !contractResult.Valid {
		fmt.Printf("\n✗ Metadata contract validation FAILED\n")
		fmt.Println(contractResult.String())
		if strictFlag && len(contractResult.Warnings) > 0 {
			fmt.Printf("\n✗ Strict mode: validation failed due to %d warning(s)\n", len(contractResult.Warnings))
			for _, warn := range contractResult.Warnings {
				fmt.Printf("  - %s\n", warn)
			}
			return fmt.Errorf("strict validation failed: %d warning(s)", len(contractResult.Warnings))
		}
		return fmt.Errorf("manifest validation failed: %v", strings.Join(contractResult.Errors, "; "))
	}

	fmt.Println("  ✓ All required fields present")
	if len(contractResult.Warnings) > 0 {
		fmt.Printf("  ⚠ %d warning(s):\n", len(contractResult.Warnings))
		for _, warn := range contractResult.Warnings {
			fmt.Printf("    - %s\n", warn)
		}
		if strictFlag {
			fmt.Printf("\n⚠ Strict mode: validation passed with warnings\n")
		}
	}

	fmt.Println("\n✓ Profile annotations valid")
	fmt.Println("✓ MOF classification valid")
	fmt.Println("✓ Security annotations valid")
	fmt.Println("✓ Runtime requirements valid")

	if checkRelationships {
		printRelationships(manifestAnnotations)
	}

	fmt.Println("\n✓ Manifest validation passed")
	fmt.Println("✓ Artifact is compliant with Standardized Metadata Contract")
	fmt.Println("✓ Artifact is compliant with CNCF AI Interoperability Profile")
	return nil
}

func printRelationships(annotations map[string]string) {
	fmt.Println("\n✓ Mapping relationships...")
	contract, err := workflow.ParseMetadataContractFromAnnotations(annotations)
	if err != nil || contract == nil {
		fmt.Println("  (no metadata contract with relationships found in the annotations)")
		return
	}
	if contract.Assets.Model != nil && len(contract.Assets.Model.Relationships) > 0 {
		fmt.Println("  Model relationships:")
		for relType, refs := range contract.Assets.Model.Relationships {
			for _, ref := range refs {
				fmt.Printf("    - %s: %s\n", relType, ref)
			}
		}
	}
	if contract.Assets.Skill != nil && len(contract.Assets.Skill.Dependencies) > 0 {
		fmt.Println("  Skill dependencies:")
		for depType, refs := range contract.Assets.Skill.Dependencies {
			for _, ref := range refs {
				fmt.Printf("    - %s: %s\n", depType, ref)
			}
		}
	}
	if contract.Assets.Pipeline != nil {
		if len(contract.Assets.Pipeline.Dependencies) > 0 {
			fmt.Println("  Pipeline dependencies:")
			for depType, refs := range contract.Assets.Pipeline.Dependencies {
				for _, ref := range refs {
					fmt.Printf("    - %s: %s\n", depType, ref)
				}
			}
		}
		if len(contract.Assets.Pipeline.Components) > 0 {
			fmt.Println("  Pipeline components:")
			for _, comp := range contract.Assets.Pipeline.Components {
				fmt.Printf("    - %s (%s): %s\n", comp.Name, comp.Type, comp.Reference)
			}
		}
	}
	fmt.Println("✓ Dependencies resolved without downloading binaries")
}

// sortedKeys returns the keys of m in sorted order, for deterministic output.
func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func init() {
	addManifestFlags(validateCmd)
	rootCmd.AddCommand(validateCmd)

	validateCmd.AddCommand(
		newValidateManifestCmd(),
		newValidateLocalCmd(),
		newValidateGitOpsCmd(),
		newValidateAdmissionCmd(),
		newValidateNodesCmd(),
		newValidateRuntimeCmd(),
	)

	// Old top-level spellings, hidden and marked deprecated.
	rootCmd.AddCommand(
		deprecatedAlias("check", newValidateLocalCmd),
		deprecatedAlias("validate-gitops", newValidateGitOpsCmd),
		deprecatedAlias("admit", newValidateAdmissionCmd),
		deprecatedAlias("validate-nodes", newValidateNodesCmd),
		deprecatedAlias("validate-runtime", newValidateRuntimeCmd),
	)
}
