package workflow

import (
	"encoding/json"
	"fmt"
	"os"
)

// OCI Image Spec media types
const (
	// Media types for manifests
	OCIManifestMediaType      = "application/vnd.oci.image.manifest.v1+json"
	OCIIndexMediaType         = "application/vnd.oci.image.index.v1+json"
	
	// Media types for AI-specific config
	AIModelConfigMediaType    = "application/vnd.cncf.ai.model.config.v1+json"
	AISkillConfigMediaType    = "application/vnd.cncf.ai.skill.config.v1+json"
	AIPipelineConfigMediaType = "application/vnd.cncf.ai.pipeline.config.v1+json"
)

// OCIDescriptor is the full OCI descriptor as per OCI Image Spec
type OCIDescriptor struct {
	MediaType    string `json:"mediaType"`
	Digest      string `json:"digest"`
	Size        int64  `json:"size"`
	Annotations map[string]string `json:"annotations,omitempty"`
	Data        []byte `json:"data,omitempty"` // For inlining small blobs
	URLs        []string `json:"urls,omitempty"` // For referencing external blobs
}

// OCILayer represents a layer in an OCI manifest
type OCILayer struct {
	MediaType string            `json:"mediaType"`
	Digest    string            `json:"digest"`
	Size      int64             `json:"size"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// AIModelConfig represents AI-specific configuration for a model
type AIModelConfig struct {
	// Standard OCI config fields
	Architecture string `json:"architecture,omitempty"`
	OS           string `json:"os,omitempty"`
	
	// AI-specific fields
	ModelType    string `json:"ai.model.type"`            // e.g., "text-generation", "embedding", "classification"
	ModelFormat  string `json:"ai.model.format"`          // e.g., "pytorch", "tensorflow", "onnx"
	InputFormat  string `json:"ai.model.input.format"`    // e.g., "text", "image", "audio"
	OutputFormat string `json:"ai.model.output.format"`   // e.g., "text", "embedding", "json"
	
	// Model capabilities
	Capabilities []string `json:"ai.model.capabilities,omitempty"` // e.g., ["chat", "completion", "embeddings"]
	
	// Runtime requirements
	Runtime      string `json:"ai.runtime,omitempty"`      // e.g., "vllm", "tensorrt-llm"
	Accelerator  string `json:"ai.accelerator,omitempty"`  // e.g., "nvidia-gpu", "cpu"
	
	// Relationships to other assets
	Relationships map[string][]string `json:"ai.model.relationships,omitempty"` // e.g., {"skills": ["skill:v1"], "pipelines": ["pipeline:v1"]}
	
	// Metadata
	Description string `json:"ai.model.description,omitempty"`
	Version     string `json:"ai.model.version,omitempty"`
	Author      string `json:"ai.model.author,omitempty"`
	License     string `json:"ai.model.license,omitempty"`
}

// AISkillConfig represents AI-specific configuration for a skill
type AISkillConfig struct {
	// Standard OCI config fields
	Architecture string `json:"architecture,omitempty"`
	OS           string `json:"os,omitempty"`
	
	// AI-specific fields
	SkillType string `json:"ai.skill.type"` // e.g., "rag", "classification", "summarization"
	
	// Dependencies on other assets
	Dependencies map[string][]string `json:"ai.skill.dependencies,omitempty"` // e.g., {"models": ["model:v1"], "pipelines": ["pipeline:v1"]}
	
	// Execution requirements
	Runtime     string `json:"ai.runtime,omitempty"`
	Accelerator string `json:"ai.accelerator,omitempty"`
	
	// Metadata
	Description string `json:"ai.skill.description,omitempty"`
	Version     string `json:"ai.skill.version,omitempty"`
	Author      string `json:"ai.skill.author,omitempty"`
}

// AIPipelineConfig represents AI-specific configuration for a pipeline
type AIPipelineConfig struct {
	// Standard OCI config fields
	Architecture string `json:"architecture,omitempty"`
	OS           string `json:"os,omitempty"`
	
	// AI-specific fields
	PipelineType string `json:"ai.pipeline.type"` // e.g., "inference", "training", "fine-tuning"
	
	// Pipeline components (nodes)
	Components []PipelineComponent `json:"ai.pipeline.components,omitempty"`
	
	// Dependencies
	Dependencies map[string][]string `json:"ai.pipeline.dependencies,omitempty"` // e.g., {"models": ["model:v1"], "skills": ["skill:v1"]}
	
	// Metadata
	Description string `json:"ai.pipeline.description,omitempty"`
	Version     string `json:"ai.pipeline.version,omitempty"`
	Author      string `json:"ai.pipeline.author,omitempty"`
}

// PipelineComponent represents a component in a pipeline
type PipelineComponent struct {
	Name        string `json:"name"`
	Type        string `json:"type"` // "model", "skill", "preprocessor", "postprocessor"
	Reference   string `json:"reference"` // e.g., "model:v1", "skill:v1"
	Input      string `json:"input,omitempty"`
	Output     string `json:"output,omitempty"`
	Parameters map[string]string `json:"parameters,omitempty"`
}

// UnifiedOCIManifest represents an OCI-compliant manifest with AI-specific extensions
type UnifiedOCIManifest struct {
	// OCI Image Spec v1.1 fields
	SchemaVersion int    `json:"schemaVersion"` // Must be 2
	MediaType     string `json:"mediaType"`      // application/vnd.oci.image.manifest.v1+json
	Config        OCIDescriptor `json:"config"`
	Layers        []OCILayer    `json:"layers,omitempty"`
	Annotations   map[string]string `json:"annotations,omitempty"`
	
	// AI-specific: the actual config data (can be inlined for small configs)
	// In production, this would typically be a separate blob referenced by Config.Digest
	AIConfig interface{} `json:"aiConfig,omitempty"` // AIModelConfig, AISkillConfig, or AIPipelineConfig
}

// ArtifactType represents the type of AI artifact
type ArtifactType string

const (
	ArtifactTypeModel    ArtifactType = "model"
	ArtifactTypeSkill    ArtifactType = "skill"
	ArtifactTypePipeline ArtifactType = "pipeline"
)

// NewUnifiedOCIManifest creates a new unified OCI manifest for the given artifact type
func NewUnifiedOCIManifest(artifactType ArtifactType, name string, layers []OCILayer) *UnifiedOCIManifest {
	manifest := &UnifiedOCIManifest{
		SchemaVersion: 2,
		MediaType:     OCIManifestMediaType,
		Layers:        layers,
		Annotations:   make(map[string]string),
	}
	
	// Set artifact type annotation
	manifest.Annotations[AnnotationArtifactType] = string(artifactType)
	
	// Set the config based on artifact type
	switch artifactType {
	case ArtifactTypeModel:
		config := AIModelConfig{
			Architecture: "amd64",
			OS:           "linux",
			ModelType:    "text-generation", // Default
		}
		manifest.AIConfig = config
		manifest.Config.MediaType = AIModelConfigMediaType
	case ArtifactTypeSkill:
		config := AISkillConfig{
			Architecture: "amd64",
			OS:           "linux",
			SkillType:    "general", // Default
		}
		manifest.AIConfig = config
		manifest.Config.MediaType = AISkillConfigMediaType
	case ArtifactTypePipeline:
		config := AIPipelineConfig{
			Architecture: "amd64",
			OS:           "linux",
			PipelineType: "inference", // Default
		}
		manifest.AIConfig = config
		manifest.Config.MediaType = AIPipelineConfigMediaType
	}
	
	// Set the name annotation
	manifest.Annotations["org.opencontainers.image.title"] = name
	
	return manifest
}

// SetModelConfig sets the model-specific configuration
func (m *UnifiedOCIManifest) SetModelConfig(config AIModelConfig) {
	m.AIConfig = config
	m.Config.MediaType = AIModelConfigMediaType
	m.Annotations[AnnotationArtifactType] = string(ArtifactTypeModel)
}

// SetSkillConfig sets the skill-specific configuration
func (m *UnifiedOCIManifest) SetSkillConfig(config AISkillConfig) {
	m.AIConfig = config
	m.Config.MediaType = AISkillConfigMediaType
	m.Annotations[AnnotationArtifactType] = string(ArtifactTypeSkill)
}

// SetPipelineConfig sets the pipeline-specific configuration
func (m *UnifiedOCIManifest) SetPipelineConfig(config AIPipelineConfig) {
	m.AIConfig = config
	m.Config.MediaType = AIPipelineConfigMediaType
	m.Annotations[AnnotationArtifactType] = string(ArtifactTypePipeline)
}

// AddLayer adds a layer to the manifest
func (m *UnifiedOCIManifest) AddLayer(layer OCILayer) {
	m.Layers = append(m.Layers, layer)
}

// SetRelationships sets the relationships for a model
func (m *UnifiedOCIManifest) SetRelationships(relationships map[string][]string) {
	if m.AIConfig != nil {
		switch config := m.AIConfig.(type) {
		case AIModelConfig:
			config.Relationships = relationships
			m.AIConfig = config
		case AISkillConfig:
			config.Dependencies = relationships
			m.AIConfig = config
		case AIPipelineConfig:
			config.Dependencies = relationships
			m.AIConfig = config
		}
	}
}

// WriteUnifiedOCIManifest writes the manifest to a file
func WriteUnifiedOCIManifest(m *UnifiedOCIManifest, path string) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal unified OCI manifest: %v", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write unified OCI manifest to %s: %v", path, err)
	}
	return nil
}

// ReadUnifiedOCIManifest reads a unified OCI manifest from a file
func ReadUnifiedOCIManifest(path string) (*UnifiedOCIManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read unified OCI manifest from %s: %v", path, err)
	}
	var m UnifiedOCIManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("failed to parse unified OCI manifest at %s: %v", path, err)
	}
	return &m, nil
}

// ValidateOCIManifest validates that a manifest conforms to OCI Image Spec
func ValidateOCIManifest(m *UnifiedOCIManifest) error {
	// Check required fields
	if m.SchemaVersion != 2 {
		return fmt.Errorf("invalid schema version: expected 2, got %d", m.SchemaVersion)
	}
	if m.MediaType != OCIManifestMediaType {
		return fmt.Errorf("invalid media type: expected %s, got %s", OCIManifestMediaType, m.MediaType)
	}
	if m.Config.MediaType == "" {
		return fmt.Errorf("config media type is required")
	}
	
	// Validate artifact type
	artifactType := m.Annotations[AnnotationArtifactType]
	if artifactType == "" {
		return fmt.Errorf("artifact type annotation is required: %s", AnnotationArtifactType)
	}
	
	// Validate based on artifact type
	switch artifactType {
	case string(ArtifactTypeModel):
		if m.Config.MediaType != AIModelConfigMediaType {
			return fmt.Errorf("model config media type should be %s", AIModelConfigMediaType)
		}
	case string(ArtifactTypeSkill):
		if m.Config.MediaType != AISkillConfigMediaType {
			return fmt.Errorf("skill config media type should be %s", AISkillConfigMediaType)
		}
	case string(ArtifactTypePipeline):
		if m.Config.MediaType != AIPipelineConfigMediaType {
			return fmt.Errorf("pipeline config media type should be %s", AIPipelineConfigMediaType)
		}
	default:
		return fmt.Errorf("unknown artifact type: %s", artifactType)
	}
	
	return nil
}
