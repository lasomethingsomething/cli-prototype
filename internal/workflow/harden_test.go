package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestHardenWorkflowSBOMFailureBlocks verifies that SBOM generation failure blocks the workflow
// as required by Phase 1 Step 2 (Issue #89)
func TestHardenWorkflowSBOMFailureBlocks(t *testing.T) {
	modelPath := t.TempDir()
	if err := os.WriteFile(filepath.Join(modelPath, "model.txt"), []byte("weights"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a harden workflow with a non-existent SBOM tool
	hw := NewHardenWorkflow("")
	hw.SetHardenInfo("test-model", modelPath, "test:v1")
	hw.SetOptions(true, false)                   // generateSBOM=true, includeMOF=false
	hw.SetSBOMTool("nonexistent-tool", SPDXJSON) // Use non-existent tool

	// Run should fail because SBOM tool is not available
	err := hw.Run()
	if err == nil {
		t.Fatal("expected SBOM generation to fail and block workflow, got nil")
	}

	// Verify the error message mentions SBOM is a prerequisite
	if !strings.Contains(err.Error(), "SBOM") || !strings.Contains(err.Error(), "prerequisite") {
		t.Errorf("expected error to mention SBOM prerequisite, got: %v", err)
	}
}

// TestHardenWorkflowSBOMGenerateFailureBlocks verifies that a failure inside
// the SBOM generator itself (tool installed, scan fails) blocks hardening
// before MOF classification runs (Issue #89).
func TestHardenWorkflowSBOMGenerateFailureBlocks(t *testing.T) {
	fakeSyft(t, "echo 'boom' >&2; exit 1")

	modelPath := t.TempDir()
	if err := os.WriteFile(filepath.Join(modelPath, "model.txt"), []byte("weights"), 0644); err != nil {
		t.Fatal(err)
	}

	hw := NewHardenWorkflow("")
	hw.SetHardenInfo("test-model", modelPath, "test:v1")
	hw.SetOptions(true, true)
	hw.SetSBOMTool("syft", SPDXJSON)

	err := hw.Run()
	if err == nil {
		t.Fatal("expected SBOM generation failure to block workflow, got nil")
	}
	if !strings.Contains(err.Error(), "syft generation failed") || !strings.Contains(err.Error(), "prerequisite") {
		t.Errorf("error = %v, want SBOM generation failure naming the prerequisite", err)
	}
	if hw.MOFClass() != "" || hw.MOFConfigPath() != "" {
		t.Errorf("MOF step ran despite SBOM failure: class %q, config %q", hw.MOFClass(), hw.MOFConfigPath())
	}
}

// TestHardenWorkflowDefaults verifies default values
func TestHardenWorkflowDefaults(t *testing.T) {
	hw := NewHardenWorkflow("oras")

	if !hw.generateSBOM {
		t.Error("generateSBOM default = false, want true")
	}
	if !hw.includeMOF {
		t.Error("includeMOF default = false, want true")
	}
	if hw.sbomTool != "syft" {
		t.Errorf("sbomTool default = %q, want %q", hw.sbomTool, "syft")
	}
	if hw.sbomFormat != SPDXJSON {
		t.Errorf("sbomFormat default = %q, want %q", hw.sbomFormat, SPDXJSON)
	}
	if hw.annotations == nil {
		t.Error("annotations default is nil, want non-nil")
	}
}
