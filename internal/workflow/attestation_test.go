package workflow

import (
	"testing"
)

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
