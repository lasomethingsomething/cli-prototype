package workflow

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestNewManifestFromDirectory(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		"model.safetensors":   "weights-bytes",
		"tokenizer/vocab.txt": "vocab",
		"manifest.json":       "{}", // written by model-cli: must be skipped
		"config.json":         "{}", // written by model-cli: must be skipped
		"attestation.json":    "{}", // written by model-cli: must be skipped
		"sbom.spdx-json":      "{}", // written by model-cli: must be skipped
	}
	for name, content := range files {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	m, err := NewManifestFromDirectory(ArtifactTypeModel, "my-model:v1", dir, map[string]string{
		AnnotationRuntime: "vllm",
	})
	if err != nil {
		t.Fatalf("NewManifestFromDirectory() error = %v", err)
	}

	if m.SchemaVersion != 2 || m.MediaType != OCIManifestMediaType {
		t.Errorf("manifest header = %d/%s, want 2/%s", m.SchemaVersion, m.MediaType, OCIManifestMediaType)
	}
	if m.ArtifactType != "application/vnd.cncf.ai.model" {
		t.Errorf("ArtifactType = %q, want application/vnd.cncf.ai.model", m.ArtifactType)
	}
	if m.Config.MediaType != AIModelConfigMediaType || m.Config.Digest == "" || m.Config.Size == 0 {
		t.Errorf("config descriptor = %+v, want media type, digest and size set", m.Config)
	}
	if m.Annotations[AnnotationRuntime] != "vllm" || m.Annotations[AnnotationArtifactType] != "model" {
		t.Errorf("annotations = %v, want runtime and artifact type set", m.Annotations)
	}
	if m.Annotations["org.opencontainers.image.title"] != "my-model:v1" {
		t.Errorf("title annotation = %q, want my-model:v1", m.Annotations["org.opencontainers.image.title"])
	}

	// Layers: sorted by title, generated files skipped, real digests and sizes.
	if len(m.Layers) != 2 {
		t.Fatalf("got %d layers, want 2 (generated files skipped): %+v", len(m.Layers), m.Layers)
	}
	wantTitles := []string{"model.safetensors", "tokenizer/vocab.txt"}
	for i, want := range wantTitles {
		layer := m.Layers[i]
		if got := layer.Annotations["org.opencontainers.image.title"]; got != want {
			t.Errorf("layer %d title = %q, want %q", i, got, want)
		}
		content := files[want]
		wantDigest := fmt.Sprintf("sha256:%x", sha256.Sum256([]byte(content)))
		if layer.Digest != wantDigest {
			t.Errorf("layer %d digest = %q, want %q", i, layer.Digest, wantDigest)
		}
		if layer.Size != int64(len(content)) {
			t.Errorf("layer %d size = %d, want %d", i, layer.Size, len(content))
		}
		if layer.MediaType != OCILayerMediaType {
			t.Errorf("layer %d media type = %q, want %q", i, layer.MediaType, OCILayerMediaType)
		}
	}

	if err := ValidateOCIManifest(m); err != nil {
		t.Errorf("ValidateOCIManifest() error = %v", err)
	}
}

// TestLayersFromDirectorySkipsGeneratedFiles pins the set of files model-cli
// writes next to the model: none of them may become a layer, so packaging a
// directory that was packaged before yields the same layers.
func TestLayersFromDirectorySkipsGeneratedFiles(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"weights.bin", "manifest.json", "config.json", "attestation.json", "sbom.spdx.json", "sbom.cyclonedx.json"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(name), 0644); err != nil {
			t.Fatal(err)
		}
	}
	layers, err := layersFromDirectory(dir)
	if err != nil {
		t.Fatalf("layersFromDirectory() error = %v", err)
	}
	if len(layers) != 1 || layers[0].Annotations["org.opencontainers.image.title"] != "weights.bin" {
		t.Errorf("layers = %+v, want only weights.bin", layers)
	}
}

func TestNewManifestFromSingleFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "model.gguf")
	if err := os.WriteFile(path, []byte("weights"), 0644); err != nil {
		t.Fatal(err)
	}
	m, err := NewManifestFromDirectory(ArtifactTypeSkill, "my-skill:v1", path, nil)
	if err != nil {
		t.Fatalf("NewManifestFromDirectory() error = %v", err)
	}
	if len(m.Layers) != 1 || m.Layers[0].Annotations["org.opencontainers.image.title"] != "model.gguf" {
		t.Errorf("layers = %+v, want a single layer titled model.gguf", m.Layers)
	}
	if m.Annotations[AnnotationArtifactType] != "skill" || m.Config.MediaType != AISkillConfigMediaType {
		t.Errorf("skill manifest = type %q / config %q", m.Annotations[AnnotationArtifactType], m.Config.MediaType)
	}
}

func TestNewManifestFromMissingPath(t *testing.T) {
	if _, err := NewManifestFromDirectory(ArtifactTypeModel, "x", filepath.Join(t.TempDir(), "nope"), nil); err == nil {
		t.Error("NewManifestFromDirectory() on a missing path should fail")
	}
}

// TestWriteReadManifestRoundTrip verifies that a manifest written to disk can
// be read back unchanged, and that writing fills the config descriptor.
func TestWriteReadManifestRoundTrip(t *testing.T) {
	annotations := NewAnnotationSet().ToMap()
	annotations[AnnotationMOFClass] = "I"

	m := NewUnifiedOCIManifest(ArtifactTypeModel, "my-model:v1", nil)
	for k, v := range annotations {
		m.Annotations[k] = v
	}
	path := filepath.Join(t.TempDir(), "manifest.json")

	if err := WriteUnifiedOCIManifest(m, path); err != nil {
		t.Fatalf("WriteUnifiedOCIManifest() error = %v", err)
	}
	if m.Config.Digest == "" || m.Config.Size == 0 {
		t.Errorf("config descriptor not filled on write: %+v", m.Config)
	}

	got, err := ReadUnifiedOCIManifest(path)
	if err != nil {
		t.Fatalf("ReadUnifiedOCIManifest() error = %v", err)
	}
	if got.SchemaVersion != m.SchemaVersion || got.MediaType != m.MediaType || got.ArtifactType != m.ArtifactType {
		t.Errorf("header mismatch: got %d/%s/%s", got.SchemaVersion, got.MediaType, got.ArtifactType)
	}
	if got.Config.MediaType != m.Config.MediaType || got.Config.Digest != m.Config.Digest || got.Config.Size != m.Config.Size {
		t.Errorf("config descriptor = %+v, want %+v", got.Config, m.Config)
	}
	for k, v := range m.Annotations {
		if got.Annotations[k] != v {
			t.Errorf("Annotations[%q] = %q, want %q", k, got.Annotations[k], v)
		}
	}
}

func TestReadManifestMissingFile(t *testing.T) {
	if _, err := ReadUnifiedOCIManifest(filepath.Join(t.TempDir(), "does-not-exist.json")); err == nil {
		t.Error("ReadUnifiedOCIManifest() on missing file expected error, got nil")
	}
}

func TestReadManifestInvalidJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "manifest.json")
	if err := os.WriteFile(path, []byte("not json"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadUnifiedOCIManifest(path); err == nil {
		t.Error("ReadUnifiedOCIManifest() on invalid JSON expected error, got nil")
	}
}
