package workflow

import (
	"encoding/json"
	"fmt"
	"strings"
)

// SearchQuery represents a query to search for AI assets in a registry
type SearchQuery struct {
	// Registry to search in
	Registry string
	
	// ArtifactType filters by artifact type (model, skill, pipeline, dataset)
	ArtifactType string
	
	// Model filters by model reference
	Model string
	
	// Skill filters by skill reference
	Skill string
	
	// Pipeline filters by pipeline reference
	Pipeline string
	
	// Dataset filters by dataset reference
	Dataset string
	
	// Metadata filters (key=value pairs for annotation filtering)
	// e.g., ai.model.type=llm, ai.skill.type=rag
	MetadataFilters map[string]string
	
	// Relationship filters
	UsesModel string
	UsesSkill string
	RequiresModel string
	UsedBy string
	
	// Limit the number of results
	Limit int
	
	// Output format
	OutputFormat string // json, yaml, table
}

// SearchResult represents a single search result from the registry
type SearchResult struct {
	// Reference is the full artifact reference (e.g., ghcr.io/org/model:v1)
	Reference string `json:"reference"`
	
	// Digest is the SHA256 digest of the artifact
	Digest string `json:"digest,omitempty"`
	
	// ArtifactType is the type of AI artifact
	ArtifactType string `json:"artifact_type,omitempty"`
	
	// Annotations contains the manifest annotations
	Annotations map[string]string `json:"annotations,omitempty"`
	
	// RelationshipGraph contains the parsed relationship graph (if available)
	RelationshipGraph *RelationshipGraph `json:"relationships,omitempty"`
	
	// Metadata contains parsed metadata from annotations
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// SearchResults represents a collection of search results
type SearchResults struct {
	// Query contains the original search query
	Query *SearchQuery `json:"query,omitempty"`
	
	// Results contains the list of matching artifacts
	Results []SearchResult `json:"results"`
	
	// TotalCount is the total number of results (may be more than len(Results) if paginated)
	TotalCount int `json:"total_count"`
	
	// Registry is the registry that was searched
	Registry string `json:"registry"`
}

// NewSearchQuery creates a new search query
func NewSearchQuery() *SearchQuery {
	return &SearchQuery{
		MetadataFilters: make(map[string]string),
		Limit:          100,
		OutputFormat:   "table",
	}
}

// String returns a human-readable representation of the search query
func (q *SearchQuery) String() string {
	var parts []string
	
	if q.Registry != "" {
		parts = append(parts, fmt.Sprintf("registry=%s", q.Registry))
	}
	if q.ArtifactType != "" {
		parts = append(parts, fmt.Sprintf("type=%s", q.ArtifactType))
	}
	if q.Model != "" {
		parts = append(parts, fmt.Sprintf("model=%s", q.Model))
	}
	if q.Skill != "" {
		parts = append(parts, fmt.Sprintf("skill=%s", q.Skill))
	}
	if q.Pipeline != "" {
		parts = append(parts, fmt.Sprintf("pipeline=%s", q.Pipeline))
	}
	if q.Dataset != "" {
		parts = append(parts, fmt.Sprintf("dataset=%s", q.Dataset))
	}
	if q.UsesModel != "" {
		parts = append(parts, fmt.Sprintf("uses_model=%s", q.UsesModel))
	}
	if q.UsesSkill != "" {
		parts = append(parts, fmt.Sprintf("uses_skill=%s", q.UsesSkill))
	}
	if q.RequiresModel != "" {
		parts = append(parts, fmt.Sprintf("requires_model=%s", q.RequiresModel))
	}
	if q.UsedBy != "" {
		parts = append(parts, fmt.Sprintf("used_by=%s", q.UsedBy))
	}
	for k, v := range q.MetadataFilters {
		parts = append(parts, fmt.Sprintf("%s=%s", k, v))
	}
	
	return strings.Join(parts, ", ")
}

// ToFilterString converts the query to a registry filter string
// This follows the OCI Distribution Spec filtering format
func (q *SearchQuery) ToFilterString() string {
	var filters []string
	
	// Filter by artifact type
	if q.ArtifactType != "" {
		filters = append(filters, fmt.Sprintf("%s=%s", AnnotationArtifactType, q.ArtifactType))
	}
	
	// Add metadata filters
	for k, v := range q.MetadataFilters {
		// Handle special AI metadata paths
		if strings.HasPrefix(k, "ai.") {
			// These are in the ai.assets annotation
			filters = append(filters, fmt.Sprintf("%s=%s", k, v))
		} else {
			filters = append(filters, fmt.Sprintf("%s=%s", k, v))
		}
	}
	
	// Add relationship filters
	// These are more complex and may require client-side filtering
	// since not all registries support complex relationship queries
	if q.Model != "" {
		filters = append(filters, fmt.Sprintf("ai.model.ref=%s", q.Model))
	}
	if q.Skill != "" {
		filters = append(filters, fmt.Sprintf("ai.skill.ref=%s", q.Skill))
	}
	if q.Pipeline != "" {
		filters = append(filters, fmt.Sprintf("ai.pipeline.ref=%s", q.Pipeline))
	}
	if q.Dataset != "" {
		filters = append(filters, fmt.Sprintf("ai.dataset.ref=%s", q.Dataset))
	}
	
	return strings.Join(filters, "&")
}

// NewSearchResults creates a new search results container
func NewSearchResults(query *SearchQuery, registry string) *SearchResults {
	return &SearchResults{
		Query:    query,
		Results:  []SearchResult{},
		Registry: registry,
	}
}

// AddResult adds a search result to the results
func (r *SearchResults) AddResult(result SearchResult) {
	r.Results = append(r.Results, result)
	r.TotalCount++
}

// String returns a human-readable representation of the search results
func (r *SearchResults) String() string {
	if len(r.Results) == 0 {
		return "No results found"
	}
	
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Search Results from %s:\n\n", r.Registry))
	
	for i, result := range r.Results {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, result.Reference))
		if result.Digest != "" {
			sb.WriteString(fmt.Sprintf("   Digest: %s\n", result.Digest))
		}
		if result.ArtifactType != "" {
			sb.WriteString(fmt.Sprintf("   Type: %s\n", result.ArtifactType))
		}
		
		// Show key annotations
		if result.Annotations != nil {
			if title, ok := result.Annotations["org.opencontainers.image.title"]; ok {
				sb.WriteString(fmt.Sprintf("   Title: %s\n", title))
			}
			if desc, ok := result.Annotations["org.opencontainers.image.description"]; ok {
				sb.WriteString(fmt.Sprintf("   Description: %s\n", desc))
			}
			
			// Show AI-specific metadata
			if modelType, ok := result.Annotations["ai.model.type"]; ok {
				sb.WriteString(fmt.Sprintf("   Model Type: %s\n", modelType))
			}
			if modelFramework, ok := result.Annotations["ai.model.framework"]; ok {
				sb.WriteString(fmt.Sprintf("   Framework: %s\n", modelFramework))
			}
			if skillType, ok := result.Annotations["ai.skill.type"]; ok {
				sb.WriteString(fmt.Sprintf("   Skill Type: %s\n", skillType))
			}
			if pipelineType, ok := result.Annotations["ai.pipeline.type"]; ok {
				sb.WriteString(fmt.Sprintf("   Pipeline Type: %s\n", pipelineType))
			}
		}
		
		// Show relationships if available
		if result.RelationshipGraph != nil {
			if len(result.RelationshipGraph.Models) > 0 {
				sb.WriteString("   Models:\n")
				for ref := range result.RelationshipGraph.Models {
					sb.WriteString(fmt.Sprintf("     - %s\n", ref))
				}
			}
			if len(result.RelationshipGraph.Skills) > 0 {
				sb.WriteString("   Skills:\n")
				for ref := range result.RelationshipGraph.Skills {
					sb.WriteString(fmt.Sprintf("     - %s\n", ref))
				}
			}
			if len(result.RelationshipGraph.Pipelines) > 0 {
				sb.WriteString("   Pipelines:\n")
				for ref := range result.RelationshipGraph.Pipelines {
					sb.WriteString(fmt.Sprintf("     - %s\n", ref))
				}
			}
		}
		
		if i < len(r.Results)-1 {
			sb.WriteString("\n")
		}
	}
	
	return sb.String()
}

// ToJSON converts the search results to JSON
func (r *SearchResults) ToJSON() (string, error) {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// FilterByRelationship filters results to only those that have the specified relationship
func (r *SearchResults) FilterByRelationship(query *SearchQuery) *SearchResults {
	filtered := NewSearchResults(query, r.Registry)
	
	for _, result := range r.Results {
		// If no relationship graph, include it (can't filter)
		if result.RelationshipGraph == nil {
			// Check if annotations contain the relationship info
			if r.matchesAnnotationQuery(result, query) {
				filtered.AddResult(result)
			}
			continue
		}
		
		graph := result.RelationshipGraph
		
		// Filter by uses_model
		if query.UsesModel != "" {
			found := false
			for _, pipeline := range graph.Pipelines {
				for _, modelRef := range pipeline.UsesModels {
					if modelRef == query.UsesModel {
						found = true
						break
					}
				}
				if found {
					break
				}
			}
			for _, skill := range graph.Skills {
				for _, modelRef := range skill.RequiresModels {
					if modelRef == query.UsesModel {
						found = true
						break
					}
				}
				if found {
					break
				}
			}
			if !found {
				continue
			}
		}
		
		// Filter by uses_skill
		if query.UsesSkill != "" {
			found := false
			for _, pipeline := range graph.Pipelines {
				for _, skillRef := range pipeline.UsesSkills {
					if skillRef == query.UsesSkill {
						found = true
						break
					}
				}
				if found {
					break
				}
			}
			if !found {
				continue
			}
		}
		
		// Filter by requires_model
		if query.RequiresModel != "" {
			found := false
			for _, skill := range graph.Skills {
				for _, modelRef := range skill.RequiresModels {
					if modelRef == query.RequiresModel {
						found = true
						break
					}
				}
				if found {
					break
				}
			}
			if !found {
				continue
			}
		}
		
		// Filter by used_by
		if query.UsedBy != "" {
			found := false
			for _, model := range graph.Models {
				for _, user := range model.UsedBy {
					if user == query.UsedBy {
						found = true
						break
					}
				}
				if found {
					break
				}
			}
			for _, skill := range graph.Skills {
				for _, user := range skill.UsedBy {
					if user == query.UsedBy {
						found = true
						break
					}
				}
				if found {
					break
				}
			}
			if !found {
				continue
			}
		}
		
		// If all filters passed, add to results
		filtered.AddResult(result)
	}
	
	return filtered
}

// matchesAnnotationQuery checks if a result matches annotation-based queries
func (r *SearchResults) matchesAnnotationQuery(result SearchResult, query *SearchQuery) bool {
	// If no specific filters, match
	if query.Model == "" && query.Skill == "" && query.Pipeline == "" && query.Dataset == "" {
		return true
	}
	
	// Check for model match
	if query.Model != "" {
		if modelType, ok := result.Annotations[AnnotationArtifactType]; ok && modelType == "model" {
			if ref, ok := result.Annotations["org.opencontainers.image.title"]; ok && ref == query.Model {
				return true
			}
				// Check if this model is referenced in relationships
			if relGraph, ok := result.Annotations[AnnotationRelationshipGraph]; ok {
				var graph RelationshipGraph
				if err := json.Unmarshal([]byte(relGraph), &graph); err == nil {
					// This is a simplified check
					if _, exists := graph.Models[query.Model]; exists {
						return true
					}
				}
			}
		}
	}
	
	// Similar checks for other types
	return false
}

// ParseSearchResultsFromManifests parses search results from a list of manifest bytes
func ParseSearchResultsFromManifests(manifests [][]byte) (*SearchResults, error) {
	results := &SearchResults{
		Results: []SearchResult{},
	}
	
	for _, manifestBytes := range manifests {
		var manifestData map[string]interface{}
		if err := json.Unmarshal(manifestBytes, &manifestData); err != nil {
			continue // Skip malformed manifests
		}
		
		result := SearchResult{
			Annotations: make(map[string]string),
		}
		
		// Extract reference
		if ref, ok := manifestData["reference"].(string); ok {
			result.Reference = ref
		}
		
		// Extract annotations
		if annotations, ok := manifestData["annotations"].(map[string]interface{}); ok {
			for k, v := range annotations {
				if vStr, ok := v.(string); ok {
					result.Annotations[k] = vStr
				}
			}
		}
		
		// Extract artifact type
		if at, ok := manifestData[AnnotationArtifactType].(string); ok {
			result.ArtifactType = at
		}
		
		// Parse relationship graph if present
		if relGraph, ok := manifestData[AnnotationRelationshipGraph].(string); ok {
			graph, err := ParseRelationshipGraphFromAnnotation(relGraph)
			if err == nil {
				result.RelationshipGraph = graph
			}
		}
		
		// Parse metadata from annotations
		result.Metadata = parseMetadataFromAnnotations(result.Annotations)
		
		results.Results = append(results.Results, result)
		results.TotalCount++
	}
	
	return results, nil
}

// parseMetadataFromAnnotations extracts metadata from annotations
func parseMetadataFromAnnotations(annotations map[string]string) map[string]interface{} {
	metadata := make(map[string]interface{})
	
	// Parse ai.assets if present
	if assetsJSON, ok := annotations[AnnotationMetadataContract]; ok {
		var assets ContractAssetMetadata
		if err := json.Unmarshal([]byte(assetsJSON), &assets); err == nil {
			if assets.Model != nil {
				metadata["model"] = assets.Model
			}
			if assets.Skill != nil {
				metadata["skill"] = assets.Skill
			}
			if assets.Pipeline != nil {
				metadata["pipeline"] = assets.Pipeline
			}
		}
	}
	
	// Copy other AI annotations
	for k, v := range annotations {
		if strings.HasPrefix(k, "ai.") {
			metadata[k] = v
		}
	}
	
	return metadata
}

// BuildOCIFilter builds an OCI-compliant filter string for registry queries
func BuildOCIFilter(query *SearchQuery) string {
	var filters []string
	
	// Artifact type filter
	if query.ArtifactType != "" {
		filters = append(filters, fmt.Sprintf("%s=%s", AnnotationArtifactType, query.ArtifactType))
	}
	
	// Metadata filters
	for k, v := range query.MetadataFilters {
		filters = append(filters, fmt.Sprintf("%s=%s", k, v))
	}
	
	// Relationship filters (these may need client-side filtering)
	// We'll add them as annotation filters for registries that support it
	if query.Model != "" {
		filters = append(filters, fmt.Sprintf("ai.model.ref=%s", query.Model))
	}
	if query.Skill != "" {
		filters = append(filters, fmt.Sprintf("ai.skill.ref=%s", query.Skill))
	}
	if query.Pipeline != "" {
		filters = append(filters, fmt.Sprintf("ai.pipeline.ref=%s", query.Pipeline))
	}
	
	return strings.Join(filters, "&")
}
