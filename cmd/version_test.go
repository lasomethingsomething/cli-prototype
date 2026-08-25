package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestGetVersion(t *testing.T) {
	origCommit, origDate := Commit, Date
	t.Cleanup(func() { Commit, Date = origCommit, origDate })

	Commit, Date = "", ""
	if got := GetVersion(); got != Version {
		t.Errorf("GetVersion() without build info = %q, want %q", got, Version)
	}

	Commit, Date = "abc1234", "2026-08-25T00:00:00Z"
	got := GetVersion()
	for _, want := range []string{Version, "abc1234", "2026-08-25T00:00:00Z"} {
		if !strings.Contains(got, want) {
			t.Errorf("GetVersion() = %q, want it to contain %q", got, want)
		}
	}
}

func TestVersionCommand(t *testing.T) {
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetArgs([]string{"version"})
	t.Cleanup(func() { rootCmd.SetOut(nil); rootCmd.SetArgs(nil) })

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("version command failed: %v", err)
	}
	if !strings.HasPrefix(out.String(), "model-cli "+Version) {
		t.Errorf("output = %q, want prefix %q", out.String(), "model-cli "+Version)
	}
}
