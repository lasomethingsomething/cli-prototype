package workflow

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// OCI Image Spec media types
const (
	// Media types for manifests
	OCIManifestMediaType = "application/vnd.oci.image.manifest.v1+json"
	OCIIndexMediaType    = "application/vnd.oci.image.index.v1+json"

	// Media type used for each packaged file (ORAS's default for files)
	OCILayerMediaType = "application/vnd.oci.image.layer.v1.tar"

	// Media types for AI-specific config
	AIModelConfigMediaType    = "application/vnd.cncf.ai.model.config.v1+json"
	AISkillConfigMediaType    = "application/vnd.cncf.ai.skill.config.v1+json"
	AIPipelineConfigMediaType = "application/vnd.cncf.ai.pipeline.config.v1+json"
)

// OCIDescriptor is the full OCI descriptor as per OCI Image Spec
type OCIDescriptor struct {
	MediaType   string            `json:"mediaType"`
	Digest      string            `json:"digest"`
	Size        int64             `json:"size"`
	Annotations map[string]string `json:"annotations,omitempty"`
	Data        []byte            `json:"data,omitempty"` // For inlining small blobs
	URLs        []string          `json:"urls,omitempty"` // For referencing external blobs
}

// OCILayer represents a layer in an OCI manifest
type OCILayer struct {
	MediaType   string            `json:"mediaType"`
	Digest      string            `json:"digest"`
	Size        int64             `json:"size"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// AIModelConfig represents AI-specific configuration for a model
type AIModelConfig struct {
	// Standard OCI config fields
	Architecture string `json:"architecture,omitempty"`
	OS           string `json:"os,omitempty"`

	// AI-specific fields
	ModelType    string `json:"ai.model.type"`          // e.g., "text-generation", "embedding", "classification"
	ModelFormat  string `json:"ai.model.format"`        // e.g., "pytorch", "tensorflow", "onnx"
	InputFormat  string `json:"ai.model.input.format"`  // e.g., "text", "image", "audio"
	OutputFormat string `json:"ai.model.output.format"` // e.g., "text", "embedding", "json"

	// Model capabilities
	Capabilities []string `json:"ai.model.capabilities,omitempty"` // e.g., ["chat", "completion", "embeddings"]

	// Runtime requirements
	Runtime     string `json:"ai.runtime,omitempty"`              // e.g., "vllm", "tensorrt-llm"
	Accelerator string `json:"ai.accelerator,omitempty"`          // e.g., "nvidia-gpu", "cpu"
	CUDAMin     string `json:"ai.accelerator.cuda.min,omitempty"` // e.g., "12.1", "11.8"
	MemoryMin   string `json:"ai.resource.memory.min,omitempty"`  // e.g., "24GiB", "16Gi"

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
	CUDAMin     string `json:"ai.accelerator.cuda.min,omitempty"`
	MemoryMin   string `json:"ai.resource.memory.min,omitempty"`

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

	// Runtime requirements
	Runtime     string `json:"ai.runtime,omitempty"`
	Accelerator string `json:"ai.accelerator,omitempty"`
	CUDAMin     string `json:"ai.accelerator.cuda.min,omitempty"`
	MemoryMin   string `json:"ai.resource.memory.min,omitempty"`

	// Metadata
	Description string `json:"ai.pipeline.description,omitempty"`
	Version     string `json:"ai.pipeline.version,omitempty"`
	Author      string `json:"ai.pipeline.author,omitempty"`
}

// PipelineComponent represents a component in a pipeline
type PipelineComponent struct {
	Name       string            `json:"name"`
	Type       string            `json:"type"`      // "model", "skill", "preprocessor", "postprocessor"
	Reference  string            `json:"reference"` // e.g., "model:v1", "skill:v1"
	Input      string            `json:"input,omitempty"`
	Output     string            `json:"output,omitempty"`
	Parameters map[string]string `json:"parameters,omitempty"`
}

// UnifiedOCIManifest is the single manifest type used throughout model-cli:
// an OCI image manifest (v1.1) carrying CNCF AI annotations, plus an inline
// copy of the AI config for tools that inspect the manifest without pulling
// the config blob.
type UnifiedOCIManifest struct {
	// OCI Image Spec v1.1 fields
	SchemaVersion int               `json:"schemaVersion"`          // Must be 2
	MediaType     string            `json:"mediaType"`              // application/vnd.oci.image.manifest.v1+json
	ArtifactType  string            `json:"artifactType,omitempty"` // application/vnd.cncf.ai.<model|skill|pipeline>
	Config        OCIDescriptor     `json:"config"`
	Layers        []OCILayer        `json:"layers,omitempty"`
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

	// Set artifact type annotation and OCI artifactType
	manifest.Annotations[AnnotationArtifactType] = string(artifactType)
	manifest.ArtifactType = "application/vnd.cncf.ai." + string(artifactType)

	// Set the config based on artifact type
	switch artifactType {
	case ArtifactTypeModel:
		manifest.SetModelConfig(AIModelConfig{
			Architecture: "amd64",
			OS:           "linux",
			ModelType:    "text-generation", // Default
		})
	case ArtifactTypeSkill:
		manifest.SetSkillConfig(AISkillConfig{
			Architecture: "amd64",
			OS:           "linux",
			SkillType:    "general", // Default
		})
	case ArtifactTypePipeline:
		manifest.SetPipelineConfig(AIPipelineConfig{
			Architecture: "amd64",
			OS:           "linux",
			PipelineType: "inference", // Default
		})
	}

	// Set the name annotation
	manifest.Annotations["org.opencontainers.image.title"] = name

	return manifest
}

// SetModelConfig sets the model-specific configuration
func (m *UnifiedOCIManifest) SetModelConfig(config AIModelConfig) {
	m.setAIConfig(config, AIModelConfigMediaType, ArtifactTypeModel)
}

// SetSkillConfig sets the skill-specific configuration
func (m *UnifiedOCIManifest) SetSkillConfig(config AISkillConfig) {
	m.setAIConfig(config, AISkillConfigMediaType, ArtifactTypeSkill)
}

// SetPipelineConfig sets the pipeline-specific configuration
func (m *UnifiedOCIManifest) SetPipelineConfig(config AIPipelineConfig) {
	m.setAIConfig(config, AIPipelineConfigMediaType, ArtifactTypePipeline)
}

// setAIConfig stores the AI config and keeps the config descriptor in sync
// with it, so a manifest validates and describes the right blob right after
// its config is set. The config types are plain structs whose JSON
// serialization cannot fail; should it ever, the descriptor is left empty
// and ValidateOCIManifest reports the missing digest.
func (m *UnifiedOCIManifest) setAIConfig(config interface{}, mediaType string, artifactType ArtifactType) {
	m.AIConfig = config
	m.Config.MediaType = mediaType
	m.Annotations[AnnotationArtifactType] = string(artifactType)
	if err := m.refreshConfigDescriptor(); err != nil {
		m.Config.Digest, m.Config.Size = "", 0
	}
}

// AddLayer adds a layer to the manifest
func (m *UnifiedOCIManifest) AddLayer(layer OCILayer) {
	m.Layers = append(m.Layers, layer)
}

// SetRelationships sets the relationships for a model
func (m *UnifiedOCIManifest) SetRelationships(relationships map[string][]string) {
	switch config := m.AIConfig.(type) {
	case AIModelConfig:
		config.Relationships = relationships
		m.SetModelConfig(config)
	case AISkillConfig:
		config.Dependencies = relationships
		m.SetSkillConfig(config)
	case AIPipelineConfig:
		config.Dependencies = relationships
		m.SetPipelineConfig(config)
	}
}

// ConfigBlobFileName is the file next to manifest.json that holds the
// serialized AI config. It is pushed as the manifest's config blob.
const ConfigBlobFileName = "config.json"

// configBlob returns the bytes of the AI config blob, or nil when the
// manifest carries no AI config. This is the single serialization of the
// config: the config descriptor's digest and size are computed from it, and
// WriteUnifiedOCIManifest writes exactly these bytes to config.json, so the
// descriptor always matches the blob that is pushed.
func (m *UnifiedOCIManifest) configBlob() ([]byte, error) {
	if m.AIConfig == nil {
		return nil, nil
	}
	data, err := json.MarshalIndent(m.AIConfig, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal AI config: %v", err)
	}
	return data, nil
}

// refreshConfigDescriptor fills the config descriptor's digest and size from
// the AI config blob so the descriptor describes a real blob.
func (m *UnifiedOCIManifest) refreshConfigDescriptor() error {
	data, err := m.configBlob()
	if err != nil || data == nil {
		return err
	}
	m.Config.Digest = fmt.Sprintf("sha256:%x", sha256.Sum256(data))
	m.Config.Size = int64(len(data))
	return nil
}

// WriteUnifiedOCIManifest writes the manifest to path and the AI config blob
// to config.json in the same directory. The config descriptor is always
// refreshed first, so the written manifest's config digest and size match
// the config.json bytes.
func WriteUnifiedOCIManifest(m *UnifiedOCIManifest, path string) error {
	if err := m.refreshConfigDescriptor(); err != nil {
		return err
	}
	configData, err := m.configBlob()
	if err != nil {
		return err
	}
	if configData != nil {
		configPath := filepath.Join(filepath.Dir(path), ConfigBlobFileName)
		if err := os.WriteFile(configPath, configData, 0644); err != nil {
			return fmt.Errorf("failed to write AI config blob to %s: %v", configPath, err)
		}
	}

	// The AI config stays inlined as aiConfig for readers that inspect the
	// local manifest without the blob (enforce, admission).
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
	if m.Config.Digest == "" {
		return fmt.Errorf("config digest is required")
	}
	if m.Config.Size <= 0 {
		return fmt.Errorf("config size must be positive, got %d", m.Config.Size)
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

// NewManifestFromDirectory builds a manifest for the files under dir: one
// layer per file with its sha256 digest, size and relative path as title.
// Files that model-cli writes next to the model (manifest, attestation,
// SBOM) are skipped. annotations are merged into the manifest; the artifact
// type annotation is always set from artifactType.
func NewManifestFromDirectory(artifactType ArtifactType, name, dir string, annotations map[string]string) (*UnifiedOCIManifest, error) {
	layers, err := layersFromDirectory(dir)
	if err != nil {
		return nil, err
	}
	manifest := NewUnifiedOCIManifest(artifactType, name, layers)
	for k, v := range annotations {
		manifest.Annotations[k] = v
	}
	manifest.Annotations[AnnotationArtifactType] = string(artifactType)
	if err := manifest.refreshConfigDescriptor(); err != nil {
		return nil, err
	}
	return manifest, nil
}

// layersFromDirectory returns a layer descriptor per regular file under dir,
// sorted by path for deterministic output. A single file is a single layer.
func layersFromDirectory(dir string) ([]OCILayer, error) {
	info, err := os.Stat(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read model path %s: %v", dir, err)
	}
	if !info.IsDir() {
		layer, err := layerForFile(dir, filepath.Base(dir))
		if err != nil {
			return nil, err
		}
		return []OCILayer{layer}, nil
	}

	var layers []OCILayer
	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !info.Mode().IsRegular() {
			return nil
		}
		if isGeneratedArtifact(strings.ToLower(info.Name())) {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		layer, err := layerForFile(path, filepath.ToSlash(rel))
		if err != nil {
			return err
		}
		layers = append(layers, layer)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to scan model path %s: %v", dir, err)
	}
	sort.Slice(layers, func(i, j int) bool {
		return layers[i].Annotations["org.opencontainers.image.title"] < layers[j].Annotations["org.opencontainers.image.title"]
	})
	return layers, nil
}

func layerForFile(path, title string) (OCILayer, error) {
	f, err := os.Open(path)
	if err != nil {
		return OCILayer{}, fmt.Errorf("failed to open %s: %v", path, err)
	}
	defer f.Close()

	h := sha256.New()
	size, err := io.Copy(h, f)
	if err != nil {
		return OCILayer{}, fmt.Errorf("failed to hash %s: %v", path, err)
	}
	return OCILayer{
		MediaType:   OCILayerMediaType,
		Digest:      fmt.Sprintf("sha256:%x", h.Sum(nil)),
		Size:        size,
		Annotations: map[string]string{"org.opencontainers.image.title": title},
	}, nil
}

// ComputeManifestDigest returns the sha256 digest of the manifest file at
// path, or "" if it cannot be read.
func ComputeManifestDigest(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("sha256:%x", sha256.Sum256(data))
}
