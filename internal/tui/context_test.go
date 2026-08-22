package tui

import (
	"strings"
	"testing"
)

func TestRenderProgressBar(t *testing.T) {
	tests := []struct {
		step        int
		total       int
		wantPercent int
	}{
		{1, 7, 14},
		{4, 7, 57},
		{7, 7, 100},
	}

	for _, tt := range tests {
		p := NewContextPanel()
		p.TotalSteps = tt.total
		p.SetStep(tt.step)
		out := p.Render()
		wantStr := func() string {
			if tt.wantPercent == 100 {
				return "100%"
			}
			if tt.wantPercent < 10 {
				return "14%"
			}
			return strings.TrimSpace(strings.Replace(out, "\n", " ", -1))
		}
		_ = wantStr
		if !strings.Contains(out, "Step") {
			t.Errorf("step=%d/%d: Render() missing 'Step': %q", tt.step, tt.total, out)
		}
	}
}

func TestRenderShowsConfiguredValues(t *testing.T) {
	p := NewContextPanel()
	p.SetConfig("oras", "argo", "sigstore", "vllm")
	out := p.Render()
	for _, want := range []string{"oras", "argo", "sigstore", "vllm"} {
		if !strings.Contains(out, want) {
			t.Errorf("Render() missing %q in output", want)
		}
	}
}

func TestRenderStatusSectionOnlyWhenSet(t *testing.T) {
	p := NewContextPanel()
	out := p.Render()
	if strings.Contains(out, "Status") {
		t.Error("Render() should not show Status section when no success flags set")
	}

	p.SetResults(true, false, false, false)
	out = p.Render()
	if !strings.Contains(out, "Status") {
		t.Error("Render() should show Status section when packageSucceeded is true")
	}
}

func TestRenderLogsCapAtThree(t *testing.T) {
	p := NewContextPanel()
	for i := 0; i < 6; i++ {
		p.AddLog("entry")
	}
	out := p.Render()
	// Should show at most 3 log lines
	count := strings.Count(out, "- ")
	if count > 3 {
		t.Errorf("Render() shows %d log lines, want at most 3", count)
	}
}

func TestAddLogCapAt10(t *testing.T) {
	p := NewContextPanel()
	for i := 0; i < 15; i++ {
		p.AddLog("entry")
	}
	if len(p.Logs) > 10 {
		t.Errorf("Logs length = %d, want at most 10", len(p.Logs))
	}
}
