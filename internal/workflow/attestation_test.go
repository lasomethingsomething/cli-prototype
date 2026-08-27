package workflow

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// fakeSigningProvider is a SigningProvider test double that records what it
// was asked to sign instead of shelling out.
type fakeSigningProvider struct {
	signed []string
}

func (f *fakeSigningProvider) Name() string                { return "fake-signer" }
func (f *fakeSigningProvider) IsInstalled() bool           { return true }
func (f *fakeSigningProvider) InstallInstructions() string { return "n/a" }
func (f *fakeSigningProvider) Sign(artifact, keyRef string) error {
	f.signed = append(f.signed, artifact)
	return nil
}
func (f *fakeSigningProvider) Verify(artifact string) error            { return nil }
func (f *fakeSigningProvider) GetSignaturePath(artifact string) string { return artifact + ".fake.sig" }

// TestAttestSignedArtifact verifies the sign step writes the attestation to
// disk and attaches that same document to the artifact as a referrer, with
// the registry's digest as the subject.
func TestAttestSignedArtifact(t *testing.T) {
	reg := &fakeRegistryProvider{
		installed:      true,
		artifactDigest: map[string]string{"my-model:v1": "sha256:abc123"},
	}
	signer := &fakeSigningProvider{}
	artifact := SignedArtifact{
		Destination:   "ghcr.io/my-org",
		Name:          "my-model:v1",
		Signer:        "sigstore",
		SignaturePath: "ghcr.io/my-org/my-model:v1.sig",
	}
	outputPath := filepath.Join(t.TempDir(), "my-model:v1.provenance.json")

	result, err := AttestSignedArtifact(reg, signer, artifact, outputPath)
	if err != nil {
		t.Fatalf("AttestSignedArtifact() error = %v", err)
	}
	if !result.Attached {
		t.Error("Attached = false, want true")
	}
	if result.Path != outputPath {
		t.Errorf("Path = %q, want %q", result.Path, outputPath)
	}

	// The local file holds a valid attestation for the signed artifact.
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("attestation not written: %v", err)
	}
	var written ProvenanceAttestation
	if err := json.Unmarshal(data, &written); err != nil {
		t.Fatalf("attestation file is not JSON: %v", err)
	}
	if err := ValidateAttestation(&written); err != nil {
		t.Errorf("written attestation is invalid: %v", err)
	}
	subject := written.StatementHeader.Subject[0]
	if subject.Name != "ghcr.io/my-org/my-model:v1" {
		t.Errorf("subject name = %q, want %q", subject.Name, "ghcr.io/my-org/my-model:v1")
	}
	if subject.Digest["sha256"] != "abc123" {
		t.Errorf("subject digest = %v, want sha256=abc123 from the registry", subject.Digest)
	}
	predicate := written.Statement.Predicate
	if predicate.Recipe.Type != RecipeSign {
		t.Errorf("recipe type = %q, want %q", predicate.Recipe.Type, RecipeSign)
	}
	if predicate.Recipe.EntryPoint != "sign" || predicate.Recipe.DefinedIn != "signer:sigstore" {
		t.Errorf("recipe = %+v, want entry point sign defined in signer:sigstore", predicate.Recipe)
	}
	if len(predicate.Invocations) != 1 || predicate.Invocations[0].Configuration["signature"] != artifact.SignaturePath {
		t.Errorf("invocations = %+v, want one recording the signature path", predicate.Invocations)
	}

	// The registry holds the identical document as a provenance referrer of the artifact.
	if len(reg.referrers) != 1 {
		t.Fatalf("got %d referrers, want 1", len(reg.referrers))
	}
	ref := reg.referrers[0]
	if ref.artifact != "my-model:v1" || ref.registry != "ghcr.io/my-org" {
		t.Errorf("referrer attached to %s/%s, want ghcr.io/my-org/my-model:v1", ref.registry, ref.artifact)
	}
	if ref.referrerType != AttestationTypeProvenance {
		t.Errorf("referrer type = %q, want %q", ref.referrerType, AttestationTypeProvenance)
	}
	if string(ref.data) != string(data) {
		t.Error("referrer content differs from the attestation written to disk")
	}
	if ref.annotations[AnnotationSigningFramework] != signer.Name() {
		t.Errorf("referrer %s = %q, want %q", AnnotationSigningFramework, ref.annotations[AnnotationSigningFramework], signer.Name())
	}
	if len(signer.signed) != 0 {
		t.Errorf("attestation step signed %v; the artifact itself is signed by the sign command", signer.signed)
	}
}

// TestAttestSignedArtifactLocalOnly verifies a reference without a registry
// prefix (or no registry provider) still gets a local attestation.
func TestAttestSignedArtifactLocalOnly(t *testing.T) {
	cases := map[string]struct {
		provider RegistryProvider
		artifact SignedArtifact
	}{
		"no registry prefix": {&fakeRegistryProvider{installed: true}, SignedArtifact{Name: "my-model:v1", Signer: "sigstore"}},
		"no provider":        {nil, SignedArtifact{Destination: "ghcr.io/my-org", Name: "my-model:v1", Signer: "sigstore"}},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			outputPath := filepath.Join(t.TempDir(), "attestation.json")
			result, err := AttestSignedArtifact(tc.provider, &fakeSigningProvider{}, tc.artifact, outputPath)
			if err != nil {
				t.Fatalf("AttestSignedArtifact() error = %v", err)
			}
			if result.Attached {
				t.Error("Attached = true, want false")
			}
			if _, err := os.Stat(outputPath); err != nil {
				t.Errorf("attestation not written locally: %v", err)
			}
			if got := result.Attestation.StatementHeader.Subject[0]; got.Name != tc.artifact.Reference() || got.Digest != nil {
				t.Errorf("subject = %+v, want name %q without digest", got, tc.artifact.Reference())
			}
			if fake, ok := tc.provider.(*fakeRegistryProvider); ok && len(fake.referrers) != 0 {
				t.Errorf("referrers pushed: %d, want none", len(fake.referrers))
			}
		})
	}
}

// TestAttestSignedArtifactRegistryFailures verifies registry problems degrade
// to warnings: the local attestation is still written.
func TestAttestSignedArtifactRegistryFailures(t *testing.T) {
	artifact := SignedArtifact{Destination: "ghcr.io/my-org", Name: "my-model:v1", Signer: "sigstore"}

	t.Run("digest lookup fails", func(t *testing.T) {
		reg := &fakeRegistryProvider{installed: true} // no digest known for the artifact
		outputPath := filepath.Join(t.TempDir(), "attestation.json")
		result, err := AttestSignedArtifact(reg, nil, artifact, outputPath)
		if err != nil {
			t.Fatalf("AttestSignedArtifact() error = %v", err)
		}
		if result.Attestation.StatementHeader.Subject[0].Digest != nil {
			t.Errorf("subject digest = %v, want none", result.Attestation.StatementHeader.Subject[0].Digest)
		}
		if !result.Attached || len(reg.referrers) != 1 {
			t.Errorf("Attached = %v with %d referrers, want the attestation attached anyway", result.Attached, len(reg.referrers))
		}
	})

	t.Run("referrer push fails", func(t *testing.T) {
		reg := &fakeRegistryProvider{
			installed:      true,
			artifactDigest: map[string]string{"my-model:v1": "sha256:abc123"},
			referrerErr:    errors.New("registry down"),
		}
		outputPath := filepath.Join(t.TempDir(), "attestation.json")
		result, err := AttestSignedArtifact(reg, nil, artifact, outputPath)
		if err != nil {
			t.Fatalf("AttestSignedArtifact() error = %v", err)
		}
		if result.Attached {
			t.Error("Attached = true, want false")
		}
		if _, err := os.Stat(outputPath); err != nil {
			t.Errorf("attestation not written locally: %v", err)
		}
	})
}

func TestDigestMap(t *testing.T) {
	cases := map[string]map[string]string{
		"sha256:abc123": {"sha256": "abc123"},
		"":              nil,
		"abc123":        nil,
		"sha256:":       nil,
		":abc123":       nil,
	}
	for in, want := range cases {
		got := DigestMap(in)
		if len(got) != len(want) {
			t.Errorf("DigestMap(%q) = %v, want %v", in, got, want)
			continue
		}
		for k, v := range want {
			if got[k] != v {
				t.Errorf("DigestMap(%q) = %v, want %v", in, got, want)
			}
		}
	}
}

func TestNewAttestationManager(t *testing.T) {
	am := NewAttestationManager(nil, nil, "ghcr.io/my-org")

	if am == nil {
		t.Fatal("NewAttestationManager returned nil")
	}

	if am.registry != "ghcr.io/my-org" {
		t.Errorf("Expected registry 'ghcr.io/my-org', got '%s'", am.registry)
	}

	if am.signer != nil {
		t.Error("Expected signer to be nil, but it was set")
	}

	if am.registryProvider != nil {
		t.Error("Expected registryProvider to be nil, but it was set")
	}

	// Verify buildStartTime was set (not zero)
	if am.buildStartTime.IsZero() {
		t.Error("Expected buildStartTime to be set, but it was zero")
	}
}

func TestValidateAndVerify(t *testing.T) {
	// Create a valid attestation
	pg := NewProvenanceGenerator()
	pg.SetSourceInfo("/path/to/model", "")
	pg.SetArtifactInfo("test-model:v1", nil)
	attestation := pg.Generate()

	// Create an attestation manager (without signer for this test)
	am := NewAttestationManager(nil, nil, "ghcr.io/my-org")

	// Validate and verify should succeed for a valid attestation
	// (signature verification will be skipped since we don't have a signer)
	err := am.ValidateAndVerify(attestation)
	if err != nil {
		t.Errorf("ValidateAndVerify failed for valid attestation: %v", err)
	}
}
