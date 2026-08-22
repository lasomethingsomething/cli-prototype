package tui

import (
	"strings"
	"testing"
)

func TestContextPanelDefaults(t *testing.T) {
	p := NewContextPanel()
	if p.CurrentStep != 1 {
		t.Errorf("CurrentStep = %d, want 1", p.CurrentStep)
	}
	if p.TotalSteps != 7 {
		t.Errorf("TotalSteps = %d, want 7", p.TotalSteps)
	}
}

func TestContextPanelRenderContainsModelName(t *testing.T) {
	p := NewContextPanel()
	p.SetModelInfo("phi-4-mini", "/models/phi", "my-org/phi:v1")
	out := p.Render()
	if !strings.Contains(out, "phi-4-mini") {
		t.Errorf("Render() output does not contain model name 'phi-4-mini'\ngot: %s", out)
	}
	if !strings.Contains(out, "my-org/phi:v1") {
		t.Errorf("Render() output does not contain artifact name 'my-org/phi:v1'\ngot: %s", out)
	}
}

func TestContextPanelRenderContainsConfig(t *testing.T) {
	p := NewContextPanel()
	p.SetConfig("oras", "argo", "sigstore", "vllm")
	out := p.Render()
	for _, want := range []string{"oras", "argo", "sigstore", "vllm"} {
		if !strings.Contains(out, want) {
			t.Errorf("Render() output does not contain config value %q\ngot: %s", want, out)
		}
	}
}

func TestContextPanelRenderProgressStep(t *testing.T) {
	p := NewContextPanel()
	p.SetStep(4)
	out := p.Render()
	if !strings.Contains(out, "4") {
		t.Errorf("Render() output does not contain step number '4'\ngot: %s", out)
	}
}

func TestContextPanelRenderResultFlags(t *testing.T) {
	p := NewContextPanel()
	p.SetResults(true, false, false, false)
	out := p.Render()
	if !strings.Contains(out, "Package") {
		t.Errorf("Render() output does not contain 'Package' after SetResults(true,...)\ngot: %s", out)
	}
}

func TestContextPanelAddLog(t *testing.T) {
	p := NewContextPanel()
	p.AddLog("artifact pushed successfully")
	out := p.Render()
	if !strings.Contains(out, "artifact pushed") {
		t.Errorf("Render() output does not contain log message\ngot: %s", out)
	}
}

func TestContextPanelLogTruncation(t *testing.T) {
	p := NewContextPanel()
	for i := 0; i < 15; i++ {
		p.AddLog("log entry")
	}
	if len(p.Logs) > 10 {
		t.Errorf("Logs length = %d, want <= 10", len(p.Logs))
	}
}

func TestContextPanelProgressPercentage(t *testing.T) {
	tests := []struct {
		step    int
		total   int
		wantPct int
	}{
		{1, 7, 14},
		{4, 7, 57},
		{7, 7, 100},
	}
	for _, tt := range tests {
		p := NewContextPanel()
		p.TotalSteps = tt.total
		p.SetStep(tt.step)
		got := (p.CurrentStep * 100) / p.TotalSteps
		if got != tt.wantPct {
			t.Errorf("step %d/%d: pct = %d, want %d", tt.step, tt.total, got, tt.wantPct)
		}
	}
}
