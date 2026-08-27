package workflow

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
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
	// Referrers attached with PushReferrer, in order; referrerErr makes it fail.
	referrers   []fakeReferrer
	referrerErr error
}

// fakeReferrer records one PushReferrer call.
type fakeReferrer struct {
	artifact, registry, referrerType string
	data                             []byte
	annotations                      map[string]string
}

func (f *fakeRegistryProvider) Name() string                         { return "fake" }
func (f *fakeRegistryProvider) IsInstalled() bool                    { return f.installed }
func (f *fakeRegistryProvider) InstallInstructions() string          { return "n/a" }
func (f *fakeRegistryProvider) PackagingFormat() string              { return "fake-format" }
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
	if f.referrerErr != nil {
		return f.referrerErr
	}
	f.referrers = append(f.referrers, fakeReferrer{artifact, registry, referrerType, data, annotations})
	return nil
}
func (f *fakeRegistryProvider) GetReferrers(artifact, registry, referrerType string) ([][]byte, error) {
	var out [][]byte
	for _, r := range f.referrers {
		if r.artifact == artifact && r.registry == registry && r.referrerType == referrerType {
			out = append(out, r.data)
		}
	}
	return out, nil
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
	if !pf.verifyParity {
		t.Error("verifyParity default = false, want true")
	}
	if pf.annotations == nil {
		t.Error("annotations default is nil, want non-nil")
	}
}

// TestPackageWorkflowLeavesSBOMAndMOFToHarden verifies packaging is only
// packaging: it neither generates an SBOM nor classifies the model, both of
// which are the separate hardening step (see HardenWorkflow).
func TestPackageWorkflowLeavesSBOMAndMOFToHarden(t *testing.T) {
	modelPath := t.TempDir()
	for name, content := range map[string]string{"model.safetensors": "weights", "README.md": "# model"} {
		if err := os.WriteFile(filepath.Join(modelPath, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	pf := &PackageWorkflow{registry: "fake", registryProvider: &fakeRegistryProvider{installed: true}, annotations: NewAnnotationSet()}
	pf.SetPackageInfo("m", modelPath, "m:v1", "", false, "")
	if err := pf.Run(); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if sboms, _ := filepath.Glob(filepath.Join(modelPath, "sbom.*")); len(sboms) != 0 {
		t.Errorf("package wrote SBOM files %v; SBOM generation belongs to harden", sboms)
	}
	if _, err := os.Stat(filepath.Join(modelPath, "mof.json")); !os.IsNotExist(err) {
		t.Error("package wrote mof.json; MOF classification belongs to harden")
	}
	m, err := ReadUnifiedOCIManifest(pf.ManifestPath())
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{AnnotationMOFClass, AnnotationMOFComponents} {
		if got, ok := m.Annotations[key]; ok {
			t.Errorf("manifest annotation %s = %q; MOF classification belongs to harden", key, got)
		}
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

	// The packaging format records what the registry tool produced, not a
	// default (issue #102).
	for name, got := range map[string]string{
		"manifest": manifest.Annotations[AnnotationPackagingFormat],
		"Push()":   fake.pushedAnnotations[AnnotationPackagingFormat],
	} {
		if got != "fake-format" {
			t.Errorf("%s annotation %s = %q, want the provider's %q", name, AnnotationPackagingFormat, got, "fake-format")
		}
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

// fakeSyft puts a shell script named "syft" first on PATH for the duration of
// the test. It answers `syft version` successfully, so the real SyftGenerator
// reaches Generate(); how the scan itself behaves is decided by scanScript,
// which sees the arguments syft was called with.
func fakeSyft(t *testing.T, scanScript string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("fake syft is a shell script")
	}
	dir := t.TempDir()
	script := "#!/bin/sh\nif [ \"$1\" = version ]; then echo v0.0.0-fake; exit 0; fi\n" + scanScript + "\n"
	if err := os.WriteFile(filepath.Join(dir, "syft"), []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

// TestPackageWorkflowDoesNotGenerateProvenance verifies that packaging stops
// at the artifact: no provenance attestation is written and nothing is
// attached to the registry. Provenance and signing are the separate
// `model-cli sign` step (Phase 1, Step 3; issue #88).
func TestPackageWorkflowDoesNotGenerateProvenance(t *testing.T) {
	modelPath := t.TempDir()
	if err := os.WriteFile(filepath.Join(modelPath, "model.txt"), []byte("weights"), 0644); err != nil {
		t.Fatal(err)
	}

	fake := &fakeRegistryProvider{installed: true}
	pf := &PackageWorkflow{
		registry:         "fake",
		registryProvider: fake,
		annotations:      NewAnnotationSet(),
		verifyParity:     false,
	}
	pf.SetPackageInfo("test-model", modelPath, "test:v1", "ghcr.io/my-org", false, "")

	if err := pf.Run(); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Packaging itself succeeded...
	if _, err := os.Stat(pf.ManifestPath()); err != nil {
		t.Errorf("manifest not written: %v", err)
	}

	// ...but no attestation was written where the old package step put it
	// (nor where the sign step puts it), and none was attached in the registry.
	for _, path := range []string{
		filepath.Join(modelPath, "attestation.json"),
		GetAttestationPath("test:v1"),
		GetAttestationPath("ghcr.io/my-org/test:v1"),
	} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Errorf("provenance attestation %s exists after packaging (stat error = %v); it belongs to the sign step", path, err)
		}
	}
	if len(fake.referrers) != 0 {
		t.Errorf("packaging attached %d referrer(s) to the artifact, want none", len(fake.referrers))
	}
}
