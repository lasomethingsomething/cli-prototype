package workflow

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewLocalParityVerifier(t *testing.T) {
	provider := &fakeRegistryProvider{installed: true}
	verifier := NewLocalParityVerifier(provider, "sha256:abc123")

	if verifier == nil {
		t.Fatal("NewLocalParityVerifier returned nil")
	}
	if verifier.localDigest != "sha256:abc123" {
		t.Errorf("Expected localDigest 'sha256:abc123', got '%s'", verifier.localDigest)
	}
}

func TestLocalParityVerifierVerifyMatch(t *testing.T) {
	// Create a fake registry provider
	provider := &fakeRegistryProvider{
		installed:      true,
		artifactDigest: make(map[string]string),
	}

	// Set up a known digest for an artifact
	artifactName := "test-model:v1"
	registry := "ghcr.io/my-org"
	localDigest := "sha256:abc123"
	provider.artifactDigest[artifactName] = localDigest

	// Create verifier and verify
	verifier := NewLocalParityVerifier(provider, localDigest)
	result, err := verifier.Verify(artifactName, registry)

	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if result == nil {
		t.Fatal("Verify returned nil result")
	}
	if !result.Match {
		t.Errorf("Expected Match to be true, got false. Local: %s, Registry: %s",
			result.LocalDigest, result.RegistryDigest)
	}
	if result.LocalDigest != localDigest {
		t.Errorf("LocalDigest mismatch: expected %s, got %s", localDigest, result.LocalDigest)
	}
	if result.RegistryDigest != localDigest {
		t.Errorf("RegistryDigest mismatch: expected %s, got %s", localDigest, result.RegistryDigest)
	}
}

func TestLocalParityVerifierVerifyMismatch(t *testing.T) {
	// Create a fake registry provider
	provider := &fakeRegistryProvider{
		installed:      true,
		artifactDigest: make(map[string]string),
	}

	// Set up a different digest for the artifact in the registry
	artifactName := "test-model:v1"
	registry := "ghcr.io/my-org"
	localDigest := "sha256:abc123"
	registryDigest := "sha256:def456" // Different from local
	provider.artifactDigest[artifactName] = registryDigest

	// Create verifier and verify
	verifier := NewLocalParityVerifier(provider, localDigest)
	result, err := verifier.Verify(artifactName, registry)

	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if result == nil {
		t.Fatal("Verify returned nil result")
	}
	if result.Match {
		t.Error("Expected Match to be false, got true")
	}
	if result.LocalDigest != localDigest {
		t.Errorf("LocalDigest mismatch: expected %s, got %s", localDigest, result.LocalDigest)
	}
	if result.RegistryDigest != registryDigest {
		t.Errorf("RegistryDigest mismatch: expected %s, got %s", registryDigest, result.RegistryDigest)
	}

	// Test String() output for mismatch
	str := result.String()
	if str == "" {
		t.Error("String() returned empty string")
	}
}

func TestVerifyWithDigest(t *testing.T) {
	// Create a fake registry provider
	provider := &fakeRegistryProvider{
		installed:      true,
		artifactDigest: make(map[string]string),
	}

	artifactName := "my-model:latest"
	registry := "ghcr.io/test"
	localDigest := "sha256:12345678"
	provider.artifactDigest[artifactName] = localDigest

	// Use the convenience function
	result, err := VerifyWithDigest(provider, localDigest, artifactName, registry)

	if err != nil {
		t.Fatalf("VerifyWithDigest failed: %v", err)
	}
	if result == nil {
		t.Fatal("VerifyWithDigest returned nil result")
	}
	if !result.Match {
		t.Errorf("Expected Match to be true, got false")
	}
}

func TestComputeManifestDigest(t *testing.T) {
	// Create a temporary manifest file
	tmpDir, err := os.MkdirTemp("", "parity_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a manifest file with known content
	manifestPath := filepath.Join(tmpDir, "manifest.json")
	manifestContent := `{"schemaVersion":2,"mediaType":"application/vnd.oci.image.manifest.v1+json"}`
	if err := os.WriteFile(manifestPath, []byte(manifestContent), 0644); err != nil {
		t.Fatalf("Failed to write manifest: %v", err)
	}

	// Compute digest
	digest := ComputeManifestDigest(manifestPath)

	// Verify it's a valid SHA256 digest
	if digest == "" {
		t.Error("ComputeManifestDigest returned empty string")
	}
	if len(digest) < 10 { // sha256: + at least a few hex chars
		t.Errorf("Digest too short: %s", digest)
	}
}

func TestLocalParityResultString(t *testing.T) {
	// Test match case
	matchResult := &LocalParityResult{
		LocalDigest:    "sha256:abc",
		RegistryDigest: "sha256:abc",
		Artifact:       "test:v1",
		Registry:       "ghcr.io/test",
		Match:          true,
	}
	str := matchResult.String()
	if str == "" {
		t.Error("String() returned empty for match case")
	}

	// Test mismatch case
	mismatchResult := &LocalParityResult{
		LocalDigest:    "sha256:abc",
		RegistryDigest: "sha256:def",
		Artifact:       "test:v1",
		Registry:       "ghcr.io/test",
		Match:          false,
	}
	str = mismatchResult.String()
	if str == "" {
		t.Error("String() returned empty for mismatch case")
	}
}
