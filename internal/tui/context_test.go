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
