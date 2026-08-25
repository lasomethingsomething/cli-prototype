package cmd

import (
	"fmt"

	"github.com/charmbracelet/huh"
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

The CLI orchestrates the search by delegating to registry tools (ORAS, ModelPack)
which perform the actual registry queries. The results are then parsed and filtered
client-side for complex relationship queries.

Examples:
  model-cli search
  model-cli search --registry ghcr.io/my-org --type model
  model-cli search --model my-llm --type pipeline
  model-cli search --metadata ai.model.type=llm
  model-cli search --uses-model model:sha256:abc123
  model-cli search --output json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Load()

		// Get flags
		destinationFlag, _ := cmd.Flags().GetString("destination")
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
		if destinationFlag != "" {
			destination = destinationFlag
		} else if cfg.Registry != "" {
			destination = cfg.Registry
		} else {
			// For search, we need at least a registry destination
			// If not configured, prompt the user
			if err := huh.NewInput().
				Title("Registry to search:").
				Description("Enter the registry to search (e.g., ghcr.io/my-org, docker.io/myuser)").
				Value(&destination).
				Run(); err != nil {
				return err
			}
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

		fmt.Printf("Searching %s for AI assets...\n\n", destination)

		// Build filter string for registry query
		filterString := query.ToFilterString()
		fmt.Printf("Query: %s\n\n", query.String())

		// Execute search via registry provider
		// The provider returns raw manifest bytes
		manifestBytes, err := provider.Search(destination, filterString)
		if err != nil {
			// If search fails, try without filters (get all manifests) and filter client-side
			fmt.Printf("Note: Registry search with filters failed: %v\n", err)
			fmt.Println("Falling back to client-side filtering...")

			// Try to get all artifacts from the registry
			// This is a fallback - in production, registries would support filtering natively
			manifestBytes, err = provider.Search(destination, "")
			if err != nil {
				return fmt.Errorf("failed to search registry: %v\n\nHint: Ensure the registry supports OCI artifact discovery (e.g., ghcr.io, docker.io)", err)
			}
		}

		// Parse search results from manifest bytes
		// This handles the client-side filtering and parsing
		searchResults, err := workflow.ParseSearchResultsFromManifests(manifestBytes)
		if err != nil {
			return fmt.Errorf("failed to parse search results: %v", err)
		}

		// Apply relationship filters client-side
		// The ParseSearchResultsFromManifests already parsed relationship graphs
		// Now we apply the query's relationship filters
		filteredResults := searchResults.FilterByRelationship(query)

		// Set the registry name
		filteredResults.Registry = destination

		// Output results
		switch query.OutputFormat {
		case "json":
			jsonOutput, err := filteredResults.ToJSON()
			if err != nil {
				return fmt.Errorf("failed to format results as JSON: %v", err)
			}
			fmt.Println(jsonOutput)
		case "yaml":
			// For YAML output, we'll use JSON and note that YAML is a superset
			// In production, use a proper YAML library
			jsonOutput, err := filteredResults.ToJSON()
			if err != nil {
				return fmt.Errorf("failed to format results: %v", err)
			}
			fmt.Println("# YAML output (JSON is a valid YAML subset)")
			fmt.Println(jsonOutput)
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
		fmt.Println("Next: Use specific artifact references to pull or validate: model-cli pull --artifact <reference>")

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
