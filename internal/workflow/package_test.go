package workflow

import (
	"os"
	"path/filepath"
	"testing"
)

// fakeRegistryProvider is a test double for RegistryProvider that records the
// annotations it was pushed with instead of shelling out to a real tool.
type fakeRegistryProvider struct {
	installed         bool
	pushedAnnotations map[string]string
	pushCalled        bool
}

func (f *fakeRegistryProvider) Name() string                         { return "fake" }
func (f *fakeRegistryProvider) IsInstalled() bool                    { return f.installed }
func (f *fakeRegistryProvider) InstallInstructions() string          { return "n/a" }
func (f *fakeRegistryProvider) Pull(artifact, registry string) error { return nil }
func (f *fakeRegistryProvider) Push(artifact, registry string, annotations map[string]string) error {
	f.pushCalled = true
	f.pushedAnnotations = annotations
	return nil
}

// Test NewPackageWorkflow
func TestNewPackageWorkflow(t *testing.T) {
	// Test with valid registry
	pf, err := NewPackageWorkflow("oras")
	if err != nil {
		t.Fatalf("NewPackageWorkflow(\"oras\") error = %v", err)
	}
	if pf == nil {
		t.Fatal("NewPackageWorkflow(\"oras\") returned nil")
	}
	if pf.registry != "oras" {
		t.Errorf("registry = %q, want %q", pf.registry, "oras")
	}

	// Test with invalid registry
	_, err = NewPackageWorkflow("invalid-registry")
	if err == nil {
		t.Error("NewPackageWorkflow(\"invalid-registry\") expected error, got nil")
	}
}

// Test SetPackageInfo
func TestSetPackageInfo(t *testing.T) {
	pf, err := NewPackageWorkflow("oras")
	if err != nil {
		t.Fatalf("NewPackageWorkflow error: %v", err)
	}

	pf.SetPackageInfo("phi-4-mini", "/path/to/model", "my-model:v1", "ghcr.io/my-org", true, "/path/to/rag")

	if pf.modelName != "phi-4-mini" {
		t.Errorf("modelName = %q, want %q", pf.modelName, "phi-4-mini")
	}
	if pf.modelPath != "/path/to/model" {
		t.Errorf("modelPath = %q, want %q", pf.modelPath, "/path/to/model")
	}
	if pf.artifactName != "my-model:v1" {
		t.Errorf("artifactName = %q, want %q", pf.artifactName, "my-model:v1")
	}
	if pf.registryURL != "ghcr.io/my-org" {
		t.Errorf("registryURL = %q, want %q", pf.registryURL, "ghcr.io/my-org")
	}
	if !pf.includeRAG {
		t.Error("includeRAG = false, want true")
	}
	if pf.ragPath != "/path/to/rag" {
		t.Errorf("ragPath = %q, want %q", pf.ragPath, "/path/to/rag")
	}
}

// Test SetAnnotations
func TestSetAnnotations(t *testing.T) {
	pf, err := NewPackageWorkflow("oras")
	if err != nil {
		t.Fatalf("NewPackageWorkflow error: %v", err)
	}

	annotations := NewAnnotationSet()
	annotations.Runtime = "vllm"
	annotations.Accelerator = "nvidia-gpu"

	pf.SetAnnotations(annotations)

	if pf.annotations == nil {
		t.Fatal("annotations is nil after SetAnnotations")
	}
	if pf.annotations.Runtime != "vllm" {
		t.Errorf("annotations.Runtime = %q, want %q", pf.annotations.Runtime, "vllm")
	}
	if pf.annotations.Accelerator != "nvidia-gpu" {
		t.Errorf("annotations.Accelerator = %q, want %q", pf.annotations.Accelerator, "nvidia-gpu")
	}
}

// Test PackageWorkflow default values
func TestPackageWorkflowDefaults(t *testing.T) {
	pf, err := NewPackageWorkflow("oras")
	if err != nil {
		t.Fatalf("NewPackageWorkflow error: %v", err)
	}

	// Check default values
	if !pf.generateSBOM {
		t.Error("generateSBOM default = false, want true")
	}
	if !pf.includeMOF {
		t.Error("includeMOF default = false, want true")
	}
	if pf.annotations == nil {
		t.Error("annotations default is nil, want non-nil")
	}
}

// TestPackageWorkflowRunWritesManifestWithAnnotations verifies that Run()
// writes a real OCI manifest.json to disk containing the CNCF AI annotations,
// and forwards that same annotation map to the registry provider's Push.
func TestPackageWorkflowRunWritesManifestWithAnnotations(t *testing.T) {
	modelPath := t.TempDir()

	fake := &fakeRegistryProvider{installed: true}
	pf := &PackageWorkflow{
		registry:         "fake",
		registryProvider: fake,
		annotations:      NewAnnotationSet(),
	}
	pf.annotations.Runtime = "vllm"
	pf.annotations.Accelerator = "nvidia-gpu"

	pf.SetPackageInfo("phi-4-mini", modelPath, "my-model:v1", "ghcr.io/my-org", false, "")

	if err := pf.Run(); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	wantManifestPath := filepath.Join(modelPath, "manifest.json")
	if pf.ManifestPath() != wantManifestPath {
		t.Errorf("ManifestPath() = %q, want %q", pf.ManifestPath(), wantManifestPath)
	}

	if _, err := os.Stat(wantManifestPath); err != nil {
		t.Fatalf("expected manifest file at %s: %v", wantManifestPath, err)
	}

	manifest, err := ReadManifest(wantManifestPath)
	if err != nil {
		t.Fatalf("ReadManifest() error = %v", err)
	}
	if manifest.Annotations[AnnotationRuntime] != "vllm" {
		t.Errorf("manifest annotation %s = %q, want %q", AnnotationRuntime, manifest.Annotations[AnnotationRuntime], "vllm")
	}
	if manifest.Annotations[AnnotationAccelerator] != "nvidia-gpu" {
		t.Errorf("manifest annotation %s = %q, want %q", AnnotationAccelerator, manifest.Annotations[AnnotationAccelerator], "nvidia-gpu")
	}

	if !fake.pushCalled {
		t.Fatal("expected registry provider Push() to be called")
	}
	if fake.pushedAnnotations[AnnotationRuntime] != "vllm" {
		t.Errorf("Push() annotations[%s] = %q, want %q", AnnotationRuntime, fake.pushedAnnotations[AnnotationRuntime], "vllm")
	}
}

// TestPackageWorkflowRunNotInstalled verifies Run() fails fast (without
// writing a manifest) when the registry provider isn't installed.
func TestPackageWorkflowRunNotInstalled(t *testing.T) {
	modelPath := t.TempDir()

	fake := &fakeRegistryProvider{installed: false}
	pf := &PackageWorkflow{
		registry:         "fake",
		registryProvider: fake,
		annotations:      NewAnnotationSet(),
	}
	pf.SetPackageInfo("phi-4-mini", modelPath, "my-model:v1", "", false, "")

	if err := pf.Run(); err == nil {
		t.Fatal("Run() expected error when registry provider is not installed, got nil")
	}

	if _, err := os.Stat(filepath.Join(modelPath, "manifest.json")); !os.IsNotExist(err) {
		t.Error("expected no manifest.json to be written when provider is not installed")
	}
}
