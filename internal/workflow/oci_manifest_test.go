package workflow

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestNewUnifiedOCIManifest(t *testing.T) {
	tests := []struct {
		name           string
		artifactType   ArtifactType
		wantConfigType string
	}{
		{
			name:           "model manifest",
			artifactType:   ArtifactTypeModel,
			wantConfigType: AIModelConfigMediaType,
		},
		{
			name:           "skill manifest",
			artifactType:   ArtifactTypeSkill,
			wantConfigType: AISkillConfigMediaType,
		},
		{
			name:           "pipeline manifest",
			artifactType:   ArtifactTypePipeline,
			wantConfigType: AIPipelineConfigMediaType,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manifest := NewUnifiedOCIManifest(tt.artifactType, "test-artifact", []OCILayer{})

			if manifest.SchemaVersion != 2 {
				t.Errorf("SchemaVersion = %d, want 2", manifest.SchemaVersion)
			}
			if manifest.MediaType != OCIManifestMediaType {
				t.Errorf("MediaType = %s, want %s", manifest.MediaType, OCIManifestMediaType)
			}
			if manifest.Config.MediaType != tt.wantConfigType {
				t.Errorf("Config.MediaType = %s, want %s", manifest.Config.MediaType, tt.wantConfigType)
			}
			if manifest.Annotations[AnnotationArtifactType] != string(tt.artifactType) {
				t.Errorf("Annotation %s = %s, want %s", AnnotationArtifactType, manifest.Annotations[AnnotationArtifactType], string(tt.artifactType))
			}
		})
	}
}

func TestSetModelConfig(t *testing.T) {
	manifest := NewUnifiedOCIManifest(ArtifactTypeModel, "test-model", []OCILayer{})

	config := AIModelConfig{
		ModelType: "text-generation",
		Runtime:   "vllm",
	}
	manifest.SetModelConfig(config)

	if manifest.Config.MediaType != AIModelConfigMediaType {
		t.Errorf("Config.MediaType = %s, want %s", manifest.Config.MediaType, AIModelConfigMediaType)
	}
	if manifest.Annotations[AnnotationArtifactType] != string(ArtifactTypeModel) {
		t.Errorf("Artifact type annotation = %s, want %s", manifest.Annotations[AnnotationArtifactType], string(ArtifactTypeModel))
	}
}

func TestSetSkillConfig(t *testing.T) {
	manifest := NewUnifiedOCIManifest(ArtifactTypeSkill, "test-skill", []OCILayer{})

	config := AISkillConfig{
		SkillType: "rag",
		Runtime:   "python",
	}
	manifest.SetSkillConfig(config)

	if manifest.Config.MediaType != AISkillConfigMediaType {
		t.Errorf("Config.MediaType = %s, want %s", manifest.Config.MediaType, AISkillConfigMediaType)
	}
	if manifest.Annotations[AnnotationArtifactType] != string(ArtifactTypeSkill) {
		t.Errorf("Artifact type annotation = %s, want %s", manifest.Annotations[AnnotationArtifactType], string(ArtifactTypeSkill))
	}
}

func TestSetPipelineConfig(t *testing.T) {
	manifest := NewUnifiedOCIManifest(ArtifactTypePipeline, "test-pipeline", []OCILayer{})

	config := AIPipelineConfig{
		PipelineType: "inference",
		Components: []PipelineComponent{
			{Name: "model", Type: "model", Reference: "model:v1"},
		},
	}
	manifest.SetPipelineConfig(config)

	if manifest.Config.MediaType != AIPipelineConfigMediaType {
		t.Errorf("Config.MediaType = %s, want %s", manifest.Config.MediaType, AIPipelineConfigMediaType)
	}
	if manifest.Annotations[AnnotationArtifactType] != string(ArtifactTypePipeline) {
		t.Errorf("Artifact type annotation = %s, want %s", manifest.Annotations[AnnotationArtifactType], string(ArtifactTypePipeline))
	}
}

func TestAddLayer(t *testing.T) {
	manifest := NewUnifiedOCIManifest(ArtifactTypeModel, "test-model", []OCILayer{})

	layer := OCILayer{
		MediaType: "application/vnd.oci.image.layer.v1.tar",
		Digest:    "sha256:abc123",
		Size:      1024,
	}
	manifest.AddLayer(layer)

	if len(manifest.Layers) != 1 {
		t.Errorf("len(Layers) = %d, want 1", len(manifest.Layers))
	}
	if manifest.Layers[0].Digest != layer.Digest {
		t.Errorf("Layers[0].Digest = %s, want %s", manifest.Layers[0].Digest, layer.Digest)
	}
}

// TestConfigDescriptorFollowsAIConfig verifies that constructing a manifest
// and every way of changing its AI config leave the config descriptor
// describing the current config, so callers (e.g. `push`) can validate the
// manifest without an extra step.
func TestConfigDescriptorFollowsAIConfig(t *testing.T) {
	m := NewUnifiedOCIManifest(ArtifactTypeModel, "test-model", []OCILayer{})
	if m.Config.Digest == "" || m.Config.Size == 0 {
		t.Fatalf("new manifest has no config descriptor: %+v", m.Config)
	}
	if err := ValidateOCIManifest(m); err != nil {
		t.Errorf("new manifest does not validate: %v", err)
	}
	initial := m.Config.Digest

	m.SetModelConfig(AIModelConfig{ModelType: "embedding"})
	afterSet := m.Config.Digest
	if afterSet == initial {
		t.Error("SetModelConfig did not update the config digest")
	}

	m.SetRelationships(map[string][]string{"skills": {"skill:v1"}})
	if m.Config.Digest == afterSet {
		t.Error("SetRelationships did not update the config digest")
	}

	blob, err := m.configBlob()
	if err != nil {
		t.Fatal(err)
	}
	if want := fmt.Sprintf("sha256:%x", sha256.Sum256(blob)); m.Config.Digest != want || m.Config.Size != int64(len(blob)) {
		t.Errorf("descriptor = %+v, want digest %s size %d", m.Config, want, len(blob))
	}
	if err := ValidateOCIManifest(m); err != nil {
		t.Errorf("manifest does not validate after config changes: %v", err)
	}
}

func TestSetRelationships(t *testing.T) {
	manifest := NewUnifiedOCIManifest(ArtifactTypeModel, "test-model", []OCILayer{})

	relationships := map[string][]string{
		"skills":    {"skill:v1", "skill:v2"},
		"pipelines": {"pipeline:v1"},
	}
	manifest.SetRelationships(relationships)

	// Check that relationships were set on the AIConfig
	if manifest.AIConfig == nil {
		t.Fatal("AIConfig is nil")
	}

	modelConfig, ok := manifest.AIConfig.(AIModelConfig)
	if !ok {
		t.Fatal("AIConfig is not AIModelConfig")
	}

	if len(modelConfig.Relationships) != 2 {
		t.Errorf("len(Relationships) = %d, want 2", len(modelConfig.Relationships))
	}
	if len(modelConfig.Relationships["skills"]) != 2 {
		t.Errorf("len(Relationships[skills]) = %d, want 2", len(modelConfig.Relationships["skills"]))
	}
}

func TestWriteUnifiedOCIManifest(t *testing.T) {
	manifest := NewUnifiedOCIManifest(ArtifactTypeModel, "test-model", []OCILayer{})

	tmpDir := t.TempDir()
	manifestPath := filepath.Join(tmpDir, "manifest.json")

	err := WriteUnifiedOCIManifest(manifest, manifestPath)
	if err != nil {
		t.Fatalf("WriteUnifiedOCIManifest failed: %v", err)
	}

	// Read it back and verify
	readManifest, err := ReadUnifiedOCIManifest(manifestPath)
	if err != nil {
		t.Fatalf("ReadUnifiedOCIManifest failed: %v", err)
	}

	if readManifest.SchemaVersion != manifest.SchemaVersion {
		t.Errorf("SchemaVersion = %d, want %d", readManifest.SchemaVersion, manifest.SchemaVersion)
	}
	if readManifest.MediaType != manifest.MediaType {
		t.Errorf("MediaType = %s, want %s", readManifest.MediaType, manifest.MediaType)
	}
	if readManifest.Annotations[AnnotationArtifactType] != manifest.Annotations[AnnotationArtifactType] {
		t.Errorf("Artifact type annotation = %s, want %s", readManifest.Annotations[AnnotationArtifactType], manifest.Annotations[AnnotationArtifactType])
	}
}

// TestWriteUnifiedOCIManifestConfigBlobMatchesDescriptor verifies that the
// config.json written next to the manifest is the blob the manifest's config
// descriptor describes: same sha256 digest and size, regardless of whether
// the descriptor was refreshed before the write.
func TestWriteUnifiedOCIManifestConfigBlobMatchesDescriptor(t *testing.T) {
	manifest := NewUnifiedOCIManifest(ArtifactTypeModel, "test-model", []OCILayer{})
	manifest.SetModelConfig(AIModelConfig{ModelType: "text-generation", Runtime: "vllm", Capabilities: []string{"chat"}})
	digestBeforeWrite := manifest.Config.Digest

	// Change the config behind the setters' back: Write must still not
	// keep the stale digest.
	manifest.AIConfig = AIModelConfig{ModelType: "embedding"}

	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "manifest.json")
	if err := WriteUnifiedOCIManifest(manifest, manifestPath); err != nil {
		t.Fatalf("WriteUnifiedOCIManifest failed: %v", err)
	}

	blob, err := os.ReadFile(filepath.Join(dir, ConfigBlobFileName))
	if err != nil {
		t.Fatalf("config blob not written: %v", err)
	}
	wantDigest := fmt.Sprintf("sha256:%x", sha256.Sum256(blob))

	written, err := ReadUnifiedOCIManifest(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	for name, m := range map[string]*UnifiedOCIManifest{"in-memory": manifest, "written": written} {
		if m.Config.Digest != wantDigest {
			t.Errorf("%s manifest config digest = %s, want sha256 of config.json %s", name, m.Config.Digest, wantDigest)
		}
		if m.Config.Size != int64(len(blob)) {
			t.Errorf("%s manifest config size = %d, want len(config.json) %d", name, m.Config.Size, len(blob))
		}
	}
	if manifest.Config.Digest == digestBeforeWrite {
		t.Error("Write kept the config digest computed before the config changed")
	}

	var parsed AIModelConfig
	if err := json.Unmarshal(blob, &parsed); err != nil {
		t.Fatalf("config.json is not an AI model config: %v", err)
	}
	if parsed.ModelType != "embedding" {
		t.Errorf("config.json ai.model.type = %q, want embedding", parsed.ModelType)
	}
}

func TestValidateOCIManifest(t *testing.T) {
	tests := []struct {
		name     string
		manifest *UnifiedOCIManifest
		wantErr  bool
	}{
		{
			name: "valid model manifest",
			manifest: func() *UnifiedOCIManifest {
				m := NewUnifiedOCIManifest(ArtifactTypeModel, "test-model", []OCILayer{})
				m.SetModelConfig(AIModelConfig{ModelType: "text-generation"})
				return m
			}(),
			wantErr: false,
		},
		{
			name: "valid skill manifest",
			manifest: func() *UnifiedOCIManifest {
				m := NewUnifiedOCIManifest(ArtifactTypeSkill, "test-skill", []OCILayer{})
				m.SetSkillConfig(AISkillConfig{SkillType: "rag"})
				return m
			}(),
			wantErr: false,
		},
		{
			name: "valid pipeline manifest",
			manifest: func() *UnifiedOCIManifest {
				m := NewUnifiedOCIManifest(ArtifactTypePipeline, "test-pipeline", []OCILayer{})
				m.SetPipelineConfig(AIPipelineConfig{PipelineType: "inference"})
				return m
			}(),
			wantErr: false,
		},
		{
			name: "invalid schema version",
			manifest: func() *UnifiedOCIManifest {
				m := NewUnifiedOCIManifest(ArtifactTypeModel, "test-model", []OCILayer{})
				m.SchemaVersion = 1
				return m
			}(),
			wantErr: true,
		},
		{
			name: "invalid media type",
			manifest: func() *UnifiedOCIManifest {
				m := NewUnifiedOCIManifest(ArtifactTypeModel, "test-model", []OCILayer{})
				m.MediaType = "invalid"
				return m
			}(),
			wantErr: true,
		},
		{
			name: "missing config media type",
			manifest: func() *UnifiedOCIManifest {
				m := NewUnifiedOCIManifest(ArtifactTypeModel, "test-model", []OCILayer{})
				m.Config.MediaType = ""
				return m
			}(),
			wantErr: true,
		},
		{
			name: "missing artifact type annotation",
			manifest: func() *UnifiedOCIManifest {
				m := NewUnifiedOCIManifest(ArtifactTypeModel, "test-model", []OCILayer{})
				delete(m.Annotations, AnnotationArtifactType)
				return m
			}(),
			wantErr: true,
		},
		{
			name: "model with wrong config media type",
			manifest: func() *UnifiedOCIManifest {
				m := NewUnifiedOCIManifest(ArtifactTypeModel, "test-model", []OCILayer{})
				m.Config.MediaType = AISkillConfigMediaType
				return m
			}(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateOCIManifest(tt.manifest)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateOCIManifest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUnifiedOCIManifestJSONStructure(t *testing.T) {
	manifest := NewUnifiedOCIManifest(ArtifactTypeModel, "test-model", []OCILayer{})
	config := AIModelConfig{
		ModelType:    "text-generation",
		Runtime:      "vllm",
		Capabilities: []string{"chat", "completion"},
		Relationships: map[string][]string{
			"skills": {"skill:v1"},
		},
	}
	manifest.SetModelConfig(config)

	// Marshal to JSON
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatalf("json.MarshalIndent failed: %v", err)
	}

	// Verify it can be unmarshaled back
	var parsed UnifiedOCIManifest
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	if parsed.SchemaVersion != manifest.SchemaVersion {
		t.Errorf("SchemaVersion = %d, want %d", parsed.SchemaVersion, manifest.SchemaVersion)
	}
	if parsed.MediaType != manifest.MediaType {
		t.Errorf("MediaType = %s, want %s", parsed.MediaType, manifest.MediaType)
	}
}

func TestUnifiedOCIManifestWithLayers(t *testing.T) {
	layers := []OCILayer{
		{MediaType: "application/vnd.oci.image.layer.v1.tar", Digest: "sha256:abc123", Size: 1024},
		{MediaType: "application/vnd.oci.image.layer.v1.tar", Digest: "sha256:def456", Size: 2048},
	}

	manifest := NewUnifiedOCIManifest(ArtifactTypeModel, "test-model", layers)

	if len(manifest.Layers) != 2 {
		t.Errorf("len(Layers) = %d, want 2", len(manifest.Layers))
	}
	if manifest.Layers[0].Digest != "sha256:abc123" {
		t.Errorf("Layers[0].Digest = %s, want sha256:abc123", manifest.Layers[0].Digest)
	}
	if manifest.Layers[1].Digest != "sha256:def456" {
		t.Errorf("Layers[1].Digest = %s, want sha256:def456", manifest.Layers[1].Digest)
	}
}

func TestReadUnifiedOCIManifestMissingFile(t *testing.T) {
	_, err := ReadUnifiedOCIManifest("/nonexistent/path/manifest.json")
	if err == nil {
		t.Error("expected error for missing file, got nil")
	}
}

func TestReadUnifiedOCIManifestInvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	manifestPath := filepath.Join(tmpDir, "manifest.json")

	// Write invalid JSON
	if err := os.WriteFile(manifestPath, []byte("invalid json"), 0644); err != nil {
		t.Fatalf("failed to write invalid JSON: %v", err)
	}

	_, err := ReadUnifiedOCIManifest(manifestPath)
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}
