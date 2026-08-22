package tui

import (
	"fmt"
	"strings"
	"testing"
)

func TestProgressPercent(t *testing.T) {
	tests := []struct {
		step    int
		total   int
		wantPct int
	}{
		{1, 7, 14},
		{4, 7, 57},
		{7, 7, 100},
		{0, 7, 0},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("step%dof%d", tt.step, tt.total), func(t *testing.T) {
			p := NewContextPanel()
			p.TotalSteps = tt.total
			p.SetStep(tt.step)
			got := (p.CurrentStep * 100) / p.TotalSteps
			if got != tt.wantPct {
				t.Errorf("progress percent = %d, want %d", got, tt.wantPct)
			}
		})
	}
}

func TestRenderContainsStepCounter(t *testing.T) {
	p := NewContextPanel()
	p.SetStep(3)
	out := p.Render()
	if !strings.Contains(out, "3") {
		t.Errorf("Render() output does not contain step number 3:\n%s", out)
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
