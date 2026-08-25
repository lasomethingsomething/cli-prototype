package workflow

import (
	"encoding/json"
	"testing"
)

// TestUnifiedOCIManifestIntegration tests the full integration of unified OCI manifest creation
func TestUnifiedOCIManifestIntegration(t *testing.T) {
	// Test creating a unified OCI manifest for a model with all features
	layers := []OCILayer{
		{MediaType: "application/vnd.oci.image.layer.v1.tar", Digest: "sha256:abc123", Size: 1024},
	}

	manifest := NewUnifiedOCIManifest(ArtifactTypeModel, "my-model:v1", layers)

	// Set model config with all fields
	config := AIModelConfig{
		Architecture: "amd64",
		OS:           "linux",
		ModelType:    "text-generation",
		ModelFormat:  "pytorch",
		InputFormat:  "text",
		OutputFormat: "text",
		Capabilities: []string{"chat", "completion"},
		Runtime:      "vllm",
		Accelerator:  "nvidia-gpu",
		Relationships: map[string][]string{
			"skills": {"rag-skill:v1", "summarization-skill:v1"},
		},
		Description: "A text generation model",
		Version:     "1.0.0",
		Author:      "test-author",
		License:     "Apache-2.0",
	}
	manifest.SetModelConfig(config)

	// Validate
	if err := ValidateOCIManifest(manifest); err != nil {
		t.Fatalf("Validation failed: %v", err)
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// Verify JSON structure
	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// Check top-level fields
	if parsed["schemaVersion"] != float64(2) {
		t.Errorf("schemaVersion = %v, want 2", parsed["schemaVersion"])
	}
	if parsed["mediaType"] != OCIManifestMediaType {
		t.Errorf("mediaType = %s, want %s", parsed["mediaType"], OCIManifestMediaType)
	}

	// Check config
	configMap, ok := parsed["config"].(map[string]interface{})
	if !ok {
		t.Fatal("config is not a map")
	}
	if configMap["mediaType"] != AIModelConfigMediaType {
		t.Errorf("config.mediaType = %s, want %s", configMap["mediaType"], AIModelConfigMediaType)
	}

	// Check aiConfig
	aiConfig, ok := parsed["aiConfig"].(map[string]interface{})
	if !ok {
		t.Fatal("aiConfig is not a map")
	}
	if aiConfig["ai.model.type"] != "text-generation" {
		t.Errorf("ai.model.type = %s, want text-generation", aiConfig["ai.model.type"])
	}
	if aiConfig["ai.runtime"] != "vllm" {
		t.Errorf("ai.runtime = %s, want vllm", aiConfig["ai.runtime"])
	}

	// Check relationships
	relationships, ok := aiConfig["ai.model.relationships"].(map[string]interface{})
	if !ok {
		t.Fatal("ai.model.relationships is not a map")
	}
	skills, ok := relationships["skills"].([]interface{})
	if !ok {
		t.Fatal("skills is not a slice")
	}
	if len(skills) != 2 {
		t.Errorf("len(skills) = %d, want 2", len(skills))
	}

	// Check annotations
	annotations, ok := parsed["annotations"].(map[string]interface{})
	if !ok {
		t.Fatal("annotations is not a map")
	}
	if annotations[AnnotationArtifactType] != string(ArtifactTypeModel) {
		t.Errorf("artifact type annotation = %s, want %s", annotations[AnnotationArtifactType], string(ArtifactTypeModel))
	}
	if annotations["org.opencontainers.image.title"] != "my-model:v1" {
		t.Errorf("title annotation = %s, want my-model:v1", annotations["org.opencontainers.image.title"])
	}
}

// TestSkillManifestIntegration tests skill manifest creation
func TestSkillManifestIntegration(t *testing.T) {
	manifest := NewUnifiedOCIManifest(ArtifactTypeSkill, "my-skill:v1", []OCILayer{})

	config := AISkillConfig{
		Architecture: "amd64",
		OS:           "linux",
		SkillType:    "rag",
		Dependencies: map[string][]string{"models": {"model:v1"}},
		Runtime:      "python",
		Accelerator:  "cpu",
		Description:  "A RAG skill",
		Version:      "1.0.0",
		Author:       "test-author",
	}
	manifest.SetSkillConfig(config)

	if err := ValidateOCIManifest(manifest); err != nil {
		t.Fatalf("Skill validation failed: %v", err)
	}

	// Verify config media type
	if manifest.Config.MediaType != AISkillConfigMediaType {
		t.Errorf("Config.MediaType = %s, want %s", manifest.Config.MediaType, AISkillConfigMediaType)
	}
}

// TestPipelineManifestIntegration tests pipeline manifest creation
func TestPipelineManifestIntegration(t *testing.T) {
	manifest := NewUnifiedOCIManifest(ArtifactTypePipeline, "my-pipeline:v1", []OCILayer{})

	config := AIPipelineConfig{
		Architecture: "amd64",
		OS:           "linux",
		PipelineType: "inference",
		Components: []PipelineComponent{
			{Name: "model", Type: "model", Reference: "model:v1", Input: "text", Output: "text"},
			{Name: "skill", Type: "skill", Reference: "skill:v1"},
		},
		Dependencies: map[string][]string{
			"models": {"model:v1"},
			"skills": {"skill:v1"},
		},
		Description: "An inference pipeline",
		Version:     "1.0.0",
		Author:      "test-author",
	}
	manifest.SetPipelineConfig(config)

	if err := ValidateOCIManifest(manifest); err != nil {
		t.Fatalf("Pipeline validation failed: %v", err)
	}

	// Verify config media type
	if manifest.Config.MediaType != AIPipelineConfigMediaType {
		t.Errorf("Config.MediaType = %s, want %s", manifest.Config.MediaType, AIPipelineConfigMediaType)
	}

	// Verify components
	if len(config.Components) != 2 {
		t.Errorf("len(Components) = %d, want 2", len(config.Components))
	}
}
