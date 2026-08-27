package packagetui

import (
	"strings"
	"testing"
)

func TestInitCreatesWorkflow(t *testing.T) {
	m := New("oras")

	if cmd := m.Init(); cmd != nil {
		t.Fatalf("Init() returned unexpected command: %v", cmd)
	}

	if m.workflow == nil {
		t.Fatal("Init() did not create a workflow")
	}

	if m.err != nil {
		t.Fatalf("Init() set unexpected error: %v", m.err)
	}
}

// The SBOM is a required prerequisite (Phase 1 Step 2), so the TUI must
// default to generating it and only skip it when explicitly told to.
func TestGenerateSBOMDefaultsToTrue(t *testing.T) {
	m := New("oras")

	if !m.generateSBOM {
		t.Fatal("New() should default generateSBOM to true")
	}

	if !strings.Contains(m.renderReviewSummary(), "Generate SBOM: true") {
		t.Errorf("review panel should show SBOM generation enabled, got:\n%s", m.renderReviewSummary())
	}

	m.loading = true
	if !strings.Contains(m.renderPackagingProgress(), "Generating SBOM") {
		t.Errorf("progress panel should list the SBOM step by default, got:\n%s", m.renderPackagingProgress())
	}
}

func TestSetGenerateSBOMFalse(t *testing.T) {
	m := New("oras")
	m.SetGenerateSBOM(false)

	if m.generateSBOM {
		t.Fatal("SetGenerateSBOM(false) should disable SBOM generation")
	}

	if !strings.Contains(m.renderReviewSummary(), "Generate SBOM: false") {
		t.Errorf("review panel should show SBOM generation disabled, got:\n%s", m.renderReviewSummary())
	}

	m.loading = true
	if strings.Contains(m.renderPackagingProgress(), "Generating SBOM") {
		t.Errorf("progress panel should not list the SBOM step when disabled, got:\n%s", m.renderPackagingProgress())
	}
}
