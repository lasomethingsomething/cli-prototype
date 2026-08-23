package workflow

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ContractModelMetadata defines required and optional metadata for AI models in the metadata contract
type ContractModelMetadata struct {
	// Required fields
	Type       string `json:"type"`               // e.g., llm, embedding, classification
	Framework  string `json:"framework"`          // e.g., pytorch, tensorflow, onnx
	Input      string `json:"input,omitempty"`    // e.g., text, image, audio
	Output     string `json:"output,omitempty"`   // e.g., text, embedding, json
	Capabilities []string `json:"capabilities,omitempty"` // e.g., chat, completion, embeddings

	// Optional fields
	Description string `json:"description,omitempty"`
	Version     string `json:"version,omitempty"`
	Author      string `json:"author,omitempty"`
	License     string `json:"license,omitempty"`
	Runtime     string `json:"runtime,omitempty"`    // e.g., vllm, tensorrt-llm
	Accelerator string `json:"accelerator,omitempty"` // e.g., nvidia-gpu, cpu

	// Relationships to other assets
	Relationships map[string][]string `json:"relationships,omitempty"` // e.g., skills: [skill:v1], pipelines: [pipeline:v1]
}

// ContractSkillMetadata defines required and optional metadata for AI skills in the metadata contract
type ContractSkillMetadata struct {
	// Required fields
	Type        string `json:"type"`                // e.g., rag, classification, summarization
	PipelineRef string `json:"pipeline_ref,omitempty"` // Reference to pipeline this skill belongs to

	// Optional fields
	Description string `json:"description,omitempty"`
	Version     string `json:"version,omitempty"`
	Author      string `json:"author,omitempty"`
	Runtime     string `json:"runtime,omitempty"`
	Accelerator string `json:"accelerator,omitempty"`

	// Dependencies on other assets (required)
	Dependencies map[string][]string `json:"dependencies"` // e.g., models: [model:sha256:abc123]
}

// ContractPipelineMetadata defines required and optional metadata for AI pipelines in the metadata contract
type ContractPipelineMetadata struct {
	// Required fields
	Type   string `json:"type"`              // e.g., inference, training, fine-tuning
	Stages []string `json:"stages"`          // e.g., preprocess, inference, postprocess

	// Optional fields
	Description string `json:"description,omitempty"`
	Version     string `json:"version,omitempty"`
	Author      string `json:"author,omitempty"`

	// Dependencies on other assets
	Dependencies map[string][]string `json:"dependencies,omitempty"` // e.g., models: [model:v1], skills: [skill:v1]

	// Components in the pipeline
	Components []ContractPipelineComponent `json:"components,omitempty"`
}

// ContractPipelineComponent represents a component in a pipeline in the metadata contract
type ContractPipelineComponent struct {
	Name      string `json:"name"`
	Type      string `json:"type"`      // model, skill, preprocessor, postprocessor
	Reference string `json:"reference"` // e.g., model:v1, skill:v1
}

// MetadataContract represents the standardized metadata contract at manifest level
// This is embedded in manifest annotations as JSON under the ai.assets key
type MetadataContract struct {
	// Assets defines the AI assets and their metadata
	Assets ContractAssetMetadata `json:"ai.assets"`
}

// ContractAssetMetadata contains metadata for all AI asset types
type ContractAssetMetadata struct {
	Model    *ContractModelMetadata    `json:"model,omitempty"`
	Skill    *ContractSkillMetadata    `json:"skill,omitempty"`
	Pipeline *ContractPipelineMetadata `json:"pipeline,omitempty"`
}

// Annotation constants for the metadata contract
const (
	// AnnotationMetadataContract is the key for the standardized metadata contract
	// This contains the JSON-encoded MetadataContract
	AnnotationMetadataContract = "ai.assets"

	// Legacy support for the relationships in labels
	AnnotationRelationships = "ai.relationships"
)

// ContractValidationResult represents the result of validating a metadata contract
type ContractValidationResult struct {
	Valid          bool
	Errors        []string
	Warnings      []string
	ArtifactType  string
	RequiredFields []string
	MissingFields []string
}

// String returns a human-readable representation of the validation result
func (r *ContractValidationResult) String() string {
	if r.Valid {
		return "Valid: Metadata contract validation passed"
	}

	var sb strings.Builder
	sb.WriteString("Invalid: Metadata contract validation failed:\n")
	for _, err := range r.Errors {
		sb.WriteString(fmt.Sprintf("  - Error: %s\n", err))
	}
	for _, warn := range r.Warnings {
		sb.WriteString(fmt.Sprintf("  - Warning: %s\n", warn))
	}
	if len(r.MissingFields) > 0 {
		sb.WriteString(fmt.Sprintf("  - Missing required fields: %s\n", strings.Join(r.MissingFields, ", ")))
	}
	return sb.String()
}

// ValidateContract validates a metadata contract based on artifact type
func ValidateContract(contract *MetadataContract, artifactType ArtifactType) *ContractValidationResult {
	result := &ContractValidationResult{
		Valid:         true,
		Errors:       []string{},
		Warnings:     []string{},
		ArtifactType: string(artifactType),
	}

	// Validate based on artifact type
	switch artifactType {
	case ArtifactTypeModel:
		validateModelContract(contract, result)
	case ArtifactTypeSkill:
		validateSkillContract(contract, result)
	case ArtifactTypePipeline:
		validatePipelineContract(contract, result)
	default:
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("unknown artifact type: %s", artifactType))
	}

	return result
}

// validateModelContract validates model-specific contract requirements
func validateModelContract(contract *MetadataContract, result *ContractValidationResult) {
	if contract.Assets.Model == nil {
		result.Valid = false
		result.Errors = append(result.Errors, "model metadata is required")
		result.MissingFields = append(result.MissingFields, "ai.assets.model")
		return
	}

	model := contract.Assets.Model

	// Required fields for models
	requiredFields := []string{"type", "framework"}
	for _, field := range requiredFields {
		if isContractFieldEmpty(model, field) {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("model.%s is required", field))
			result.MissingFields = append(result.MissingFields, fmt.Sprintf("ai.assets.model.%s", field))
		}
	}

	// Validate type values
	validTypes := map[string]bool{"llm": true, "embedding": true, "classification": true, "text-generation": true, "chat": true, "completion": true}
	if model.Type != "" && !validTypes[model.Type] {
		result.Warnings = append(result.Warnings, fmt.Sprintf("unknown model type: %s", model.Type))
	}

	// Validate framework values
	validFrameworks := map[string]bool{"pytorch": true, "tensorflow": true, "onnx": true, "jax": true, "safetensors": true}
	if model.Framework != "" && !validFrameworks[model.Framework] {
		result.Warnings = append(result.Warnings, fmt.Sprintf("unknown framework: %s", model.Framework))
	}

	// Validate relationships if present
	if len(model.Relationships) > 0 {
		validateContractRelationships(model.Relationships, result)
	}
}

// validateSkillContract validates skill-specific contract requirements
func validateSkillContract(contract *MetadataContract, result *ContractValidationResult) {
	if contract.Assets.Skill == nil {
		result.Valid = false
		result.Errors = append(result.Errors, "skill metadata is required")
		result.MissingFields = append(result.MissingFields, "ai.assets.skill")
		return
	}

	skill := contract.Assets.Skill

	// Required fields for skills
	requiredFields := []string{"type", "dependencies"}
	for _, field := range requiredFields {
		if isContractFieldEmpty(skill, field) {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("skill.%s is required", field))
			result.MissingFields = append(result.MissingFields, fmt.Sprintf("ai.assets.skill.%s", field))
		}
	}

	// Validate dependencies - must have at least one model dependency
	if len(skill.Dependencies) == 0 {
		result.Valid = false
		result.Errors = append(result.Errors, "skill must have at least one dependency")
		return
	}

	// Check for model dependencies (required for skills)
	if models, ok := skill.Dependencies["models"]; !ok || len(models) == 0 {
		result.Valid = false
		result.Errors = append(result.Errors, "skill must have at least one model dependency")
		result.MissingFields = append(result.MissingFields, "ai.assets.skill.dependencies.models")
	} else {
		// Validate model references format
		for _, modelRef := range models {
			if !isValidContractAssetReference(modelRef) {
				result.Valid = false
				result.Errors = append(result.Errors, fmt.Sprintf("invalid model reference format: %s", modelRef))
			}
		}
	}

	// Validate type values
	validTypes := map[string]bool{"rag": true, "classification": true, "summarization": true, "translation": true, "general": true}
	if skill.Type != "" && !validTypes[skill.Type] {
		result.Warnings = append(result.Warnings, fmt.Sprintf("unknown skill type: %s", skill.Type))
	}
}

// validatePipelineContract validates pipeline-specific contract requirements
func validatePipelineContract(contract *MetadataContract, result *ContractValidationResult) {
	if contract.Assets.Pipeline == nil {
		result.Valid = false
		result.Errors = append(result.Errors, "pipeline metadata is required")
		result.MissingFields = append(result.MissingFields, "ai.assets.pipeline")
		return
	}

	pipeline := contract.Assets.Pipeline

	// Required fields for pipelines
	requiredFields := []string{"type", "stages"}
	for _, field := range requiredFields {
		if isContractFieldEmpty(pipeline, field) {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("pipeline.%s is required", field))
			result.MissingFields = append(result.MissingFields, fmt.Sprintf("ai.assets.pipeline.%s", field))
		}
	}

	// Validate stages - must have at least one stage
	if len(pipeline.Stages) == 0 {
		result.Valid = false
		result.Errors = append(result.Errors, "pipeline must have at least one stage")
		result.MissingFields = append(result.MissingFields, "ai.assets.pipeline.stages")
	}

	// Validate type values
	validTypes := map[string]bool{"inference": true, "training": true, "fine-tuning": true, "batch": true, "realtime": true}
	if pipeline.Type != "" && !validTypes[pipeline.Type] {
		result.Warnings = append(result.Warnings, fmt.Sprintf("unknown pipeline type: %s", pipeline.Type))
	}

	// Validate components if present
	if len(pipeline.Components) > 0 {
		for i, comp := range pipeline.Components {
			if comp.Name == "" {
				result.Valid = false
				result.Errors = append(result.Errors, fmt.Sprintf("component %d is missing name", i))
			}
			if comp.Type == "" {
				result.Valid = false
				result.Errors = append(result.Errors, fmt.Sprintf("component %d is missing type", i))
			}
			if comp.Reference == "" {
				result.Valid = false
				result.Errors = append(result.Errors, fmt.Sprintf("component %d is missing reference", i))
			}

			// Validate component type
			validComponentTypes := map[string]bool{"model": true, "skill": true, "preprocessor": true, "postprocessor": true}
			if !validComponentTypes[comp.Type] {
				result.Warnings = append(result.Warnings, fmt.Sprintf("unknown component type: %s", comp.Type))
			}

			// Validate component reference format
			if !isValidContractAssetReference(comp.Reference) {
				result.Warnings = append(result.Warnings, fmt.Sprintf("invalid component reference format: %s", comp.Reference))
			}
		}
	}

	// Validate dependencies if present
	if len(pipeline.Dependencies) > 0 {
		validateContractRelationships(pipeline.Dependencies, result)
	}
}

// isContractFieldEmpty checks if a field in a contract struct is empty
func isContractFieldEmpty(obj interface{}, fieldName string) bool {
	switch v := obj.(type) {
	case *ContractModelMetadata:
		if v == nil {
			return true
		}
		switch fieldName {
		case "type":
			return v.Type == ""
		case "framework":
			return v.Framework == ""
		case "input":
			return v.Input == ""
		case "output":
			return v.Output == ""
		case "capabilities":
			return len(v.Capabilities) == 0
		case "relationships":
			return len(v.Relationships) == 0
		}
	case *ContractSkillMetadata:
		if v == nil {
			return true
		}
		switch fieldName {
		case "type":
			return v.Type == ""
		case "pipeline_ref":
			return v.PipelineRef == ""
		case "dependencies":
			return len(v.Dependencies) == 0
		}
	case *ContractPipelineMetadata:
		if v == nil {
			return true
		}
		switch fieldName {
		case "type":
			return v.Type == ""
		case "stages":
			return len(v.Stages) == 0
		case "dependencies":
			return len(v.Dependencies) == 0
		case "components":
			return len(v.Components) == 0
		}
	}
	return false
}

// validateContractRelationships validates that relationship references are well-formed
func validateContractRelationships(relationships map[string][]string, result *ContractValidationResult) {
	validRelationshipTypes := map[string]bool{"models": true, "skills": true, "pipelines": true, "datasets": true}

	for relType, refs := range relationships {
		if !validRelationshipTypes[relType] {
			result.Warnings = append(result.Warnings, fmt.Sprintf("unknown relationship type: %s", relType))
		}

		for _, ref := range refs {
			if !isValidContractAssetReference(ref) {
				result.Warnings = append(result.Warnings, fmt.Sprintf("invalid reference format for %s: %s", relType, ref))
			}
		}
	}
}

// isValidContractAssetReference validates an asset reference format
func isValidContractAssetReference(ref string) bool {
	if ref == "" {
		return false
	}

	// Must contain at least one alphanumeric character
	hasAlphanumeric := false
	for _, c := range ref {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
			hasAlphanumeric = true
			break
		}
	}

	return hasAlphanumeric
}

// ParseMetadataContractFromAnnotations parses the metadata contract from manifest annotations
func ParseMetadataContractFromAnnotations(annotations map[string]string) (*MetadataContract, error) {
	if annotations == nil {
		return nil, fmt.Errorf("annotations map is nil")
	}

	var contractJSON string

	// Check for ai.assets annotation (primary)
	if val, ok := annotations[AnnotationMetadataContract]; ok {
		contractJSON = val
	} else {
		// Check for legacy ai.relationships
		if val, ok := annotations[AnnotationRelationships]; ok {
			return parseLegacyContractRelationships(val)
		}
		return nil, fmt.Errorf("no metadata contract found in annotations. Expected annotation: %s", AnnotationMetadataContract)
	}

	var contract MetadataContract
	if err := json.Unmarshal([]byte(contractJSON), &contract); err != nil {
		return nil, fmt.Errorf("failed to parse metadata contract JSON: %v", err)
	}

	return &contract, nil
}

// parseLegacyContractRelationships parses legacy ai.relationships format
func parseLegacyContractRelationships(relationshipsJSON string) (*MetadataContract, error) {
	var relationships map[string][]string
	if err := json.Unmarshal([]byte(relationshipsJSON), &relationships); err != nil {
		return nil, fmt.Errorf("failed to parse legacy relationships: %v", err)
	}

	contract := &MetadataContract{
		Assets: ContractAssetMetadata{},
	}

	// If we have model references, assume it's a model
	if models, ok := relationships["models"]; ok && len(models) > 0 {
		contract.Assets.Model = &ContractModelMetadata{
			Type:        "llm",
			Framework:   "pytorch",
			Relationships: relationships,
		}
	}

	// If we have skill references, assume it's a skill
	if skills, ok := relationships["skills"]; ok && len(skills) > 0 {
		contract.Assets.Skill = &ContractSkillMetadata{
			Type:         "general",
			Dependencies: relationships,
		}
	}

	// If we have pipeline references, assume it's a pipeline
	if pipelines, ok := relationships["pipelines"]; ok && len(pipelines) > 0 {
		contract.Assets.Pipeline = &ContractPipelineMetadata{
			Type:        "inference",
			Stages:      []string{"inference"},
			Dependencies: relationships,
		}
	}

	return contract, nil
}

// ValidateManifestMetadata validates manifest annotations for the metadata contract
func ValidateManifestMetadata(annotations map[string]string, artifactType ArtifactType) *ContractValidationResult {
	contract, err := ParseMetadataContractFromAnnotations(annotations)
	if err != nil {
		return &ContractValidationResult{
			Valid:         false,
			Errors:       []string{err.Error()},
			ArtifactType: string(artifactType),
			MissingFields: []string{AnnotationMetadataContract},
		}
	}

	return ValidateContract(contract, artifactType)
}

// GenerateMetadataContractAnnotation generates the annotation value for the metadata contract
func GenerateMetadataContractAnnotation(contract *MetadataContract) (string, error) {
	data, err := json.Marshal(contract)
	if err != nil {
		return "", fmt.Errorf("failed to marshal metadata contract: %v", err)
	}
	return string(data), nil
}
