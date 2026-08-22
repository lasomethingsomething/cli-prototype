package cmd

import (
	"strings"
	"testing"
)

func TestBuildWizardSummary(t *testing.T) {
	tests := []struct {
		name             string
		packageSucceeded bool
		signSucceeded    bool
		verifySucceeded  bool
		skipSigning      bool
		hasKubernetes    bool
		skipDeploy       bool
		wantContains     []string
		wantNotContains  []string
	}{
		{
			name:             "all succeeded",
			packageSucceeded: true, signSucceeded: true, verifySucceeded: true,
			hasKubernetes: true,
			wantContains:  []string{"packaged", "signed with", "verified signature", "deployed to Kubernetes"},
		},
		{
			name:             "signing tool not installed",
			packageSucceeded: true, signSucceeded: false, verifySucceeded: false,
			wantContains:    []string{"skipped signing", "skipped verification"},
			wantNotContains: []string{"signed with"},
		},
		{
			name:             "skip signing flag",
			packageSucceeded: true, skipSigning: true,
			wantContains:    []string{"skipped signing"},
			wantNotContains: []string{"signed with", "verified"},
		},
		{
			name:             "no kubernetes",
			packageSucceeded: true, signSucceeded: true, verifySucceeded: true,
			hasKubernetes: false,
			wantContains:  []string{"skipped deployment"},
		},
		{
			name:             "packaging failed",
			packageSucceeded: false,
			wantContains:    []string{"skipped packaging"},
			wantNotContains: []string{"packaged '"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lines := buildWizardSummary(
				tt.packageSucceeded, tt.signSucceeded, tt.verifySucceeded,
				tt.skipSigning, tt.hasKubernetes, tt.skipDeploy,
				"test-model", "sigstore", "argo",
			)
			joined := strings.Join(lines, "\n")
			for _, want := range tt.wantContains {
				if !strings.Contains(joined, want) {
					t.Errorf("summary missing %q\ngot:\n%s", want, joined)
				}
			}
			for _, notWant := range tt.wantNotContains {
				if strings.Contains(joined, notWant) {
					t.Errorf("summary should not contain %q\ngot:\n%s", notWant, joined)
				}
			}
		})
	}
}
