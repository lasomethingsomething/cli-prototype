package hardentui

import (
	"strings"
	"testing"

	"github.com/lasomethingsomething/cli-prototype/internal/workflow"
)

func TestInitCreatesWorkflow(t *testing.T) {
	m := New("ghcr.io/example")

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

// TestSBOMPanelsShowSelectedTool verifies the SBOM panel and context tab
// render the tool and format actually selected instead of a fixed "Syft".
func TestSBOMPanelsShowSelectedTool(t *testing.T) {
	m := New("ghcr.io/example")
	m.SetSBOMTool("trivy", workflow.CycloneDXJSON)

	if got := m.renderSBOMPanel(); !strings.Contains(got, "using trivy, cyclonedx-json") {
		t.Errorf("renderSBOMPanel() = %q, want the selected tool and format", got)
	}

	ctx := NewHardenContextModel()
	ctx.SetSBOMTool("trivy", string(workflow.CycloneDXJSON))
	tab := ctx.renderSBOMTab()
	for _, want := range []string{"Tool: trivy", "Format: cyclonedx-json"} {
		if !strings.Contains(tab, want) {
			t.Errorf("renderSBOMTab() = %q, want %q", tab, want)
		}
	}
	if strings.Contains(tab, "Syft") {
		t.Errorf("renderSBOMTab() still mentions Syft: %q", tab)
	}
}

func TestSBOMDefaultsAreRecommendedTool(t *testing.T) {
	ctx := NewHardenContextModel()
	if ctx.SBOMTool != workflow.SBOMToolOptions().Recommended() || ctx.SBOMFormat != string(workflow.SPDXJSON) {
		t.Errorf("defaults = %q/%q", ctx.SBOMTool, ctx.SBOMFormat)
	}
}
