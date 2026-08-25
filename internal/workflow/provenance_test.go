package workflow

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewProvenanceGenerator(t *testing.T) {
	pg := NewProvenanceGenerator()

	if pg == nil {
		t.Fatal("NewProvenanceGenerator returned nil")
	}

	if pg.builderID != "model-cli" {
		t.Errorf("Expected default builderID 'model-cli', got '%s'", pg.builderID)
	}
	if pg.buildType != "https://model-cli.dev/build/v1" {
		t.Errorf("Expected default buildType 'https://model-cli.dev/build/v1', got '%s'", pg.buildType)
	}
}

func TestProvenanceGeneratorSetMethods(t *testing.T) {
	pg := NewProvenanceGenerator()

	// Test SetBuildInfo
	pg.SetBuildInfo("custom-builder", "custom-type")
	if pg.builderID != "custom-builder" {
		t.Errorf("SetBuildInfo builderID: expected 'custom-builder', got '%s'", pg.builderID)
	}
	if pg.buildType != "custom-type" {
		t.Errorf("SetBuildInfo buildType: expected 'custom-type', got '%s'", pg.buildType)
	}

	// Test SetSourceInfo
	pg.SetSourceInfo("/path/to/source", "https://example.com/source")
	if pg.sourceID != "/path/to/source" {
		t.Errorf("SetSourceInfo sourceID: expected '/path/to/source', got '%s'", pg.sourceID)
	}
	if pg.sourceURI != "https://example.com/source" {
		t.Errorf("SetSourceInfo sourceURI: expected 'https://example.com/source', got '%s'", pg.sourceURI)
	}

	// Test SetArtifactInfo
	pg.SetArtifactInfo("my-artifact:v1", map[string]string{"sha256": "abc123"})
	if pg.artifactName != "my-artifact:v1" {
		t.Errorf("SetArtifactInfo artifactName: expected 'my-artifact:v1', got '%s'", pg.artifactName)
	}
	if pg.artifactDigest == nil || pg.artifactDigest["sha256"] != "abc123" {
		t.Errorf("SetArtifactInfo digest: expected sha256=abc123, got %v", pg.artifactDigest)
	}
}

func TestProvenanceGeneratorGenerate(t *testing.T) {
	pg := NewProvenanceGenerator()
	pg.SetSourceInfo("/path/to/model", "file:///path/to/model")
	pg.SetArtifactInfo("ghcr.io/my-org/my-model:v1", nil)

	attestation := pg.Generate()

	if attestation == nil {
		t.Fatal("Generate() returned nil")
	}

	// Validate header
	if attestation.StatementHeader.Type != "https://in-toto.io/Statement/v0.1" {
		t.Errorf("StatementHeader.Type: expected 'https://in-toto.io/Statement/v0.1', got '%s'", attestation.StatementHeader.Type)
	}
	if attestation.StatementHeader.PredicateType != "https://slsa.dev/provenance/v0.2" {
		t.Errorf("StatementHeader.PredicateType: expected 'https://slsa.dev/provenance/v0.2', got '%s'", attestation.StatementHeader.PredicateType)
	}

	// Validate subject
	if len(attestation.StatementHeader.Subject) == 0 {
		t.Error("StatementHeader.Subject is empty")
	} else {
		if attestation.StatementHeader.Subject[0].Name != "ghcr.io/my-org/my-model:v1" {
			t.Errorf("Subject.Name: expected 'ghcr.io/my-org/my-model:v1', got '%s'", attestation.StatementHeader.Subject[0].Name)
		}
	}

	// Validate predicate
	predicate := attestation.Statement.Predicate
	if predicate.BuildType != "https://model-cli.dev/build/v1" {
		t.Errorf("Predicate.BuildType: expected 'https://model-cli.dev/build/v1', got '%s'", predicate.BuildType)
	}
	if predicate.Builder.ID != "model-cli" {
		t.Errorf("Predicate.Builder.ID: expected 'model-cli', got '%s'", predicate.Builder.ID)
	}
	if predicate.Source.ID != "/path/to/model" {
		t.Errorf("Predicate.Source.ID: expected '/path/to/model', got '%s'", predicate.Source.ID)
	}

	// Validate metadata
	if predicate.Metadata.BuildFinishedOn.IsZero() {
		t.Error("Predicate.Metadata.BuildFinishedOn is zero")
	}
	if !predicate.Metadata.Completeness.Environment {
		t.Error("Predicate.Metadata.Completeness.Environment is false")
	}
	if !predicate.Metadata.Completeness.Parameters {
		t.Error("Predicate.Metadata.Completeness.Parameters is false")
	}
	if !predicate.Metadata.Completeness.Materials {
		t.Error("Predicate.Metadata.Completeness.Materials is false")
	}
	if predicate.Metadata.Reproducible {
		t.Error("Predicate.Metadata.Reproducible should be false for AI models")
	}

	// Validate build ID is unique
	if predicate.BuildID == "" {
		t.Error("Predicate.BuildID is empty")
	}
	if !strings.HasPrefix(predicate.BuildID, "model-cli-") {
		t.Errorf("Predicate.BuildID should start with 'model-cli-', got '%s'", predicate.BuildID)
	}
}

func TestProvenanceGeneratorWriteToFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "provenance_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	pg := NewProvenanceGenerator()
	pg.SetSourceInfo("/path/to/model", "")
	pg.SetArtifactInfo("test-model:v1", nil)

	outputPath := filepath.Join(tmpDir, "attestation.json")
	attestation, err := pg.WriteToFile(outputPath)
	if err != nil {
		t.Fatalf("WriteToFile failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Fatal("Attestation file was not created")
	}

	// Verify file contents
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read attestation file: %v", err)
	}

	var parsedAttestation ProvenanceAttestation
	if err := json.Unmarshal(data, &parsedAttestation); err != nil {
		t.Fatalf("Failed to parse attestation file: %v", err)
	}

	if parsedAttestation.Statement.Predicate.BuildType != "https://model-cli.dev/build/v1" {
		t.Errorf("BuildType mismatch in file")
	}

	// Verify the returned attestation matches
	if attestation.Statement.Predicate.BuildID != parsedAttestation.Statement.Predicate.BuildID {
		t.Error("Returned attestation doesn't match file contents")
	}
}

func TestValidateAttestation(t *testing.T) {
	// Test valid attestation
	pg := NewProvenanceGenerator()
	pg.SetSourceInfo("/path/to/model", "")
	pg.SetArtifactInfo("test-model:v1", nil)
	attestation := pg.Generate()

	if err := ValidateAttestation(attestation); err != nil {
		t.Errorf("ValidateAttestation failed for valid attestation: %v", err)
	}

	// Test nil attestation
	if err := ValidateAttestation(nil); err == nil {
		t.Error("ValidateAttestation should fail for nil attestation")
	}

	// Test invalid statement header type
	invalidAttestation := &ProvenanceAttestation{
		StatementHeader: StatementHeader{
			Type:          "invalid-type",
			PredicateType: "https://slsa.dev/provenance/v0.2",
		},
	}
	if err := ValidateAttestation(invalidAttestation); err == nil {
		t.Error("ValidateAttestation should fail for invalid statement header type")
	}

	// Test invalid predicate type
	invalidAttestation2 := &ProvenanceAttestation{
		StatementHeader: StatementHeader{
			Type:          "https://in-toto.io/Statement/v0.1",
			PredicateType: "invalid-predicate",
		},
	}
	if err := ValidateAttestation(invalidAttestation2); err == nil {
		t.Error("ValidateAttestation should fail for invalid predicate type")
	}

	// Test empty build type
	invalidAttestation3 := &ProvenanceAttestation{
		StatementHeader: StatementHeader{
			Type:          "https://in-toto.io/Statement/v0.1",
			PredicateType: "https://slsa.dev/provenance/v0.2",
		},
		Statement: Statement{
			Predicate: ProvenancePredicate{
				BuildType: "", // Empty
				Builder:   Builder{ID: "test"},
			},
		},
	}
	if err := ValidateAttestation(invalidAttestation3); err == nil {
		t.Error("ValidateAttestation should fail for empty build type")
	}

	// Test empty builder ID
	invalidAttestation4 := &ProvenanceAttestation{
		StatementHeader: StatementHeader{
			Type:          "https://in-toto.io/Statement/v0.1",
			PredicateType: "https://slsa.dev/provenance/v0.2",
		},
		Statement: Statement{
			Predicate: ProvenancePredicate{
				BuildType: "test",
				Builder:   Builder{ID: ""}, // Empty
			},
		},
	}
	if err := ValidateAttestation(invalidAttestation4); err == nil {
		t.Error("ValidateAttestation should fail for empty builder ID")
	}

	// Test zero timestamp
	invalidAttestation5 := &ProvenanceAttestation{
		StatementHeader: StatementHeader{
			Type:          "https://in-toto.io/Statement/v0.1",
			PredicateType: "https://slsa.dev/provenance/v0.2",
		},
		Statement: Statement{
			Predicate: ProvenancePredicate{
				BuildType: "test",
				Builder:   Builder{ID: "test"},
				Metadata: Metadata{
					BuildFinishedOn: time.Time{}, // Zero time
				},
			},
		},
	}
	if err := ValidateAttestation(invalidAttestation5); err == nil {
		t.Error("ValidateAttestation should fail for zero timestamp")
	}
}

func TestGetAttestationPath(t *testing.T) {
	path := GetAttestationPath("my-model:v1")
	expected := "my-model:v1.provenance.json"
	if path != expected {
		t.Errorf("GetAttestationPath('my-model:v1') = %q, want %q", path, expected)
	}

	path2 := GetAttestationPath("ghcr.io/my-org/my-model:latest")
	expected2 := "ghcr.io/my-org/my-model:latest.provenance.json"
	if path2 != expected2 {
		t.Errorf("GetAttestationPath('ghcr.io/...') = %q, want %q", path2, expected2)
	}
}

func TestProvenanceAttestationJSONStructure(t *testing.T) {
	pg := NewProvenanceGenerator()
	pg.SetSourceInfo("/path/to/model", "https://example.com/model")
	pg.SetArtifactInfo("test-model:v1", map[string]string{"sha256": "abc123"})

	attestation := pg.Generate()

	// Marshal to JSON
	jsonData, err := json.MarshalIndent(attestation, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal attestation to JSON: %v", err)
	}

	// Verify it contains required fields
	jsonStr := string(jsonData)

	// These fields should be in the JSON output
	requiredFields := []string{
		`"statement_header"`,
		`"type"`,
		`"predicate_type"`,
		`"predicate"`,
		`"buildType"`,
		`"buildId"`,
		`"builder"`,
		`"recipe"`,
		`"source"`,
		`"metadata"`,
	}

	for _, field := range requiredFields {
		if !strings.Contains(jsonStr, field) {
			t.Errorf("JSON output missing required field: %s", field)
		}
	}
}

func TestGenerateFromWorkflow(t *testing.T) {
	pg := NewProvenanceGenerator()

	// Create a minimal workflow for testing
	pf := &PackageWorkflow{
		registry:     "oras",
		modelPath:    "/path/to/model",
		artifactName: "test-model:v1",
	}

	attestation := pg.GenerateFromWorkflow(pf)

	if attestation == nil {
		t.Fatal("GenerateFromWorkflow returned nil")
	}

	// Verify source was set from workflow
	if attestation.Statement.Predicate.Source.ID != "/path/to/model" {
		t.Errorf("Source.ID: expected '/path/to/model', got '%s'", attestation.Statement.Predicate.Source.ID)
	}

	// Verify artifact was set from workflow
	if attestation.StatementHeader.Subject[0].Name != "test-model:v1" {
		t.Errorf("Subject.Name: expected 'test-model:v1', got '%s'", attestation.StatementHeader.Subject[0].Name)
	}

	// Verify recipe was set
	if !strings.Contains(attestation.Statement.Predicate.Recipe.DefinedIn, "registry:oras") {
		t.Errorf("Recipe.DefinedIn: expected to contain 'registry:oras', got '%s'", attestation.Statement.Predicate.Recipe.DefinedIn)
	}
}

func TestProvenanceGeneratorWithMaterials(t *testing.T) {
	pg := NewProvenanceGenerator()
	pg.SetSourceInfo("/path/to/model", "")
	pg.SetArtifactInfo("test-model:v1", nil)

	// Add materials
	pg.AddMaterial("https://github.com/example/repo", map[string]string{"sha256": "def456"})
	pg.AddMaterial("oci://ghcr.io/example/base:latest", map[string]string{"sha256": "ghi789"})

	attestation := pg.Generate()

	// Verify materials are included
	if len(attestation.Statement.Predicate.Materials) != 2 {
		t.Errorf("Expected 2 materials, got %d", len(attestation.Statement.Predicate.Materials))
	}

	// Verify first material
	if attestation.Statement.Predicate.Materials[0].URI != "https://github.com/example/repo" {
		t.Errorf("Material[0].URI: expected 'https://github.com/example/repo', got '%s'", attestation.Statement.Predicate.Materials[0].URI)
	}
	if attestation.Statement.Predicate.Materials[0].Digest["sha256"] != "def456" {
		t.Errorf("Material[0].Digest[sha256]: expected 'def456', got '%s'", attestation.Statement.Predicate.Materials[0].Digest["sha256"])
	}

	// Verify second material
	if attestation.Statement.Predicate.Materials[1].URI != "oci://ghcr.io/example/base:latest" {
		t.Errorf("Material[1].URI: expected 'oci://ghcr.io/example/base:latest', got '%s'", attestation.Statement.Predicate.Materials[1].URI)
	}

	// Verify completeness reflects materials
	if !attestation.Statement.Predicate.Metadata.Completeness.Materials {
		t.Error("Completeness.Materials should be true when materials are specified")
	}
}

func TestProvenanceGeneratorWithInvocationInfo(t *testing.T) {
	pg := NewProvenanceGenerator()
	pg.SetSourceInfo("/path/to/model", "")
	pg.SetArtifactInfo("test-model:v1", nil)

	// Set invocation info
	pg.SetInvocationInfo("build-12345", map[string]interface{}{
		"builder":   "model-cli/v1.0.0",
		"buildType": "https://model-cli.dev/build/push/v1",
		"registry":  "oras",
		"pushedAt":  "2024-01-01T00:00:00Z",
	})

	attestation := pg.Generate()

	// Verify invocation is included
	if len(attestation.Statement.Predicate.Invocations) != 1 {
		t.Errorf("Expected 1 invocation, got %d", len(attestation.Statement.Predicate.Invocations))
	}

	inv := attestation.Statement.Predicate.Invocations[0]

	// Verify build ID was used
	if attestation.Statement.Predicate.BuildID != "build-12345" {
		t.Errorf("BuildID: expected 'build-12345', got '%s'", attestation.Statement.Predicate.BuildID)
	}

	// Verify configuration was set
	if inv.Configuration == nil {
		t.Error("Invocation.Configuration is nil")
	} else {
		if inv.Configuration["builder"] != "model-cli/v1.0.0" {
			t.Errorf("Invocation.Configuration[builder]: expected 'model-cli/v1.0.0', got '%v'", inv.Configuration["builder"])
		}
		if inv.Configuration["registry"] != "oras" {
			t.Errorf("Invocation.Configuration[registry]: expected 'oras', got '%v'", inv.Configuration["registry"])
		}
	}

	// Verify parameters include artifact name
	if inv.Parameters["artifact"] != "test-model:v1" {
		t.Errorf("Invocation.Parameters[artifact]: expected 'test-model:v1', got '%s'", inv.Parameters["artifact"])
	}
}
