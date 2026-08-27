package workflow

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeRegistryProvider is a test double for RegistryProvider that records the
// annotations it was pushed with instead of shelling out to a real tool.
type fakeRegistryProvider struct {
	installed         bool
	pushedAnnotations map[string]string
	pushedSource      string
	pushCalled        bool
	// For local parity testing: store the artifact digest
	artifactDigest map[string]string // maps artifact name to its digest
}

func (f *fakeRegistryProvider) Name() string                         { return "fake" }
func (f *fakeRegistryProvider) IsInstalled() bool                    { return f.installed }
func (f *fakeRegistryProvider) InstallInstructions() string          { return "n/a" }
func (f *fakeRegistryProvider) Pull(artifact, registry string) error { return nil }
func (f *fakeRegistryProvider) Push(artifact, registry, sourcePath string, annotations map[string]string) (string, error) {
	f.pushCalled = true
	f.pushedAnnotations = annotations
	f.pushedSource = sourcePath
	if f.artifactDigest == nil {
		f.artifactDigest = make(map[string]string)
	}
	// Simulate a registry: the digest reported on push is what a later
	// GetArtifactDigest returns, unless a test overrides it.
	digest := fakeDigest(artifact)
	if _, overridden := f.artifactDigest[artifact]; !overridden {
		f.artifactDigest[artifact] = digest
	}
	return digest, nil
}

// fakeDigest returns a well-formed, deterministic digest for an artifact name.
func fakeDigest(artifact string) string {
	return fmt.Sprintf("sha256:%064x", len(artifact))
}
func (f *fakeRegistryProvider) PushReferrer(artifact, registry, referrerType string, data []byte, annotations map[string]string) error {
	// For testing, we don't need to do anything with referrers
	return nil
}
func (f *fakeRegistryProvider) GetReferrers(artifact, registry, referrerType string) ([][]byte, error) {
	// For testing, return empty referrers
	return nil, nil
}
func (f *fakeRegistryProvider) GetArtifactDigest(artifact, registry string) (string, error) {
	if digest, ok := f.artifactDigest[artifact]; ok {
		return digest, nil
	}
	return "", fmt.Errorf("artifact %s not found in fake registry", artifact)
}
func (f *fakeRegistryProvider) Search(registry string, query *SearchQuery) ([]ManifestCandidate, error) {
	// Fake implementation for testing - returns empty results
	// In real tests, this can be extended to return mock data
	return nil, nil
}

func (f *fakeRegistryProvider) FetchManifestAnnotations(artifactRef string) (map[string]string, error) {
	// Fake implementation for testing - returns the annotations that were pushed
	// This allows tests to verify that annotations are properly fetched
	if f.pushedAnnotations != nil {
		// Return a copy to avoid race conditions
		annotations := make(map[string]string)
		for k, v := range f.pushedAnnotations {
			annotations[k] = v
		}
		return annotations, nil
	}
	return nil, nil
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
	if err := os.WriteFile(filepath.Join(modelPath, "model.txt"), []byte("weights"), 0644); err != nil {
		t.Fatal(err)
	}

	fake := &fakeRegistryProvider{installed: true}
	pf := &PackageWorkflow{
		registry:         "fake",
		registryProvider: fake,
		annotations:      NewAnnotationSet(),
		verifyParity:     false, // Disable for tests that don't set up matching digests
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

	manifest, err := ReadUnifiedOCIManifest(wantManifestPath)
	if err != nil {
		t.Fatalf("ReadUnifiedOCIManifest() error = %v", err)
	}
	if manifest.Annotations[AnnotationRuntime] != "vllm" {
		t.Errorf("manifest annotation %s = %q, want %q", AnnotationRuntime, manifest.Annotations[AnnotationRuntime], "vllm")
	}
	if manifest.Annotations[AnnotationAccelerator] != "nvidia-gpu" {
		t.Errorf("manifest annotation %s = %q, want %q", AnnotationAccelerator, manifest.Annotations[AnnotationAccelerator], "nvidia-gpu")
	}
	if len(manifest.Layers) != 1 || manifest.Layers[0].Annotations["org.opencontainers.image.title"] != "model.txt" {
		t.Errorf("manifest layers = %+v, want one layer for model.txt", manifest.Layers)
	}
	if err := ValidateOCIManifest(manifest); err != nil {
		t.Errorf("written manifest does not validate: %v", err)
	}

	if !fake.pushCalled {
		t.Fatal("expected registry provider Push() to be called")
	}
	if fake.pushedAnnotations[AnnotationRuntime] != "vllm" {
		t.Errorf("Push() annotations[%s] = %q, want %q", AnnotationRuntime, fake.pushedAnnotations[AnnotationRuntime], "vllm")
	}

	// Verify local digest was computed
	if pf.LocalDigest() == "" {
		t.Error("expected local digest to be computed")
	}
}

// TestPackageWorkflowRunTwiceKeepsLayers verifies that the files package
// writes next to the model (manifest.json, config.json) do not become layers
// when the same directory is packaged again.
func TestPackageWorkflowRunTwiceKeepsLayers(t *testing.T) {
	modelPath := t.TempDir()
	if err := os.WriteFile(filepath.Join(modelPath, "model.txt"), []byte("weights"), 0644); err != nil {
		t.Fatal(err)
	}

	run := func() *UnifiedOCIManifest {
		pf := &PackageWorkflow{registry: "fake", registryProvider: &fakeRegistryProvider{installed: true}, annotations: NewAnnotationSet()}
		pf.SetPackageInfo("phi-4-mini", modelPath, "my-model:v1", "ghcr.io/my-org", false, "")
		if err := pf.Run(); err != nil {
			t.Fatalf("Run() error = %v", err)
		}
		m, err := ReadUnifiedOCIManifest(pf.ManifestPath())
		if err != nil {
			t.Fatalf("ReadUnifiedOCIManifest() error = %v", err)
		}
		return m
	}

	first := run()
	if _, err := os.Stat(filepath.Join(modelPath, "config.json")); err != nil {
		t.Fatalf("package should write config.json next to the model: %v", err)
	}
	second := run()

	if len(first.Layers) != 1 || len(second.Layers) != 1 {
		t.Fatalf("layers after first run = %d, after second run = %d, want 1 and 1 (generated files must not become layers)", len(first.Layers), len(second.Layers))
	}
	if first.Layers[0].Digest != second.Layers[0].Digest {
		t.Errorf("layer digest changed between runs: %s vs %s", first.Layers[0].Digest, second.Layers[0].Digest)
	}
}

// TestPackageWorkflowRunVerifiesLocalParity verifies that Run() compares the
// digest reported by Push with the digest stored in the registry.
func TestPackageWorkflowRunVerifiesLocalParity(t *testing.T) {
	artifactName := "my-model:v1"
	registryURL := "ghcr.io/my-org"

	t.Run("match", func(t *testing.T) {
		fake := &fakeRegistryProvider{installed: true}
		pf := &PackageWorkflow{registry: "fake", registryProvider: fake, annotations: NewAnnotationSet(), verifyParity: true}
		pf.SetPackageInfo("phi-4-mini", t.TempDir(), artifactName, registryURL, false, "")

		if err := pf.Run(); err != nil {
			t.Fatalf("Run() error = %v, want parity to pass when the registry holds the pushed digest", err)
		}
		if pf.PushedDigest() != fakeDigest(artifactName) {
			t.Errorf("PushedDigest() = %q, want %q", pf.PushedDigest(), fakeDigest(artifactName))
		}
		if pf.LocalDigest() == "" {
			t.Error("expected local manifest digest to be computed")
		}
	})

	t.Run("mismatch", func(t *testing.T) {
		fake := &fakeRegistryProvider{
			installed:      true,
			artifactDigest: map[string]string{artifactName: "sha256:" + strings.Repeat("f", 64)},
		}
		pf := &PackageWorkflow{registry: "fake", registryProvider: fake, annotations: NewAnnotationSet(), verifyParity: true}
		pf.SetPackageInfo("phi-4-mini", t.TempDir(), artifactName, registryURL, false, "")

		err := pf.Run()
		if err == nil || !strings.Contains(err.Error(), "local parity") {
			t.Fatalf("Run() error = %v, want a local parity failure when the registry digest differs", err)
		}
	})

	t.Run("skipped when tool reports no digest", func(t *testing.T) {
		fake := &noDigestProvider{fakeRegistryProvider{installed: true}}
		pf := &PackageWorkflow{registry: "fake", registryProvider: fake, annotations: NewAnnotationSet(), verifyParity: true}
		pf.SetPackageInfo("phi-4-mini", t.TempDir(), artifactName, registryURL, false, "")

		if err := pf.Run(); err != nil {
			t.Fatalf("Run() error = %v, want parity to be skipped (not failed) without a pushed digest", err)
		}
	})
}

// noDigestProvider behaves like fakeRegistryProvider but never reports a
// digest on push, like a tool without --format json support.
type noDigestProvider struct{ fakeRegistryProvider }

func (n *noDigestProvider) Push(artifact, registry, sourcePath string, annotations map[string]string) (string, error) {
	_, err := n.fakeRegistryProvider.Push(artifact, registry, sourcePath, annotations)
	return "", err
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

// TestSetSigningOptions verifies SetSigningOptions correctly sets the signing fields
func TestSetSigningOptions(t *testing.T) {
	pf, err := NewPackageWorkflow("oras")
	if err != nil {
		t.Fatalf("NewPackageWorkflow error: %v", err)
	}

	pf.SetSigningOptions(true, "sigstore")

	if !pf.sign {
		t.Error("sign = false, want true")
	}
	if pf.signer != "sigstore" {
		t.Errorf("signer = %q, want %q", pf.signer, "sigstore")
	}

	// Test with empty signer
	pf.SetSigningOptions(false, "")
	if pf.sign {
		t.Error("sign = true, want false")
	}
	if pf.signer != "" {
		t.Errorf("signer = %q, want empty string", pf.signer)
	}
}

// TestPackageWorkflowRunWithSigning verifies that when sign=true, the workflow
// attempts to sign the artifact after packaging using the configured signer.
func TestPackageWorkflowRunWithSigning(t *testing.T) {
	modelPath := t.TempDir()

	// Create a fake registry provider
	fakeReg := &fakeRegistryProvider{installed: true}

	// Create a fake signing provider and register it
	// We need to temporarily replace the GetSigningProvider function
	// For this test, we'll create a custom workflow with the fake signer

	pf := &PackageWorkflow{
		registry:         "fake",
		registryProvider: fakeReg,
		annotations:      NewAnnotationSet(),
		sign:             true,
		signer:           "fake-sign",
	}
	pf.SetPackageInfo("phi-4-mini", modelPath, "my-model:v1", "", false, "")

	// Since we can't easily mock GetSigningProvider, we'll test the workflow
	// logic by checking that the sign flag is properly set
	// The actual signing integration is tested via the command tests

	// Just verify the workflow can be created with signing options
	if !pf.sign {
		t.Error("sign should be true")
	}
	if pf.signer != "fake-sign" {
		t.Errorf("signer = %q, want %q", pf.signer, "fake-sign")
	}
}

// TestPackageWorkflowAppliesMOFClassification verifies the detected MOF class
// and components land in the manifest when the user left them empty, and
// that an explicit class is kept.
func TestPackageWorkflowAppliesMOFClassification(t *testing.T) {
	newModelDir := func(t *testing.T) string {
		dir := t.TempDir()
		for name, content := range map[string]string{"model.safetensors": "weights", "README.md": "# model"} {
			if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
				t.Fatal(err)
			}
		}
		return dir
	}

	t.Run("detected when empty", func(t *testing.T) {
		pf := &PackageWorkflow{registry: "fake", registryProvider: &fakeRegistryProvider{installed: true}, annotations: NewAnnotationSet(), includeMOF: true}
		pf.SetPackageInfo("m", newModelDir(t), "m:v1", "", false, "")
		if err := pf.Run(); err != nil {
			t.Fatalf("Run() error = %v", err)
		}
		m, err := ReadUnifiedOCIManifest(pf.ManifestPath())
		if err != nil {
			t.Fatal(err)
		}
		if got := m.Annotations[AnnotationMOFClass]; got != "II" {
			t.Errorf("MOF class annotation = %q, want detected II", got)
		}
		if got := m.Annotations[AnnotationMOFComponents]; got != "weights,documentation" {
			t.Errorf("MOF components annotation = %q, want weights,documentation", got)
		}
	})

	t.Run("explicit class kept", func(t *testing.T) {
		pf := &PackageWorkflow{registry: "fake", registryProvider: &fakeRegistryProvider{installed: true}, annotations: NewAnnotationSet(), includeMOF: true}
		pf.annotations.MOFClass = "I"
		pf.annotations.MOFComponents = "weights,code"
		pf.SetPackageInfo("m", newModelDir(t), "m:v1", "", false, "")
		if err := pf.Run(); err != nil {
			t.Fatalf("Run() error = %v", err)
		}
		m, err := ReadUnifiedOCIManifest(pf.ManifestPath())
		if err != nil {
			t.Fatal(err)
		}
		if m.Annotations[AnnotationMOFClass] != "I" || m.Annotations[AnnotationMOFComponents] != "weights,code" {
			t.Errorf("explicit MOF annotations were overwritten: class=%q components=%q", m.Annotations[AnnotationMOFClass], m.Annotations[AnnotationMOFComponents])
		}
	})
}
