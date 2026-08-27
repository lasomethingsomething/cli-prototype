package cmd

import (
	"fmt"

	"github.com/lasomethingsomething/cli-prototype/config"
	"github.com/lasomethingsomething/cli-prototype/internal/workflow"
	"github.com/spf13/cobra"
)

var searchCmd = &cobra.Command{
	Use:   "search",
	Short: "Search for AI assets in a registry",
	Long: `Search for AI artifacts in an OCI registry with filtering support.

This command allows you to discover and cross-reference AI assets (models, skills, pipelines)
in a registry. It supports filtering by metadata annotations and relationships,
enabling queries like "show all pipelines using model X" or "find all skills of type rag".

With the ORAS tool the registry is walked with "oras repo ls", "oras repo tags"
and "oras manifest fetch", so the registry must expose the OCI catalog API
(zot, registry:2, Harbor, ...). --type and --metadata match the manifest
annotations exactly; relationship filters such as --uses-model are evaluated
against the ai.relationships graph annotation.

With --output json only the result document is printed, so it can be piped
to tools like jq.

Examples:
  model-cli search --destination localhost:5000
  model-cli search --destination ghcr.io/my-org --type model
  model-cli search --destination localhost:5000 --metadata ai.model.type=llm
  model-cli search --destination localhost:5000 --uses-model localhost:5000/my-org/my-llm:v1
  model-cli search --destination localhost:5000 --type model --output json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Load()

		// Get flags
		registryToolFlag, _ := cmd.Flags().GetString("registry-tool")
		artifactTypeFlag, _ := cmd.Flags().GetString("type")
		modelFlag, _ := cmd.Flags().GetString("model")
		skillFlag, _ := cmd.Flags().GetString("skill")
		pipelineFlag, _ := cmd.Flags().GetString("pipeline")
		datasetFlag, _ := cmd.Flags().GetString("dataset")
		usesModelFlag, _ := cmd.Flags().GetString("uses-model")
		usesSkillFlag, _ := cmd.Flags().GetString("uses-skill")
		requiresModelFlag, _ := cmd.Flags().GetString("requires-model")
		usedByFlag, _ := cmd.Flags().GetString("used-by")
		metadataFlag, _ := cmd.Flags().GetStringSlice("metadata")
		outputFlag, _ := cmd.Flags().GetString("output")
		limitFlag, _ := cmd.Flags().GetInt("limit")

		// Build search query
		query := workflow.NewSearchQuery()
		query.Limit = limitFlag
		query.OutputFormat = outputFlag

		// Get registry destination
		var destination string
		if cfg.Registry != "" {
			destination = cfg.Registry
		}
		if cmd.Flags().Changed("destination") || destination == "" {
			if err := askString(cmd, "destination", &destination, "Registry to search:", "Enter the registry to search (e.g., ghcr.io/my-org, docker.io/myuser)"); err != nil {
				return err
			}
		}
		if err := requireValues("destination", destination); err != nil {
			return err
		}
		query.Registry = destination

		// Get artifact type
		if artifactTypeFlag != "" {
			query.ArtifactType = artifactTypeFlag
		}

		// Get filter values
		query.Model = modelFlag
		query.Skill = skillFlag
		query.Pipeline = pipelineFlag
		query.Dataset = datasetFlag
		query.UsesModel = usesModelFlag
		query.UsesSkill = usesSkillFlag
		query.RequiresModel = requiresModelFlag
		query.UsedBy = usedByFlag

		// Parse metadata filters
		for _, meta := range metadataFlag {
			// Parse key=value pairs
			var key, value string
			for i, c := range meta {
				if c == '=' {
					key = meta[:i]
					value = meta[i+1:]
					break
				}
			}
			if key != "" {
				query.MetadataFilters[key] = value
			}
		}

		// Get registry provider tool (oras or modelpack)
		var registryTool string
		if registryToolFlag != "" {
			registryTool = registryToolFlag
		} else if cfg.Registry != "" {
			// Check if config has registry tool stored (for backwards compatibility)
			registryTool = "oras" // Default to ORAS
		} else {
			registryTool = "oras" // Default to ORAS
		}

		provider, err := workflow.GetRegistryProvider(registryTool)
		if err != nil {
			return fmt.Errorf("failed to get registry provider: %v", err)
		}

		// Check if tool is installed
		if !provider.IsInstalled() {
			return fmt.Errorf("%s not installed. Install with: %s", provider.Name(), provider.InstallInstructions())
		}

		// json and yaml print nothing but the document so it can be piped.
		machineReadable := query.OutputFormat == "json" || query.OutputFormat == "yaml"
		if !machineReadable {
			fmt.Printf("Searching %s for AI assets...\n\n", destination)
			fmt.Printf("Query: %s\n\n", query.String())
		}

		// The provider applies the artifact-type and metadata filters
		// (client-side for ORAS); relationship filters are applied below.
		candidates, err := provider.Search(destination, query)
		if err != nil {
			return fmt.Errorf("failed to search registry: %v", err)
		}
		filteredResults := workflow.SearchResultsFromCandidates(candidates).FilterByRelationship(query)
		filteredResults.Registry = destination

		// Output results
		switch query.OutputFormat {
		case "json":
			jsonOutput, err := filteredResults.ToJSON()
			if err != nil {
				return fmt.Errorf("failed to format results as JSON: %v", err)
			}
			fmt.Println(jsonOutput)
			return nil
		case "yaml":
			// For YAML output, we'll use JSON and note that YAML is a superset
			// In production, use a proper YAML library
			jsonOutput, err := filteredResults.ToJSON()
			if err != nil {
				return fmt.Errorf("failed to format results: %v", err)
			}
			fmt.Println("# YAML output (JSON is a valid YAML subset)")
			fmt.Println(jsonOutput)
			return nil
		default: // table
			fmt.Println(filteredResults.String())
		}

		// Print summary
		fmt.Printf("\nFound %d result(s) matching criteria.\n", filteredResults.TotalCount)

		if filteredResults.TotalCount == 0 {
			fmt.Println("\nTip: Try broadening your search criteria or check if artifacts exist in the registry.")
			fmt.Println("You can also try searching without filters to see all available artifacts.")
		}

		fmt.Println("\nPhase 3 Complete: Cross-Reference Assets in Registry")
		fmt.Println("Next: Validate or verify a specific artifact reference:")
		fmt.Println("  model-cli validate gitops --artifact <reference>")
		fmt.Println("  model-cli verify --artifact <reference>")

		return nil
	},
}

func init() {
	rootCmd.AddCommand(searchCmd)

	// Registry and query flags
	searchCmd.Flags().String("destination", "", "Registry to search (e.g., ghcr.io/my-org, docker.io/myuser)")
	searchCmd.Flags().String("registry-tool", "", "Registry tool to use: oras or modelpack (default: oras)")
	searchCmd.Flags().String("type", "", "Filter by artifact type: model, skill, pipeline, dataset")

	// Direct reference filters
	searchCmd.Flags().String("model", "", "Filter by model reference")
	searchCmd.Flags().String("skill", "", "Filter by skill reference")
	searchCmd.Flags().String("pipeline", "", "Filter by pipeline reference")
	searchCmd.Flags().String("dataset", "", "Filter by dataset reference")

	// Relationship filters
	searchCmd.Flags().String("uses-model", "", "Find artifacts that use this model (e.g., model:sha256:abc123)")
	searchCmd.Flags().String("uses-skill", "", "Find artifacts that use this skill")
	searchCmd.Flags().String("requires-model", "", "Find skills that require this model")
	searchCmd.Flags().String("used-by", "", "Find artifacts used by this reference")

	// Metadata filters (can be specified multiple times)
	searchCmd.Flags().StringSlice("metadata", []string{}, "Metadata filter in format key=value (e.g., ai.model.type=llm)")

	// Output flags
	searchCmd.Flags().String("output", "table", "Output format: table, json, yaml")
	searchCmd.Flags().Int("limit", 100, "Maximum number of results to return")
}
