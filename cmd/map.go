package cmd

import (
	"fmt"
	"strings"

	"github.com/lasomethingsomething/cli-prototype/internal/workflow"
	"github.com/spf13/cobra"
)

var mapCmd = &cobra.Command{
	Use:   "map",
	Short: "Map complex relationships between AI artifacts",
	Long: `Map complex relationships between AI artifacts and embed them in OCI manifests.

This command handles the relationship mapping for Phase 2 (Enterprise OCI Registry Integration).
It generates a relationship graph that can be indexed by registries for fast dependency queries
without downloading artifact binaries.

The command supports:
- Model -> Skill relationships (skill requires model)
- Skill -> Pipeline relationships (pipeline uses skill)
- Model -> Pipeline relationships (pipeline uses model)
- Dataset relationships

The CLI generates the relationship graph and embeds it in the manifest annotations.
Registries can then use this metadata to index and query relationships efficiently.

Examples:
  # Map a model to a skill
  model-cli map --model my-model:v1 --requires my-skill:v1

  # Map a skill to a pipeline
  model-cli map --skill my-skill:v1 --used-by my-pipeline:v1

  # Map with digest references
  model-cli map --model my-model:sha256:abc123 --requires my-skill:sha256:def456

  # Generate relationship graph from a manifest
  model-cli map --manifest ./manifest.json --output ./mapped-manifest.json

  # Map multiple relationships at once
  model-cli map --model my-model:v1 --requires my-skill:v1,my-other-skill:v1
  model-cli map --skill my-skill:v1 --used-by my-pipeline:v1,my-other-pipeline:v1`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get flags
		manifestFlag, _ := cmd.Flags().GetString("manifest")
		outputFlag, _ := cmd.Flags().GetString("output")
		modelFlag, _ := cmd.Flags().GetString("model")
		skillFlag, _ := cmd.Flags().GetString("skill")
		pipelineFlag, _ := cmd.Flags().GetString("pipeline")
		datasetFlag, _ := cmd.Flags().GetString("dataset")
		requiresFlag, _ := cmd.Flags().GetStringSlice("requires")
		usedByFlag, _ := cmd.Flags().GetStringSlice("used-by")
		modelTypeFlag, _ := cmd.Flags().GetString("model-type")
		modelFrameworkFlag, _ := cmd.Flags().GetString("model-framework")
		skillTypeFlag, _ := cmd.Flags().GetString("skill-type")
		pipelineTypeFlag, _ := cmd.Flags().GetString("pipeline-type")
		pipelineStagesFlag, _ := cmd.Flags().GetStringSlice("pipeline-stages")
		printFlag, _ := cmd.Flags().GetBool("print")

		// Create relationship graph
		graph := workflow.NewRelationshipGraph()

		// If manifest is provided, load existing relationships
		if manifestFlag != "" {
			unifiedManifest, err := workflow.ReadUnifiedOCIManifest(manifestFlag)
			if err == nil {
				// Parse existing relationship graph from annotations if present
				if relGraph, ok := unifiedManifest.Annotations[workflow.AnnotationRelationshipGraph]; ok {
					existingGraph, err := workflow.ParseRelationshipGraphFromAnnotation(relGraph)
					if err == nil && existingGraph != nil {
						// Merge existing graph
						for ref, node := range existingGraph.Models {
							graph.Models[ref] = node
						}
						for ref, node := range existingGraph.Skills {
							graph.Skills[ref] = node
						}
						for ref, node := range existingGraph.Pipelines {
							graph.Pipelines[ref] = node
						}
						for ref, node := range existingGraph.Datasets {
							graph.Datasets[ref] = node
						}
					}
				}
			}
		}

		// Add model if specified
		if modelFlag != "" {
			sha256 := extractSHA256(modelFlag)
			graph.AddModel(modelFlag, sha256, modelTypeFlag, modelFrameworkFlag)
			fmt.Printf("Added model: %s\n", modelFlag)
		}

		// Add skill if specified
		if skillFlag != "" {
			sha256 := extractSHA256(skillFlag)
			graph.AddSkill(skillFlag, sha256, skillTypeFlag)
			fmt.Printf("Added skill: %s\n", skillFlag)

			// Add requirements (models that this skill requires)
			for _, req := range requiresFlag {
				modelRef := cleanReference(req)
				if modelRef != "" {
					if _, exists := graph.Models[modelRef]; !exists {
						graph.AddModel(modelRef, "", "", "")
					}
					if err := graph.AddModelToSkill(skillFlag, modelRef); err == nil {
						fmt.Printf("  Skill %s requires model: %s\n", skillFlag, modelRef)
					}
				}
			}

			// Add used-by (pipelines that use this skill)
			for _, user := range usedByFlag {
				pipelineRef := cleanReference(user)
				if pipelineRef != "" {
					if _, exists := graph.Pipelines[pipelineRef]; !exists {
						graph.AddPipeline(pipelineRef, "", "", []string{})
					}
					if err := graph.AddSkillToPipeline(pipelineRef, skillFlag); err == nil {
						fmt.Printf("  Skill %s used by pipeline: %s\n", skillFlag, pipelineRef)
					}
				}
			}
		}

		// Add pipeline if specified
		if pipelineFlag != "" {
			sha256 := extractSHA256(pipelineFlag)
			stages := pipelineStagesFlag
			if len(stages) == 0 && pipelineTypeFlag == "inference" {
				stages = []string{"inference"}
			} else if len(stages) == 0 {
				stages = []string{pipelineTypeFlag}
			}
			graph.AddPipeline(pipelineFlag, sha256, pipelineTypeFlag, stages)
			fmt.Printf("Added pipeline: %s\n", pipelineFlag)

			// Add dependencies (models and skills that this pipeline uses)
			for _, req := range requiresFlag {
				if strings.HasPrefix(req, "model:") || strings.HasPrefix(req, "skill:") ||
					strings.HasPrefix(req, "pipeline:") || strings.HasPrefix(req, "dataset:") {
					// Typed reference - extract the actual reference
					ref := cleanReference(req)
					if _, exists := graph.Models[ref]; !exists {
						// Check what type of reference this is
						if strings.HasPrefix(req, "model:") {
							graph.AddModel(ref, "", "", "")
							if err := graph.AddModelToPipeline(pipelineFlag, ref); err == nil {
								fmt.Printf("  Pipeline %s uses model: %s\n", pipelineFlag, ref)
							}
						} else if strings.HasPrefix(req, "skill:") {
							graph.AddSkill(ref, "", "")
							if err := graph.AddSkillToPipeline(pipelineFlag, ref); err == nil {
								fmt.Printf("  Pipeline %s uses skill: %s\n", pipelineFlag, ref)
							}
						}
					}
				}
			}
		}

		// Add dataset if specified
		if datasetFlag != "" {
			sha256 := extractSHA256(datasetFlag)
			graph.AddDataset(datasetFlag, sha256, "")
			fmt.Printf("Added dataset: %s\n", datasetFlag)
		}

		// Check if we have any relationships
		isEmpty := len(graph.Models) == 0 && len(graph.Skills) == 0 && len(graph.Pipelines) == 0 && len(graph.Datasets) == 0

		// If no relationships were added, check if we should print existing
		if isEmpty {
			fmt.Println("No relationships specified.")

			if manifestFlag != "" {
				fmt.Println("\nExisting relationship graph:")
				unifiedManifest, err := workflow.ReadUnifiedOCIManifest(manifestFlag)
				if err != nil {
					return err
				}

				if relGraph, ok := unifiedManifest.Annotations[workflow.AnnotationRelationshipGraph]; ok {
					existingGraph, err := workflow.ParseRelationshipGraphFromAnnotation(relGraph)
					if err == nil && existingGraph != nil {
						fmt.Println(existingGraph.String())
					} else {
						fmt.Printf("  Error parsing existing graph: %v\n", err)
					}
				} else {
					fmt.Println("  No existing relationship graph found in manifest.")
				}
			}

			return nil
		}

		// Generate the relationship graph annotation
		annotationValue, err := workflow.GenerateRelationshipGraphAnnotation(graph)
		if err != nil {
			return fmt.Errorf("failed to generate relationship graph annotation: %v", err)
		}

		// Print the relationship graph
		if printFlag || outputFlag == "" {
			fmt.Println("\n=== Relationship Graph ===")
			fmt.Println(graph.String())

			fmt.Println("\n=== Generated Annotation ===")
			fmt.Printf("%s: %s\n", workflow.AnnotationRelationshipGraph, annotationValue)
		}

		// Determine output path
		outputPath := outputFlag
		if outputPath == "" && manifestFlag != "" {
			outputPath = manifestFlag
		}

		// If output flag is specified or we're updating in place
		if outputPath != "" {
			var manifest *workflow.UnifiedOCIManifest
			var readFrom string

			if manifestFlag != "" {
				readFrom = manifestFlag
				manifest, err = workflow.ReadUnifiedOCIManifest(readFrom)
				if err != nil {
					return fmt.Errorf("failed to read manifest: %v", err)
				}
			} else {
				// Create a new manifest
				manifest = workflow.NewUnifiedOCIManifest(workflow.ArtifactTypeModel, "relationships", []workflow.OCILayer{})
			}

			// Set the relationship graph annotation
			if manifest.Annotations == nil {
				manifest.Annotations = make(map[string]string)
			}
			manifest.Annotations[workflow.AnnotationRelationshipGraph] = annotationValue

			// Write the manifest
			if err := workflow.WriteUnifiedOCIManifest(manifest, outputPath); err != nil {
				return fmt.Errorf("failed to write manifest: %v", err)
			}

			if outputFlag != "" {
				fmt.Printf("\n✓ Relationship graph written to: %s\n", outputPath)
			} else {
				fmt.Printf("\n✓ Relationship graph added to: %s\n", outputPath)
			}
		}

		fmt.Println("\n✓ Relationship mapping complete")
		fmt.Println(" Registries can now index these relationships for fast dependency queries")

		return nil
	},
}

// extractSHA256 extracts sha256 digest from a reference if present
func extractSHA256(ref string) string {
	if strings.Contains(ref, "sha256:") {
		parts := strings.Split(ref, "sha256:")
		if len(parts) > 1 {
			return "sha256:" + parts[1]
		}
	}
	return ""
}

// cleanReference extracts the actual reference from a typed reference
// e.g., "model:my-model:v1" -> "my-model:v1"
// e.g., "my-model:v1" -> "my-model:v1"
// e.g., "model:v1" -> "model:v1" (this is a reference named "model:v1", not a typed reference)
func cleanReference(ref string) string {
	// Check for type prefixes followed by a name with tag or digest
	// Pattern: type:name where type is one of: model, skill, pipeline, dataset
	// But only strip if name contains : or / (indicating it's a full reference)

	typePrefixes := []string{"model:", "skill:", "pipeline:", "dataset:"}
	for _, prefix := range typePrefixes {
		if strings.HasPrefix(ref, prefix) {
			// Check if the part after prefix looks like a reference (has : or /)
			remainder := strings.TrimPrefix(ref, prefix)
			if strings.Contains(remainder, ":") || strings.Contains(remainder, "/") {
				return remainder
			}
			// Otherwise, it's a reference that happens to start with the type name
			// e.g., "model:v1" is a valid reference name
			return ref
		}
	}
	return ref
}

func init() {
	rootCmd.AddCommand(mapCmd)
	mapCmd.Flags().String("manifest", "", "Path to existing manifest to update")
	mapCmd.Flags().String("output", "", "Path to write the updated manifest")
	mapCmd.Flags().String("model", "", "Model reference (e.g., my-model:v1 or my-model:sha256:abc123)")
	mapCmd.Flags().String("skill", "", "Skill reference (e.g., my-skill:v1)")
	mapCmd.Flags().String("pipeline", "", "Pipeline reference (e.g., my-pipeline:v1)")
	mapCmd.Flags().String("dataset", "", "Dataset reference (e.g., my-dataset:v1)")
	mapCmd.Flags().StringSlice("requires", []string{}, "References this asset requires (e.g., model:my-model:v1)")
	mapCmd.Flags().StringSlice("used-by", []string{}, "References that use this asset (e.g., pipeline:my-pipeline:v1)")
	mapCmd.Flags().String("model-type", "", "Model type (e.g., llm, embedding, classification)")
	mapCmd.Flags().String("model-framework", "", "Model framework (e.g., pytorch, tensorflow)")
	mapCmd.Flags().String("skill-type", "", "Skill type (e.g., rag, classification)")
	mapCmd.Flags().String("pipeline-type", "", "Pipeline type (e.g., inference, training)")
	mapCmd.Flags().StringSlice("pipeline-stages", []string{}, "Pipeline stages (e.g., preprocess, inference)")
	mapCmd.Flags().Bool("print", false, "Print the relationship graph without updating manifest")
}
