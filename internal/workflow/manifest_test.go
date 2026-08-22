package workflow

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewManifest(t *testing.T) {
	annotations := map[string]string{
		AnnotationRuntime:     "vllm",
		AnnotationAccelerator: "nvidia-gpu",
	}

	m := NewManifest(annotations)

	if m.SchemaVersion != 2 {
		t.Errorf("SchemaVersion = %d, want 2", m.SchemaVersion)
	}
	if m.MediaType != ManifestMediaType {
		t.Errorf("MediaType = %q, want %q", m.MediaType, ManifestMediaType)
	}
	if m.Config.MediaType != ConfigMediaType {
		t.Errorf("Config.MediaType = %q, want %q", m.Config.MediaType, ConfigMediaType)
	}
	if len(m.Annotations) != 2 {
		t.Errorf("Annotations = %v, want 2 entries", m.Annotations)
	}
}

// TestWriteReadManifestRoundTrip verifies that annotations written to disk
// via WriteManifest can be read back unchanged via ReadManifest.
func TestWriteReadManifestRoundTrip(t *testing.T) {
	annotations := NewAnnotationSet().ToMap()
	annotations[AnnotationMOFClass] = "I"

	m := NewManifest(annotations)
	path := filepath.Join(t.TempDir(), "manifest.json")

	if err := WriteManifest(m, path); err != nil {
		t.Fatalf("WriteManifest() error = %v", err)
	}

	got, err := ReadManifest(path)
	if err != nil {
		t.Fatalf("ReadManifest() error = %v", err)
	}

	if got.SchemaVersion != m.SchemaVersion {
		t.Errorf("SchemaVersion = %d, want %d", got.SchemaVersion, m.SchemaVersion)
	}
	if got.MediaType != m.MediaType {
		t.Errorf("MediaType = %q, want %q", got.MediaType, m.MediaType)
	}
	if len(got.Annotations) != len(annotations) {
		t.Fatalf("Annotations count = %d, want %d", len(got.Annotations), len(annotations))
	}
	for k, v := range annotations {
		if got.Annotations[k] != v {
			t.Errorf("Annotations[%q] = %q, want %q", k, got.Annotations[k], v)
		}
	}
}

func TestReadManifestMissingFile(t *testing.T) {
	_, err := ReadManifest(filepath.Join(t.TempDir(), "does-not-exist.json"))
	if err == nil {
		t.Error("ReadManifest() on missing file expected error, got nil")
	}
}

func TestReadManifestInvalidJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "manifest.json")
	if err := os.WriteFile(path, []byte("not json"), 0644); err != nil {
		t.Fatalf("failed to write test fixture: %v", err)
	}

	_, err := ReadManifest(path)
	if err == nil {
		t.Error("ReadManifest() on invalid JSON expected error, got nil")
	}
}
