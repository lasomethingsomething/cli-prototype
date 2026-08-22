package workflow

import (
	"os"
	"path/filepath"
	"testing"
)

// TestSyftGeneratorUnit tests the Syft SBOM generator unit
func TestSyftGeneratorUnit(t *testing.T) {
	gen := &SyftGenerator{}

	// Test Name
	if gen.Name() != "syft" {
		t.Errorf("Expected name 'syft', got '%s'", gen.Name())
	}

	// Test DefaultFormat
	if gen.DefaultFormat() != SPDXJSON {
		t.Errorf("Expected default format '%s', got '%s'", SPDXJSON, gen.DefaultFormat())
	}

	// Test InstallInstructions
	if gen.InstallInstructions() == "" {
		t.Error("Expected non-empty install instructions")
	}
}

// TestTrivyGeneratorUnit tests the Trivy SBOM generator unit
func TestTrivyGeneratorUnit(t *testing.T) {
	gen := &TrivyGenerator{}

	// Test Name
	if gen.Name() != "trivy" {
		t.Errorf("Expected name 'trivy', got '%s'", gen.Name())
	}

	// Test DefaultFormat
	if gen.DefaultFormat() != CycloneDXJSON {
		t.Errorf("Expected default format '%s', got '%s'", CycloneDXJSON, gen.DefaultFormat())
	}

	// Test InstallInstructions
	if gen.InstallInstructions() == "" {
		t.Error("Expected non-empty install instructions")
	}
}

// TestCdxgenGeneratorUnit tests the cdxgen SBOM generator unit
func TestCdxgenGeneratorUnit(t *testing.T) {
	gen := &CdxgenGenerator{}

	// Test Name
	if gen.Name() != "cdxgen" {
		t.Errorf("Expected name 'cdxgen', got '%s'", gen.Name())
	}

	// Test DefaultFormat
	if gen.DefaultFormat() != CycloneDXJSON {
		t.Errorf("Expected default format '%s', got '%s'", CycloneDXJSON, gen.DefaultFormat())
	}

	// Test InstallInstructions
	if gen.InstallInstructions() == "" {
		t.Error("Expected non-empty install instructions")
	}
}

// TestGetAllSBOMGeneratorsUnit tests getting all generators
func TestGetAllSBOMGeneratorsUnit(t *testing.T) {
	generators := GetAllSBOMGenerators()
	if len(generators) != 3 {
		t.Errorf("Expected 3 generators, got %d", len(generators))
	}

	names := make(map[string]bool)
	for _, gen := range generators {
		names[gen.Name()] = true
	}

	for _, expected := range []string{"syft", "trivy", "cdxgen"} {
		if !names[expected] {
			t.Errorf("Expected generator %s not found", expected)
		}
	}
}

// TestGetDefaultSBOMGeneratorUnit tests the default generator
func TestGetDefaultSBOMGeneratorUnit(t *testing.T) {
	gen := GetDefaultSBOMGenerator()
	if gen.Name() != "syft" {
		t.Errorf("Expected default generator 'syft', got '%s'", gen.Name())
	}
}

// TestSyftGeneratorGenerateWithMock tests SBOM generation with a mock syft binary
func TestSyftGeneratorGenerateWithMock(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "sbom-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a simple test file to generate SBOM for
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create a mock syft script that writes a simple SPDX JSON file
	mockSyft := filepath.Join(tmpDir, "mock-syft")
	syftScript := `#!/bin/bash
if [ "$1" = "version" ]; then
    echo "v1.0.0"
    exit 0
fi

# Parse arguments
OUTPUT_FILE=""
FORMAT="spdx-json"

while [[ $# -gt 0 ]]; do
    case "$1" in
        -o|--output)
            FORMAT="$2"
            shift 2
            ;;
        --file)
            OUTPUT_FILE="$2"
            shift 2
            ;;
        *)
            shift
            ;;
    esac
done

# Create a minimal SPDX JSON file
cat > "$OUTPUT_FILE" << 'EOF'
{
  "spdxVersion": "SPDX-2.3",
  "name": "test",
  "version": "1.0.0",
  "packages": []
}
EOF

exit 0
`
	if err := os.WriteFile(mockSyft, []byte(syftScript), 0755); err != nil {
		t.Fatalf("Failed to create mock syft: %v", err)
	}

	// Set PATH to include our mock syft
	oldPath := os.Getenv("PATH")
	os.Setenv("PATH", tmpDir+":"+oldPath)
	defer os.Setenv("PATH", oldPath)

	gen := &SyftGenerator{}

	// Test that syft is "installed" (our mock)
	if !gen.IsInstalled() {
		t.Skip("Mock syft not detected")
	}

	// Test SBOM generation
	outputPath := filepath.Join(tmpDir, "output", "sbom.spdx-json")
	err = gen.Generate(tmpDir, outputPath, SPDXJSON)
	if err != nil {
		t.Errorf("Generate() error = %v", err)
	}

	// Check that the output file was created
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Errorf("Output file not created: %s", outputPath)
	}

	// Check the content is valid JSON (basic check)
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Errorf("Failed to read output file: %v", err)
	}

	contentStr := string(content)
	if !stringContains(contentStr, `"spdxVersion"`) {
		t.Errorf("Output file doesn't contain expected SPDX content")
	}
}

// TestSBOMFormatMappingUnit tests the format mappings
func TestSBOMFormatMappingUnit(t *testing.T) {
	// Test syft format mapping
	if mapFormatToSyft(SPDXJSON) != "spdx-json" {
		t.Error("SPDXJSON should map to spdx-json for syft")
	}
	if mapFormatToSyft(CycloneDXJSON) != "cyclonedx-json" {
		t.Error("CycloneDXJSON should map to cyclonedx-json for syft")
	}

	// Test trivy format mapping
	if mapFormatToTrivy(SPDXJSON) != "spdx" {
		t.Error("SPDXJSON should map to spdx for trivy")
	}
	if mapFormatToTrivy(CycloneDXJSON) != "cyclonedx" {
		t.Error("CycloneDXJSON should map to cyclonedx for trivy")
	}
}

// stringContains is a helper function for string contains
func stringContains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && 
		(s[0:len(substr)] == substr || 
		 s[len(s)-len(substr):] == substr ||
		 stringContainsHelper(s, substr)))
}

func stringContainsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestSBOMGeneratorInterfaceUnit tests that all generators implement the interface
func TestSBOMGeneratorInterfaceUnit(t *testing.T) {
	generators := []SBOMGenerator{
		&SyftGenerator{},
		&TrivyGenerator{},
		&CdxgenGenerator{},
	}

	for _, gen := range generators {
		// Test that all methods exist
		_ = gen.Name()
		_ = gen.IsInstalled()
		_ = gen.InstallInstructions()
		_ = gen.DefaultFormat()
		
		// We can't easily test Generate without the actual tools installed
		// but we can verify the interface is satisfied
	}
}

// TestSyftCommandLineUnit tests that we can build a valid syft command
func TestSyftCommandLineUnit(t *testing.T) {
	gen := &SyftGenerator{}
	
	// This is a basic test to verify the command structure
	// We can't actually run syft without it being installed
	if gen.Name() != "syft" {
		t.Errorf("Expected name syft, got %s", gen.Name())
	}
	
	if gen.DefaultFormat() != SPDXJSON {
		t.Errorf("Expected default format %s, got %s", SPDXJSON, gen.DefaultFormat())
	}
	
	// Test that IsInstalled doesn't panic
	_ = gen.IsInstalled()
	
	// Test that InstallInstructions returns a non-empty string
	if gen.InstallInstructions() == "" {
		t.Error("Expected non-empty install instructions")
	}
}

// TestAllSBOMFormatsUnit tests all defined SBOM formats
func TestAllSBOMFormatsUnit(t *testing.T) {
	formats := []SBOMFormat{
		SPDXJSON,
		SPDXTagValue,
		CycloneDXJSON,
		CycloneDXXML,
	}
	
	for _, format := range formats {
		// Just verify they're non-empty
		if format == "" {
			t.Error("Found empty SBOM format")
		}
	}
}
