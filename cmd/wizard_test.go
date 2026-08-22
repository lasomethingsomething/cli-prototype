package cmd

import (
	"strings"
	"testing"
)

func TestBuildSummaryLines_AllSucceeded(t *testing.T) {
	r := wizardResult{
		packageSucceeded: true,
		signSucceeded:    true,
		verifySucceeded:  true,
		deploySucceeded:  true,
		modelName:        "phi-4-mini",
		signer:           "sigstore",
		gitOps:           "argo",
	}
	lines := buildSummaryLines(r)
	checks := []string{"✓ Packaged", "✓ Signed with sigstore", "✓ Verified", "✓ Deployed"}
	for i, want := range checks {
		if i >= len(lines) {
			t.Fatalf("expected at least %d lines, got %d", i+1, len(lines))
		}
		if !strings.Contains(lines[i], want) {
			t.Errorf("line[%d] = %q, want to contain %q", i, lines[i], want)
		}
	}
}

func TestBuildSummaryLines_SignToolNotInstalled(t *testing.T) {
	r := wizardResult{
		packageSucceeded: true,
		signSucceeded:    false,
		verifySucceeded:  false,
		skipSigning:      false,
		modelName:        "phi-4-mini",
	}
	lines := buildSummaryLines(r)
	found := false
	for _, l := range lines {
		if strings.Contains(l, "⚠") && strings.Contains(l, "not installed") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected a warning about tool not installed, got: %v", lines)
	}
}

func TestBuildSummaryLines_SkipSigning(t *testing.T) {
	r := wizardResult{
		packageSucceeded: true,
		skipSigning:      true,
		modelName:        "phi-4-mini",
	}
	lines := buildSummaryLines(r)
	for _, l := range lines {
		if strings.Contains(l, "✓ Signed") || strings.Contains(l, "✓ Verified") {
			t.Errorf("expected no sign/verify success lines when skipSigning=true, got: %q", l)
		}
	}
}

func TestBuildSummaryLines_SkipDeploy(t *testing.T) {
	r := wizardResult{
		packageSucceeded: true,
		signSucceeded:    true,
		verifySucceeded:  true,
		skipDeploy:       true,
		modelName:        "phi-4-mini",
		signer:           "sigstore",
	}
	lines := buildSummaryLines(r)
	found := false
	for _, l := range lines {
		if strings.Contains(l, "⚠") && strings.Contains(l, "deployment") {
			found = true
		}
		if strings.Contains(l, "✓ Deployed") {
			t.Errorf("expected no deploy success line when skipDeploy=true, got: %q", l)
		}
	}
	if !found {
		t.Errorf("expected a skipped-deployment warning, got: %v", lines)
	}
}
