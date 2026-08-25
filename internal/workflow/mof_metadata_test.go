package workflow

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewMOFMetadataGenerator(t *testing.T) {
	gen := NewMOFMetadataGenerator()

	if gen == nil {
		t.Fatal("NewMOFMetadataGenerator returned nil")
	}

	if gen.releaseType != "model" {
		t.Errorf("Expected default releaseType to be 'model', got '%s'", gen.releaseType)
	}
}

func TestMOFMetadataGeneratorSetModelInfo(t *testing.T) {
	gen := NewMOFMetadataGenerator()
	gen.SetModelInfo("phi-4-mini", "/path/to/model", "ghcr.io/org/phi:1.0")

	if gen.modelName != "phi-4-mini" {
		t.Errorf("Expected modelName 'phi-4-mini', got '%s'", gen.modelName)
	}
	if gen.modelPath != "/path/to/model" {
		t.Errorf("Expected modelPath '/path/to/model', got '%s'", gen.modelPath)
	}
	if gen.artifactName != "ghcr.io/org/phi:1.0" {
		t.Errorf("Expected artifactName 'ghcr.io/org/phi:1.0', got '%s'", gen.artifactName)
	}
}

func TestMOFMetadataGeneratorSetMOFClassification(t *testing.T) {
	gen := NewMOFMetadataGenerator()
	gen.SetMOFClassification("II", []string{"weights", "code"}, "Partially open")

	if gen.mofClass != "II" {
		t.Errorf("Expected mofClass 'II', got '%s'", gen.mofClass)
	}
	if len(gen.components) != 2 {
		t.Errorf("Expected 2 components, got %d", len(gen.components))
	}
	if gen.explanation != "Partially open" {
		t.Errorf("Expected explanation 'Partially open', got '%s'", gen.explanation)
	}
}

func TestMOFMetadataGeneratorSetReleaseInfo(t *testing.T) {
	gen := NewMOFMetadataGenerator()
	gen.SetReleaseInfo("my-model", "2.0.0", "2026-01-01", "model")

	if gen.releaseName != "my-model" {
		t.Errorf("Expected releaseName 'my-model', got '%s'", gen.releaseName)
	}
	if gen.releaseVersion != "2.0.0" {
		t.Errorf("Expected releaseVersion '2.0.0', got '%s'", gen.releaseVersion)
	}
	if gen.releaseDate != "2026-01-01" {
		t.Errorf("Expected releaseDate '2026-01-01', got '%s'", gen.releaseDate)
	}
	if gen.releaseType != "model" {
		t.Errorf("Expected releaseType 'model', got '%s'", gen.releaseType)
	}
}

func TestMOFMetadataDefaultLicense(t *testing.T) {
	// Test that the default license is CC-BY-4.0 when not specified
	gen := NewMOFMetadataGenerator()
	gen.SetModelInfo("test-model", "/path/to/model", "ghcr.io/test:1.0")
	gen.SetMOFClassification("I", []string{"weights"}, "Full")
	gen.SetReleaseInfo("test-model", "1.0.0", "2026-01-01", "model")
	// Note: NOT setting license - should default to CC-BY-4.0

	metadata := gen.Generate()

	if metadata.Release.License != "CC-BY-4.0" {
		t.Errorf("Expected default license 'CC-BY-4.0', got '%s'", metadata.Release.License)
	}
}

func TestMOFMetadataCustomLicense(t *testing.T) {
	// Test that custom license can be set
	gen := NewMOFMetadataGenerator()
	gen.SetModelInfo("test-model", "/path/to/model", "ghcr.io/test:1.0")
	gen.SetMOFClassification("I", []string{"weights"}, "Full")
	gen.SetReleaseInfo("test-model", "1.0.0", "2026-01-01", "model")
	gen.SetReleaseLicense("MIT")

	metadata := gen.Generate()

	if metadata.Release.License != "MIT" {
		t.Errorf("Expected license 'MIT', got '%s'", metadata.Release.License)
	}
}

func TestMOFMetadataGeneratorGenerate(t *testing.T) {
	gen := NewMOFMetadataGenerator()
	gen.SetModelInfo("test-model", "/path/to/model", "ghcr.io/test:1.0")
	gen.SetMOFClassification("I", []string{"weights", "code", "training-data", "documentation", "license"}, "Fully open")
	gen.SetReleaseInfo("test-model", "1.0.0", "2026-01-01", "model")

	metadata := gen.Generate()

	if metadata == nil {
		t.Fatal("Generate() returned nil")
	}

	if metadata.MOFVersion != "1.0" {
		t.Errorf("Expected MOFVersion '1.0', got '%s'", metadata.MOFVersion)
	}

	if metadata.Generator != "model-cli" {
		t.Errorf("Expected Generator 'model-cli', got '%s'", metadata.Generator)
	}

	if metadata.Model.Name != "test-model" {
		t.Errorf("Expected Model.Name 'test-model', got '%s'", metadata.Model.Name)
	}

	if metadata.Model.Class != "I" {
		t.Errorf("Expected Model.Class 'I', got '%s'", metadata.Model.Class)
	}

	if metadata.Model.ClassName != "Fully Open" {
		t.Errorf("Expected Model.ClassName 'Fully Open', got '%s'", metadata.Model.ClassName)
	}

	if metadata.Release.Version != "1.0.0" {
		t.Errorf("Expected Release.Version '1.0.0', got '%s'", metadata.Release.Version)
	}

	// Check components
	if len(metadata.Components) != 5 {
		t.Errorf("Expected 5 components, got %d", len(metadata.Components))
	}

	// Verify all components are present
	for _, comp := range metadata.Components {
		if !comp.Present {
			t.Errorf("Component %s should be present", comp.Type)
		}
	}
}

func TestMOFMetadataGeneratorGenerateWithDefaults(t *testing.T) {
	gen := NewMOFMetadataGenerator()
	gen.SetModelInfo("test-model", "/path/to/model", "")
	gen.SetMOFClassification("III", []string{}, "Closed")
	// Don't set release info - should use defaults

	metadata := gen.Generate()

	// Release should have defaults
	if metadata.Release.Name != "test-model" {
		t.Errorf("Expected Release.Name to default to modelName 'test-model', got '%s'", metadata.Release.Name)
	}

	if metadata.Release.Version != "1.0.0" {
		t.Errorf("Expected Release.Version to default to '1.0.0', got '%s'", metadata.Release.Version)
	}

	if metadata.Release.Date == "" {
		t.Error("Expected Release.Date to have a default value")
	}

	if metadata.Model.Class != "III" {
		t.Errorf("Expected Model.Class 'III', got '%s'", metadata.Model.Class)
	}

	if metadata.Model.ClassName != "Closed/Proprietary" {
		t.Errorf("Expected Model.ClassName 'Closed/Proprietary', got '%s'", metadata.Model.ClassName)
	}
}

func TestMOFMetadataGeneratorWriteToFile(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "mof_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	gen := NewMOFMetadataGenerator()
	gen.SetModelInfo("test-model", tmpDir, "ghcr.io/test:1.0")
	gen.SetMOFClassification("II", []string{"weights", "license"}, "Partially open")
	gen.SetReleaseInfo("test-model", "1.0.0", "2026-01-01", "model")

	outputPath := filepath.Join(tmpDir, "mof.json")
	err = gen.WriteToFile(outputPath, "json")
	if err != nil {
		t.Fatalf("Failed to write MOF metadata: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Fatal("MOF metadata file was not created")
	}

	// Read and parse the file
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read MOF metadata file: %v", err)
	}

	var metadata MOFMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		t.Fatalf("Failed to parse MOF metadata file as JSON: %v", err)
	}

	if metadata.Model.Name != "test-model" {
		t.Errorf("Expected model name 'test-model', got '%s'", metadata.Model.Name)
	}

	if metadata.Model.Class != "II" {
		t.Errorf("Expected MOF class 'II', got '%s'", metadata.Model.Class)
	}

	// Check that components are correctly marked
	weightsPresent := false
	codePresent := false
	licensePresent := false

	for _, comp := range metadata.Components {
		switch comp.Type {
		case "weights":
			weightsPresent = comp.Present
		case "code":
			codePresent = comp.Present
		case "license":
			licensePresent = comp.Present
		}
	}

	if !weightsPresent {
		t.Error("Expected weights component to be present")
	}
	if codePresent {
		t.Error("Expected code component to NOT be present")
	}
	if !licensePresent {
		t.Error("Expected license component to be present")
	}
}

func TestMOFMetadataFromClassification(t *testing.T) {
	// Create a temp directory with model files
	tmpDir, err := os.MkdirTemp("", "mof_class_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create some model files
	weightsFile := filepath.Join(tmpDir, "model.safetensors")
	codeFile := filepath.Join(tmpDir, "train.py")
	licenseFile := filepath.Join(tmpDir, "LICENSE")

	if err := os.WriteFile(weightsFile, []byte("weights"), 0644); err != nil {
		t.Fatalf("Failed to create weights file: %v", err)
	}
	if err := os.WriteFile(codeFile, []byte("code"), 0644); err != nil {
		t.Fatalf("Failed to create code file: %v", err)
	}
	if err := os.WriteFile(licenseFile, []byte("MIT"), 0644); err != nil {
		t.Fatalf("Failed to create license file: %v", err)
	}

	// Classify the model
	classStr, result, err := ClassifyModelPath(tmpDir)
	if err != nil {
		t.Fatalf("Failed to classify model: %v", err)
	}

	// Generate metadata from classification
	metadata := MOFMetadataFromClassification("test-model", tmpDir, "ghcr.io/test:1.0", "1.0.0", result)

	if metadata == nil {
		t.Fatal("MOFMetadataFromClassification returned nil")
	}

	if metadata.Model.Class != classStr {
		t.Errorf("Expected model class '%s', got '%s'", classStr, metadata.Model.Class)
	}

	// Should have weights, code, license in components
	foundWeights := false
	foundCode := false
	foundLicense := false
	for _, comp := range metadata.Components {
		if comp.Type == "weights" && comp.Present {
			foundWeights = true
		}
		if comp.Type == "code" && comp.Present {
			foundCode = true
		}
		if comp.Type == "license" && comp.Present {
			foundLicense = true
		}
	}

	if !foundWeights {
		t.Error("Expected weights component in MOF metadata")
	}
	if !foundCode {
		t.Error("Expected code component in MOF metadata")
	}
	if !foundLicense {
		t.Error("Expected license component in MOF metadata")
	}

	if metadata.Generator != "model-cli" {
		t.Errorf("Expected generator 'model-cli', got '%s'", metadata.Generator)
	}
}

func TestGetMOFClassName(t *testing.T) {
	tests := []struct {
		class    string
		expected string
	}{
		{"I", "Fully Open"},
		{"II", "Partially Open"},
		{"III", "Closed/Proprietary"},
		{"unknown", "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.class, func(t *testing.T) {
			result := getMOFClassName(tt.class)
			if result != tt.expected {
				t.Errorf("getMOFClassName(%q) = %q, want %q", tt.class, result, tt.expected)
			}
		})
	}
}

func TestSliceContains(t *testing.T) {
	tests := []struct {
		slice    []string
		value    string
		expected bool
	}{
		{[]string{"a", "b", "c"}, "b", true},
		{[]string{"a", "b", "c"}, "d", false},
		{[]string{}, "a", false},
		{nil, "a", false},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := sliceContains(tt.slice, tt.value)
			if result != tt.expected {
				t.Errorf("sliceContains(%v, %q) = %v, want %v", tt.slice, tt.value, result, tt.expected)
			}
		})
	}
}

func TestMOFMetadataJSONStructure(t *testing.T) {
	gen := NewMOFMetadataGenerator()
	gen.SetModelInfo("test", "/path", "artifact")
	gen.SetMOFClassification("I", []string{"weights", "code", "training-data", "documentation", "license"}, "Full")
	gen.SetReleaseInfo("test", "1.0", "2026-01-01", "model")

	metadata := gen.Generate()

	// Marshal to JSON
	jsonData, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal metadata to JSON: %v", err)
	}

	// Verify it contains required fields
	jsonStr := string(jsonData)

	// These fields should be in the JSON output
	requiredFields := []string{
		`"mof_version"`,
		`"generator"`,
		`"generated_at"`,
		`"release"`,
		`"model"`,
		`"components"`,
	}

	for _, field := range requiredFields {
		if !containsString(jsonStr, field) {
			t.Errorf("JSON output missing required field: %s", field)
		}
	}
}

func containsString(s, substr string) bool {
	return strings.Contains(s, substr)
}

// Simpler contains for strings
func TestMOFMetadataTimestamp(t *testing.T) {
	before := time.Now()
	gen := NewMOFMetadataGenerator()
	gen.SetModelInfo("test", "/path", "artifact")
	gen.SetMOFClassification("I", []string{"weights"}, "Full")

	metadata := gen.Generate()

	after := time.Now()

	// GeneratedAt should be between before and after
	if metadata.GeneratedAt.Before(before) || metadata.GeneratedAt.After(after) {
		t.Errorf("GeneratedAt timestamp is not in expected range: %v", metadata.GeneratedAt)
	}
}
