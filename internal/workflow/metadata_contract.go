package workflow

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

// ContractModelMetadata defines required and optional metadata for AI models in the metadata contract
type ContractModelMetadata struct {
	// Required fields
	Type         string   `json:"type"`                   // e.g., llm, embedding, classification
	Framework    string   `json:"framework"`              // e.g., pytorch, tensorflow, onnx
	Input        string   `json:"input,omitempty"`        // e.g., text, image, audio
	Output       string   `json:"output,omitempty"`       // e.g., text, embedding, json
	Capabilities []string `json:"capabilities,omitempty"` // e.g., chat, completion, embeddings

	// Optional fields
	Description string `json:"description,omitempty"`
	Version     string `json:"version,omitempty"`
	Author      string `json:"author,omitempty"`
	License     string `json:"license,omitempty"`
	Runtime     string `json:"runtime,omitempty"`     // e.g., vllm, tensorrt-llm
	Accelerator string `json:"accelerator,omitempty"` // e.g., nvidia-gpu, cpu

	// Relationships to other assets
	Relationships map[string][]string `json:"relationships,omitempty"` // e.g., skills: [skill:v1], pipelines: [pipeline:v1]
}

// ContractSkillMetadata defines required and optional metadata for AI skills in the metadata contract
type ContractSkillMetadata struct {
	// Required fields
	Type        string `json:"type"`                   // e.g., rag, classification, summarization
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
	Type   string   `json:"type"`   // e.g., inference, training, fine-tuning
	Stages []string `json:"stages"` // e.g., preprocess, inference, postprocess

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
	Errors         []string
	Warnings       []string
	ArtifactType   string
	RequiredFields []string
	MissingFields  []string
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
		Valid:        true,
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
			Type:          "llm",
			Framework:     "pytorch",
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
			Type:         "inference",
			Stages:       []string{"inference"},
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
			Errors:        []string{err.Error()},
			ArtifactType:  string(artifactType),
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

// jsonSchema is the embedded JSON Schema for metadata contract validation
// This is defined as a constant to avoid loading from file at runtime
const jsonSchema = `{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "$id": "https://model-cli.dev/schemas/metadata-contract/v1.json",
  "title": "Standardized Metadata Contract Schema",
  "description": "Schema for validating AI asset metadata contracts in OCI manifests",
  "type": "object",
  "required": ["ai.assets"],
  "properties": {
    "ai.assets": {
      "type": "object",
      "description": "AI asset metadata",
      "oneOf": [
        {"required": ["model"], "properties": {"model": {"type": "object"}}},
        {"required": ["skill"], "properties": {"skill": {"type": "object"}}},
        {"required": ["pipeline"], "properties": {"pipeline": {"type": "object"}}}
      ]
    }
  }
}`

// compiledSchema holds the compiled JSON Schema validator
var compiledSchema *jsonschema.Schema

// getCompiledSchema returns the compiled JSON Schema, compiling it once on first use
func getCompiledSchema() (*jsonschema.Schema, error) {
	if compiledSchema != nil {
		return compiledSchema, nil
	}

	schema, err := jsonschema.CompileString("metadata-contract.json", jsonSchema)
	if err != nil {
		return nil, fmt.Errorf("failed to compile JSON schema: %v", err)
	}

	compiledSchema = schema
	return compiledSchema, nil
}

// ValidateContractWithJSONSchema validates a metadata contract using JSON Schema
// This provides an alternative validation method that can catch schema-level issues
func ValidateContractWithJSONSchema(contract *MetadataContract) (*ContractValidationResult, error) {
	result := &ContractValidationResult{
		Valid:        true,
		Errors:       []string{},
		Warnings:     []string{},
		ArtifactType: "unknown",
	}

	if contract == nil {
		result.Valid = false
		result.Errors = append(result.Errors, "metadata contract is nil")
		return result, nil
	}

	// Determine artifact type from contract
	if contract.Assets.Model != nil {
		result.ArtifactType = "model"
	} else if contract.Assets.Skill != nil {
		result.ArtifactType = "skill"
	} else if contract.Assets.Pipeline != nil {
		result.ArtifactType = "pipeline"
	}

	// Compile and get the schema
	schema, err := getCompiledSchema()
	if err != nil {
		// Schema compilation failed - fall back to programmatic validation
		return ValidateContract(contract, ArtifactType(result.ArtifactType)), nil
	}

	// Convert contract to map for validation
	contractData, err := contract.ToMap()
	if err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("failed to convert contract to map: %v", err))
		return result, nil
	}

	// Validate against schema
	validationErr := schema.Validate(contractData)
	if validationErr != nil {
		result.Valid = false
		// Extract error details
		var errs []string
		for _, desc := range strings.Split(validationErr.Error(), "\n") {
			if desc != "" {
				errs = append(errs, strings.TrimSpace(desc))
			}
		}
		result.Errors = append(result.Errors, errs...)
	}

	return result, nil
}

// ToMap converts the MetadataContract to a map for JSON Schema validation
func (c *MetadataContract) ToMap() (map[string]interface{}, error) {
	result := make(map[string]interface{})

	assetsMap := make(map[string]interface{})

	if c.Assets.Model != nil {
		modelMap := make(map[string]interface{})
		// Add required fields
		if c.Assets.Model.Type != "" {
			modelMap["type"] = c.Assets.Model.Type
		}
		if c.Assets.Model.Framework != "" {
			modelMap["framework"] = c.Assets.Model.Framework
		}
		// Add optional fields
		if c.Assets.Model.Input != "" {
			modelMap["input"] = c.Assets.Model.Input
		}
		if c.Assets.Model.Output != "" {
			modelMap["output"] = c.Assets.Model.Output
		}
		if len(c.Assets.Model.Capabilities) > 0 {
			modelMap["capabilities"] = c.Assets.Model.Capabilities
		}
		if c.Assets.Model.Description != "" {
			modelMap["description"] = c.Assets.Model.Description
		}
		if c.Assets.Model.Version != "" {
			modelMap["version"] = c.Assets.Model.Version
		}
		if c.Assets.Model.Author != "" {
			modelMap["author"] = c.Assets.Model.Author
		}
		if c.Assets.Model.License != "" {
			modelMap["license"] = c.Assets.Model.License
		}
		if c.Assets.Model.Runtime != "" {
			modelMap["runtime"] = c.Assets.Model.Runtime
		}
		if c.Assets.Model.Accelerator != "" {
			modelMap["accelerator"] = c.Assets.Model.Accelerator
		}
		if len(c.Assets.Model.Relationships) > 0 {
			modelMap["relationships"] = c.Assets.Model.Relationships
		}
		assetsMap["model"] = modelMap
	}

	if c.Assets.Skill != nil {
		skillMap := make(map[string]interface{})
		// Add required fields
		if c.Assets.Skill.Type != "" {
			skillMap["type"] = c.Assets.Skill.Type
		}
		if len(c.Assets.Skill.Dependencies) > 0 {
			skillMap["dependencies"] = c.Assets.Skill.Dependencies
		}
		// Add optional fields
		if c.Assets.Skill.PipelineRef != "" {
			skillMap["pipeline_ref"] = c.Assets.Skill.PipelineRef
		}
		if c.Assets.Skill.Description != "" {
			skillMap["description"] = c.Assets.Skill.Description
		}
		if c.Assets.Skill.Version != "" {
			skillMap["version"] = c.Assets.Skill.Version
		}
		if c.Assets.Skill.Author != "" {
			skillMap["author"] = c.Assets.Skill.Author
		}
		if c.Assets.Skill.Runtime != "" {
			skillMap["runtime"] = c.Assets.Skill.Runtime
		}
		if c.Assets.Skill.Accelerator != "" {
			skillMap["accelerator"] = c.Assets.Skill.Accelerator
		}
		assetsMap["skill"] = skillMap
	}

	if c.Assets.Pipeline != nil {
		pipelineMap := make(map[string]interface{})
		// Add required fields
		if c.Assets.Pipeline.Type != "" {
			pipelineMap["type"] = c.Assets.Pipeline.Type
		}
		if len(c.Assets.Pipeline.Stages) > 0 {
			pipelineMap["stages"] = c.Assets.Pipeline.Stages
		}
		// Add optional fields
		if c.Assets.Pipeline.Description != "" {
			pipelineMap["description"] = c.Assets.Pipeline.Description
		}
		if c.Assets.Pipeline.Version != "" {
			pipelineMap["version"] = c.Assets.Pipeline.Version
		}
		if c.Assets.Pipeline.Author != "" {
			pipelineMap["author"] = c.Assets.Pipeline.Author
		}
		if len(c.Assets.Pipeline.Dependencies) > 0 {
			pipelineMap["dependencies"] = c.Assets.Pipeline.Dependencies
		}
		if len(c.Assets.Pipeline.Components) > 0 {
			var comps []map[string]interface{}
			for _, comp := range c.Assets.Pipeline.Components {
				compMap := map[string]interface{}{
					"name":      comp.Name,
					"type":      comp.Type,
					"reference": comp.Reference,
				}
				comps = append(comps, compMap)
			}
			pipelineMap["components"] = comps
		}
		assetsMap["pipeline"] = pipelineMap
	}

	if len(assetsMap) > 0 {
		result["ai.assets"] = assetsMap
	}

	return result, nil
}

// ValidateContractWithStrictJSONSchema validates a metadata contract strictly using JSON Schema
// This is the primary validation method for Story #62
func ValidateContractWithStrictJSONSchema(contractJSON string) (*ContractValidationResult, error) {
	result := &ContractValidationResult{
		Valid:        true,
		Errors:       []string{},
		Warnings:     []string{},
		ArtifactType: "unknown",
	}

	if contractJSON == "" {
		result.Valid = false
		result.Errors = append(result.Errors, "metadata contract JSON is empty")
		result.MissingFields = append(result.MissingFields, AnnotationMetadataContract)
		return result, nil
	}

	// Parse the JSON
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(contractJSON), &data); err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("failed to parse metadata contract JSON: %v", err))
		return result, nil
	}

	// Compile and get the schema
	schema, err := getCompiledSchema()
	if err != nil {
		// Fall back to programmatic validation
		contract, parseErr := ParseMetadataContractFromAnnotations(map[string]string{
			AnnotationMetadataContract: contractJSON,
		})
		if parseErr != nil {
			result.Valid = false
			result.Errors = append(result.Errors, parseErr.Error())
			return result, nil
		}
		return ValidateContract(contract, ArtifactType(result.ArtifactType)), nil
	}

	// Validate against schema
	validationErr := schema.Validate(data)
	if validationErr != nil {
		result.Valid = false
		// Extract error details
		for _, desc := range strings.Split(validationErr.Error(), "\n") {
			if desc != "" {
				errMsg := strings.TrimSpace(desc)
				// Clean up the error message
				errMsg = strings.TrimPrefix(errMsg, "(root): ")
				errMsg = strings.TrimPrefix(errMsg, "(root):")
				errMsg = strings.TrimSpace(errMsg)
				if errMsg != "" {
					result.Errors = append(result.Errors, errMsg)
				}
			}
		}

		// Determine which fields are missing based on schema requirements
		// The schema requires ai.assets, and within that, one of model/skill/pipeline
		if _, hasAssets := data["ai.assets"]; !hasAssets {
			result.MissingFields = append(result.MissingFields, "ai.assets")
		} else {
			assets := data["ai.assets"].(map[string]interface{})
			if len(assets) == 0 {
				result.MissingFields = append(result.MissingFields, "ai.assets.<type>")
				result.Errors = append(result.Errors, "ai.assets must contain at least one of: model, skill, pipeline")
			} else {
				// Check for required fields in each type
				for assetType, assetData := range assets {
					assetMap := assetData.(map[string]interface{})
					switch assetType {
					case "model":
						result.ArtifactType = "model"
						if _, hasType := assetMap["type"]; !hasType {
							result.MissingFields = append(result.MissingFields, "ai.assets.model.type")
						}
						if _, hasFramework := assetMap["framework"]; !hasFramework {
							result.MissingFields = append(result.MissingFields, "ai.assets.model.framework")
						}
					case "skill":
						result.ArtifactType = "skill"
						if _, hasType := assetMap["type"]; !hasType {
							result.MissingFields = append(result.MissingFields, "ai.assets.skill.type")
						}
						if _, hasDeps := assetMap["dependencies"]; !hasDeps {
							result.MissingFields = append(result.MissingFields, "ai.assets.skill.dependencies")
						}
					case "pipeline":
						result.ArtifactType = "pipeline"
						if _, hasType := assetMap["type"]; !hasType {
							result.MissingFields = append(result.MissingFields, "ai.assets.pipeline.type")
						}
						if _, hasStages := assetMap["stages"]; !hasStages {
							result.MissingFields = append(result.MissingFields, "ai.assets.pipeline.stages")
						}
					}
				}
			}
		}
	}

	return result, nil
}
