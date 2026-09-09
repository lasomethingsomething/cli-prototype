package tui

import (
	"strings"
	"testing"
)

func TestProgressPercentInRender(t *testing.T) {
	tests := []struct {
		step    int
		wantPct string
	}{
		{1, "14%"},
		{4, "57%"},
		{7, "100%"},
	}

	for _, tt := range tests {
		t.Run(tt.wantPct, func(t *testing.T) {
			p := NewContextPanel()
			p.SetStep(tt.step)
			out := p.Render()
			if !strings.Contains(out, tt.wantPct) {
				t.Errorf("Render() at step %d does not contain %q\ngot: %s", tt.step, tt.wantPct, out)
			}
		})
	}
}

func TestRenderContainsStepCounter(t *testing.T) {
	p := NewContextPanel()
	p.SetStep(3)
	out := p.Render()
	if !strings.Contains(out, "3 of 7") {
		t.Errorf("Render() output does not contain '3 of 7':\n%s", out)
	}
}

func TestRenderContainsConfigValues(t *testing.T) {
	p := NewContextPanel()
	p.SetConfig("oras", "argo", "sigstore", "")
	out := p.Render()
	for _, want := range []string{"oras", "argo", "sigstore"} {
		if !strings.Contains(out, want) {
			t.Errorf("Render() output does not contain %q:\n%s", want, out)
		}
	}
}

func TestRenderContainsModelInfo(t *testing.T) {
	p := NewContextPanel()
	p.SetModelInfo("phi-4-mini", "/models/phi", "my-org/phi:v1")
	out := p.Render()
	for _, want := range []string{"phi-4-mini", "my-org/phi:v1"} {
		if !strings.Contains(out, want) {
			t.Errorf("Render() output does not contain %q:\n%s", want, out)
		}
	}
}

func TestRenderSetResultsPackage(t *testing.T) {
	p := NewContextPanel()
	p.SetResults(true, false, false, false)
	out := p.Render()
	if !strings.Contains(out, "Package") {
		t.Errorf("Render() output does not contain 'Package' after SetResults(true,...):\n%s", out)
	}
}

func TestRenderAddLog(t *testing.T) {
	p := NewContextPanel()
	p.AddLog("artifact pushed successfully")
	out := p.Render()
	if !strings.Contains(out, "artifact pushed") {
		t.Errorf("Render() output does not contain log message:\n%s", out)
	}
}

func TestLogTruncation(t *testing.T) {
	p := NewContextPanel()
	for i := 0; i < 15; i++ {
		p.AddLog("log entry")
	}
	if len(p.Logs) > 10 {
		t.Errorf("Logs length = %d, want <= 10", len(p.Logs))
	}
}

// Tests for the interactive ContextModel (TUI)
func TestContextModelHandleKey(t *testing.T) {
	model := NewContextModel()

	// Test tab switching
	model.HandleKey("tab")
	if model.ActiveTab != TabConfig {
		t.Errorf("Expected ActiveTab to be TabConfig after 'tab', got %d", model.ActiveTab)
	}

	model.HandleKey("shift+tab")
	if model.ActiveTab != TabProgress {
		t.Errorf("Expected ActiveTab to be TabProgress after 'shift+tab', got %d", model.ActiveTab)
	}

	// Test number keys
	model.HandleKey("2")
	if model.ActiveTab != TabConfig {
		t.Errorf("Expected ActiveTab to be TabConfig after '2', got %d", model.ActiveTab)
	}

	model.HandleKey("1")
	if model.ActiveTab != TabProgress {
		t.Errorf("Expected ActiveTab to be TabProgress after '1', got %d", model.ActiveTab)
	}

	model.HandleKey("5")
	if model.ActiveTab != TabEnv {
		t.Errorf("Expected ActiveTab to be TabEnv after '5', got %d", model.ActiveTab)
	}

	model.HandleKey("6")
	if model.ActiveTab != TabHelp {
		t.Errorf("Expected ActiveTab to be TabHelp after '6', got %d", model.ActiveTab)
	}

	// Test Enter key sets Done flag
	model.HandleKey("enter")
	if !model.Done {
		t.Error("Expected Done to be true after 'enter', got false")
	}
	if model.Cancelled {
		t.Error("Expected Cancelled to be false after 'enter', got true")
	}

	// Reset and test Esc key
	model.ResetControlFlags()
	model.HandleKey("esc")
	if !model.Done {
		t.Error("Expected Done to be true after 'esc', got false")
	}
	if !model.Cancelled {
		t.Error("Expected Cancelled to be true after 'esc', got false")
	}

	// Reset and test ctrl+c
	model.ResetControlFlags()
	model.HandleKey("ctrl+c")
	if !model.Done {
		t.Error("Expected Done to be true after 'ctrl+c', got false")
	}
	if !model.Cancelled {
		t.Error("Expected Cancelled to be true after 'ctrl+c', got false")
	}

	// Test IsDone and IsCancelled
	if !model.IsDone() {
		t.Error("Expected IsDone() to be true")
	}
	if !model.IsCancelled() {
		t.Error("Expected IsCancelled() to be true")
	}
}

func TestContextModelProgressMatchesWizardJourney(t *testing.T) {
	model := NewContextModel()
	output := model.renderProgressTab()

	for _, want := range []string{"Step 1 of 8", "→ Model Details", "· Package", "· Harden", "· Publish & Discovery"} {
		if !strings.Contains(output, want) {
			t.Errorf("progress tab does not contain %q:\n%s", want, output)
		}
	}
	if strings.Contains(output, "K8s Setup") {
		t.Errorf("progress tab contains removed K8s setup stage:\n%s", output)
	}
}

func TestContextModelProgressShowsSkippedStages(t *testing.T) {
	model := NewContextModel()
	model.SetSkippedSteps(true, true)
	model.SetStep(7)
	output := model.renderProgressTab()

	for _, want := range []string{"- Sign", "- Verify", "→ Publish & Discovery", "- GitOps Promotion"} {
		if !strings.Contains(output, want) {
			t.Errorf("progress tab does not contain %q:\n%s", want, output)
		}
	}
}
