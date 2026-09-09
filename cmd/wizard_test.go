package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lasomethingsomething/cli-prototype/internal/workflow"
)

func TestExpandHomePath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("UserHomeDir() error = %v", err)
	}

	tests := []struct {
		name string
		path string
		want string
	}{
		{name: "home", path: "~", want: home},
		{name: "home child", path: "~/test-model", want: filepath.Join(home, "test-model")},
		{name: "relative", path: "./models/test", want: "./models/test"},
		{name: "absolute", path: "/tmp/test-model", want: "/tmp/test-model"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := expandHomePath(test.path)
			if err != nil {
				t.Fatalf("expandHomePath(%q) error = %v", test.path, err)
			}
			if got != test.want {
				t.Errorf("expandHomePath(%q) = %q, want %q", test.path, got, test.want)
			}
		})
	}
}

func TestBuildSummaryLines_AllSucceeded(t *testing.T) {
	r := wizardResult{
		packageSucceeded:   true,
		checkSucceeded:     true,
		signSucceeded:      true,
		verifySucceeded:    true,
		publishSucceeded:   true,
		publishDestination: "ghcr.io/my-org",
		deploySucceeded:    true,
		modelName:          "phi-4-mini",
		signer:             "sigstore",
		gitOps:             "argo",
	}
	lines := buildSummaryLines(r)
	checks := []string{"✓ Packaged", "✓ Compliance check passed", "✓ Signed with sigstore", "✓ Verified", "✓ Published to ghcr.io/my-org", "✓ Deployed"}
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

func TestWizardCompletionMessage(t *testing.T) {
	if got := wizardCompletionMessage(wizardResult{}); !strings.Contains(got, "local artifact") {
		t.Errorf("local completion message = %q", got)
	}
	if got := wizardCompletionMessage(wizardResult{deploySucceeded: true}); !strings.Contains(got, "ready for production") {
		t.Errorf("deployed completion message = %q", got)
	}
	if got := wizardCompletionMessage(wizardResult{publishSucceeded: true}); !strings.Contains(got, "published") {
		t.Errorf("published completion message = %q", got)
	}
}

func TestWizardManifestInstructions(t *testing.T) {
	result := wizardResult{artifactName: "test:v1", publishSucceeded: true, publishDestination: "localhost:5000"}
	instructions := wizardManifestInstructions("/tmp/test-model", result)
	for _, want := range []string{"cat /tmp/test-model/manifest.json", "oras manifest fetch localhost:5000/test:v1"} {
		found := false
		for _, instruction := range instructions {
			if strings.Contains(instruction, want) {
				found = true
			}
		}
		if !found {
			t.Errorf("instructions = %v, want %q", instructions, want)
		}
	}
}

func TestSummarizeManifest(t *testing.T) {
	summary := summarizeManifest(map[string]string{
		workflow.AnnotationArtifactType:    "model",
		workflow.AnnotationPackagingFormat: "oci",
		workflow.AnnotationSBOMFormat:      "spdx-json",
		workflow.AnnotationMOFClass:        "III",
		workflow.AnnotationRuntime:         "vllm",
	})
	for _, want := range []string{"model artifact metadata", "oci packaging", "spdx-json SBOM metadata", "MOF Class III", "vllm runtime requirements"} {
		if !strings.Contains(summary, want) {
			t.Errorf("summary = %q, want %q", summary, want)
		}
	}
}
