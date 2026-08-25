package workflow

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Annotation for the relationship graph in manifest annotations
const (
	// AnnotationRelationshipGraph is the key for the relationship graph in manifest annotations
	// This contains a JSON-encoded RelationshipGraph
	AnnotationRelationshipGraph = "ai.relationships"
)

// RelationshipGraph represents a graph of relationships between AI assets
// This structure allows registries to index and query relationships efficiently
type RelationshipGraph struct {
	// Models maps model references to their dependencies and dependents
	Models map[string]*ModelNode `json:"model,omitempty"`

	// Skills maps skill references to their dependencies and dependents
	Skills map[string]*SkillNode `json:"skill,omitempty"`

	// Pipelines maps pipeline references to their dependencies and dependents
	Pipelines map[string]*PipelineNode `json:"pipeline,omitempty"`

	// Datasets maps dataset references to their consumers
	Datasets map[string]*DatasetNode `json:"dataset,omitempty"`
}

// ModelNode represents a model in the relationship graph
type ModelNode struct {
	// Reference is the full reference to this model (e.g., "model:v1", "ghcr.io/org/model:sha256:abc123")
	Reference string `json:"ref"`

	// SHA256 is the digest of this model (optional, for indexing)
	SHA256 string `json:"sha256,omitempty"`

	// Type is the model type (e.g., "llm", "embedding")
	Type string `json:"type,omitempty"`

	// Framework is the model framework (e.g., "pytorch", "tensorflow")
	Framework string `json:"framework,omitempty"`

	// UsedBy lists references of assets that depend on this model
	UsedBy []string `json:"used_by,omitempty"`

	// DependsOn lists references of assets this model depends on
	DependsOn []string `json:"depends_on,omitempty"`
}

// SkillNode represents a skill in the relationship graph
type SkillNode struct {
	// Reference is the full reference to this skill
	Reference string `json:"ref"`

	// SHA256 is the digest of this skill (optional, for indexing)
	SHA256 string `json:"sha256,omitempty"`

	// Type is the skill type (e.g., "rag", "classification")
	Type string `json:"type,omitempty"`

	// RequiresModels lists model references this skill requires
	RequiresModels []string `json:"requires_models,omitempty"`

	// RequiresPipelines lists pipeline references this skill requires
	RequiresPipelines []string `json:"requires_pipelines,omitempty"`

	// UsedBy lists references of assets that use this skill
	UsedBy []string `json:"used_by,omitempty"`

	// DependsOn lists all dependencies of this skill
	DependsOn []string `json:"depends_on,omitempty"`
}

// PipelineNode represents a pipeline in the relationship graph
type PipelineNode struct {
	// Reference is the full reference to this pipeline
	Reference string `json:"ref"`

	// SHA256 is the digest of this pipeline (optional, for indexing)
	SHA256 string `json:"sha256,omitempty"`

	// Type is the pipeline type (e.g., "inference", "training")
	Type string `json:"type,omitempty"`

	// Stages lists the stages in this pipeline
	Stages []string `json:"stages,omitempty"`

	// UsesModels lists model references this pipeline uses
	UsesModels []string `json:"uses_models,omitempty"`

	// UsesSkills lists skill references this pipeline uses
	UsesSkills []string `json:"uses_skills,omitempty"`

	// UsesDatasets lists dataset references this pipeline uses
	UsesDatasets []string `json:"uses_datasets,omitempty"`

	// DependsOn lists all dependencies of this pipeline
	DependsOn []string `json:"depends_on,omitempty"`
}

// DatasetNode represents a dataset in the relationship graph
type DatasetNode struct {
	// Reference is the full reference to this dataset
	Reference string `json:"ref"`

	// SHA256 is the digest of this dataset (optional, for indexing)
	SHA256 string `json:"sha256,omitempty"`

	// Type is the dataset type (e.g., "training", "evaluation")
	Type string `json:"type,omitempty"`

	// UsedBy lists references of assets that use this dataset
	UsedBy []string `json:"used_by,omitempty"`
}

// NewRelationshipGraph creates a new empty relationship graph
func NewRelationshipGraph() *RelationshipGraph {
	return &RelationshipGraph{
		Models:    make(map[string]*ModelNode),
		Skills:    make(map[string]*SkillNode),
		Pipelines: make(map[string]*PipelineNode),
		Datasets:  make(map[string]*DatasetNode),
	}
}

// AddModel adds a model to the relationship graph
func (rg *RelationshipGraph) AddModel(ref, sha256, modelType, framework string) *ModelNode {
	node := &ModelNode{
		Reference: ref,
		SHA256:    sha256,
		Type:      modelType,
		Framework: framework,
		UsedBy:    []string{},
		DependsOn: []string{},
	}
	rg.Models[ref] = node
	return node
}

// AddSkill adds a skill to the relationship graph
func (rg *RelationshipGraph) AddSkill(ref, sha256, skillType string) *SkillNode {
	node := &SkillNode{
		Reference:         ref,
		SHA256:            sha256,
		Type:              skillType,
		RequiresModels:    []string{},
		RequiresPipelines: []string{},
		UsedBy:            []string{},
		DependsOn:         []string{},
	}
	rg.Skills[ref] = node
	return node
}

// AddPipeline adds a pipeline to the relationship graph
func (rg *RelationshipGraph) AddPipeline(ref, sha256, pipelineType string, stages []string) *PipelineNode {
	node := &PipelineNode{
		Reference:    ref,
		SHA256:       sha256,
		Type:         pipelineType,
		Stages:       stages,
		UsesModels:   []string{},
		UsesSkills:   []string{},
		UsesDatasets: []string{},
		DependsOn:    []string{},
	}
	rg.Pipelines[ref] = node
	return node
}

// AddDataset adds a dataset to the relationship graph
func (rg *RelationshipGraph) AddDataset(ref, sha256, datasetType string) *DatasetNode {
	node := &DatasetNode{
		Reference: ref,
		SHA256:    sha256,
		Type:      datasetType,
		UsedBy:    []string{},
	}
	rg.Datasets[ref] = node
	return node
}

// AddModelToSkill adds a dependency from a skill to a model
// This means: skill requires model
func (rg *RelationshipGraph) AddModelToSkill(skillRef, modelRef string) error {
	skill, ok := rg.Skills[skillRef]
	if !ok {
		return fmt.Errorf("skill %s not found in graph", skillRef)
	}

	model, ok := rg.Models[modelRef]
	if !ok {
		return fmt.Errorf("model %s not found in graph", modelRef)
	}

	// Add to skill's requirements
	skill.RequiresModels = append(skill.RequiresModels, modelRef)
	skill.DependsOn = append(skill.DependsOn, modelRef)

	// Add to model's used_by
	model.UsedBy = append(model.UsedBy, skillRef)

	return nil
}

// AddSkillToPipeline adds a dependency from a pipeline to a skill
// This means: pipeline uses skill
func (rg *RelationshipGraph) AddSkillToPipeline(pipelineRef, skillRef string) error {
	pipeline, ok := rg.Pipelines[pipelineRef]
	if !ok {
		return fmt.Errorf("pipeline %s not found in graph", pipelineRef)
	}

	skill, ok := rg.Skills[skillRef]
	if !ok {
		return fmt.Errorf("skill %s not found in graph", skillRef)
	}

	// Add to pipeline's dependencies
	pipeline.UsesSkills = append(pipeline.UsesSkills, skillRef)
	pipeline.DependsOn = append(pipeline.DependsOn, skillRef)

	// Add to skill's used_by
	skill.UsedBy = append(skill.UsedBy, pipelineRef)

	return nil
}

// AddModelToPipeline adds a dependency from a pipeline to a model
// This means: pipeline uses model
func (rg *RelationshipGraph) AddModelToPipeline(pipelineRef, modelRef string) error {
	pipeline, ok := rg.Pipelines[pipelineRef]
	if !ok {
		return fmt.Errorf("pipeline %s not found in graph", pipelineRef)
	}

	model, ok := rg.Models[modelRef]
	if !ok {
		return fmt.Errorf("model %s not found in graph", modelRef)
	}

	// Add to pipeline's dependencies
	pipeline.UsesModels = append(pipeline.UsesModels, modelRef)
	pipeline.DependsOn = append(pipeline.DependsOn, modelRef)

	// Add to model's used_by
	model.UsedBy = append(model.UsedBy, pipelineRef)

	return nil
}

// AddDatasetToPipeline adds a dependency from a pipeline to a dataset
func (rg *RelationshipGraph) AddDatasetToPipeline(pipelineRef, datasetRef string) error {
	pipeline, ok := rg.Pipelines[pipelineRef]
	if !ok {
		return fmt.Errorf("pipeline %s not found in graph", pipelineRef)
	}

	dataset, ok := rg.Datasets[datasetRef]
	if !ok {
		return fmt.Errorf("dataset %s not found in graph", datasetRef)
	}

	pipeline.UsesDatasets = append(pipeline.UsesDatasets, datasetRef)
	pipeline.DependsOn = append(pipeline.DependsOn, datasetRef)
	dataset.UsedBy = append(dataset.UsedBy, pipelineRef)

	return nil
}

// AddDatasetToSkill adds a dependency from a skill to a dataset
func (rg *RelationshipGraph) AddDatasetToSkill(skillRef, datasetRef string) error {
	skill, ok := rg.Skills[skillRef]
	if !ok {
		return fmt.Errorf("skill %s not found in graph", skillRef)
	}

	dataset, ok := rg.Datasets[datasetRef]
	if !ok {
		return fmt.Errorf("dataset %s not found in graph", datasetRef)
	}

	// Skills can depend on datasets too
	skill.DependsOn = append(skill.DependsOn, datasetRef)
	dataset.UsedBy = append(dataset.UsedBy, skillRef)

	return nil
}

// ToMap converts the relationship graph to a map for JSON serialization
func (rg *RelationshipGraph) ToMap() map[string]interface{} {
	result := make(map[string]interface{})

	if len(rg.Models) > 0 {
		modelsMap := make(map[string]interface{})
		for ref, node := range rg.Models {
			modelsMap[ref] = node
		}
		result["model"] = modelsMap
	}

	if len(rg.Skills) > 0 {
		skillsMap := make(map[string]interface{})
		for ref, node := range rg.Skills {
			skillsMap[ref] = node
		}
		result["skill"] = skillsMap
	}

	if len(rg.Pipelines) > 0 {
		pipelinesMap := make(map[string]interface{})
		for ref, node := range rg.Pipelines {
			pipelinesMap[ref] = node
		}
		result["pipeline"] = pipelinesMap
	}

	if len(rg.Datasets) > 0 {
		datasetsMap := make(map[string]interface{})
		for ref, node := range rg.Datasets {
			datasetsMap[ref] = node
		}
		result["dataset"] = datasetsMap
	}

	return result
}

// ToJSONString converts the relationship graph to a JSON string
func (rg *RelationshipGraph) ToJSONString() (string, error) {
	data, err := json.Marshal(rg.ToMap())
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// FromJSONString parses a relationship graph from a JSON string
func FromJSONString(jsonStr string) (*RelationshipGraph, error) {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return nil, err
	}

	graph := NewRelationshipGraph()

	// Parse models
	if models, ok := data["model"].(map[string]interface{}); ok {
		for ref, nodeData := range models {
			nodeBytes, _ := json.Marshal(nodeData)
			var node ModelNode
			if err := json.Unmarshal(nodeBytes, &node); err == nil {
				graph.Models[ref] = &node
			}
		}
	}

	// Parse skills
	if skills, ok := data["skill"].(map[string]interface{}); ok {
		for ref, nodeData := range skills {
			nodeBytes, _ := json.Marshal(nodeData)
			var node SkillNode
			if err := json.Unmarshal(nodeBytes, &node); err == nil {
				graph.Skills[ref] = &node
			}
		}
	}

	// Parse pipelines
	if pipelines, ok := data["pipeline"].(map[string]interface{}); ok {
		for ref, nodeData := range pipelines {
			nodeBytes, _ := json.Marshal(nodeData)
			var node PipelineNode
			if err := json.Unmarshal(nodeBytes, &node); err == nil {
				graph.Pipelines[ref] = &node
			}
		}
	}

	// Parse datasets
	if datasets, ok := data["dataset"].(map[string]interface{}); ok {
		for ref, nodeData := range datasets {
			nodeBytes, _ := json.Marshal(nodeData)
			var node DatasetNode
			if err := json.Unmarshal(nodeBytes, &node); err == nil {
				graph.Datasets[ref] = &node
			}
		}
	}

	return graph, nil
}

// GenerateRelationshipGraphAnnotation generates the annotation value for the relationship graph
func GenerateRelationshipGraphAnnotation(graph *RelationshipGraph) (string, error) {
	if graph == nil {
		return "", nil
	}

	// If empty graph, return empty string
	if len(graph.Models) == 0 && len(graph.Skills) == 0 && len(graph.Pipelines) == 0 && len(graph.Datasets) == 0 {
		return "", nil
	}

	return graph.ToJSONString()
}

// ParseRelationshipGraphFromAnnotation parses a relationship graph from a manifest annotation
func ParseRelationshipGraphFromAnnotation(annotationValue string) (*RelationshipGraph, error) {
	if annotationValue == "" {
		return NewRelationshipGraph(), nil
	}

	return FromJSONString(annotationValue)
}

// String returns a human-readable representation of the relationship graph
func (rg *RelationshipGraph) String() string {
	var sb strings.Builder

	if len(rg.Models) > 0 {
		sb.WriteString("Models:\n")
		for ref, node := range rg.Models {
			sb.WriteString(fmt.Sprintf("  %s (sha256:%s, type:%s, framework:%s)\n",
				ref, node.SHA256, node.Type, node.Framework))
			if len(node.UsedBy) > 0 {
				sb.WriteString(fmt.Sprintf("    Used by: %s\n", strings.Join(node.UsedBy, ", ")))
			}
			if len(node.DependsOn) > 0 {
				sb.WriteString(fmt.Sprintf("    Depends on: %s\n", strings.Join(node.DependsOn, ", ")))
			}
		}
		sb.WriteString("\n")
	}

	if len(rg.Skills) > 0 {
		sb.WriteString("Skills:\n")
		for ref, node := range rg.Skills {
			sb.WriteString(fmt.Sprintf("  %s (sha256:%s, type:%s)\n",
				ref, node.SHA256, node.Type))
			if len(node.RequiresModels) > 0 {
				sb.WriteString(fmt.Sprintf("    Requires models: %s\n", strings.Join(node.RequiresModels, ", ")))
			}
			if len(node.UsedBy) > 0 {
				sb.WriteString(fmt.Sprintf("    Used by: %s\n", strings.Join(node.UsedBy, ", ")))
			}
		}
		sb.WriteString("\n")
	}

	if len(rg.Pipelines) > 0 {
		sb.WriteString("Pipelines:\n")
		for ref, node := range rg.Pipelines {
			sb.WriteString(fmt.Sprintf("  %s (sha256:%s, type:%s)\n",
				ref, node.SHA256, node.Type))
			if len(node.Stages) > 0 {
				sb.WriteString(fmt.Sprintf("    Stages: %s\n", strings.Join(node.Stages, ", ")))
			}
			if len(node.UsesModels) > 0 {
				sb.WriteString(fmt.Sprintf("    Uses models: %s\n", strings.Join(node.UsesModels, ", ")))
			}
			if len(node.UsesSkills) > 0 {
				sb.WriteString(fmt.Sprintf("    Uses skills: %s\n", strings.Join(node.UsesSkills, ", ")))
			}
			if len(node.UsesDatasets) > 0 {
				sb.WriteString(fmt.Sprintf("    Uses datasets: %s\n", strings.Join(node.UsesDatasets, ", ")))
			}
		}
		sb.WriteString("\n")
	}

	if len(rg.Datasets) > 0 {
		sb.WriteString("Datasets:\n")
		for ref, node := range rg.Datasets {
			sb.WriteString(fmt.Sprintf("  %s (sha256:%s, type:%s)\n",
				ref, node.SHA256, node.Type))
			if len(node.UsedBy) > 0 {
				sb.WriteString(fmt.Sprintf("    Used by: %s\n", strings.Join(node.UsedBy, ", ")))
			}
		}
	}

	return sb.String()
}
