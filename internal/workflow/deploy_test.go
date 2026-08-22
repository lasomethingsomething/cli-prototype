package workflow

import (
	"strings"
	"testing"
)

// Test NewDeployWorkflow
func TestNewDeployWorkflow(t *testing.T) {
	// Test with valid gitops and registry
	wf, err := NewDeployWorkflow("argo", "oras")
	if err != nil {
		t.Fatalf("NewDeployWorkflow(\"argo\", \"oras\") error = %v", err)
	}
	if wf == nil {
		t.Fatal("NewDeployWorkflow(\"argo\", \"oras\") returned nil")
	}
	if wf.gitOps != "argo" {
		t.Errorf("gitOps = %q, want %q", wf.gitOps, "argo")
	}
	if wf.registry != "oras" {
		t.Errorf("registry = %q, want %q", wf.registry, "oras")
	}

	// Test with invalid gitops
	_, err = NewDeployWorkflow("invalid-gitops", "oras")
	if err == nil {
		t.Error("NewDeployWorkflow(\"invalid-gitops\", \"oras\") expected error, got nil")
	}

	// Test with invalid registry
	_, err = NewDeployWorkflow("argo", "invalid-registry")
	if err == nil {
		t.Error("NewDeployWorkflow(\"argo\", \"invalid-registry\") expected error, got nil")
	}
}

// Test SetModelInfo
func TestSetModelInfo(t *testing.T) {
	wf, err := NewDeployWorkflow("argo", "oras")
	if err != nil {
		t.Fatalf("NewDeployWorkflow error: %v", err)
	}

	wf.SetModelInfo("phi-4-mini", "https://github.com/me/manifests", "/manifests")

	if wf.modelName != "phi-4-mini" {
		t.Errorf("modelName = %q, want %q", wf.modelName, "phi-4-mini")
	}
	if wf.repoURL != "https://github.com/me/manifests" {
		t.Errorf("repoURL = %q, want %q", wf.repoURL, "https://github.com/me/manifests")
	}
	if wf.manifestPath != "/manifests" {
		t.Errorf("manifestPath = %q, want %q", wf.manifestPath, "/manifests")
	}
}

// Test DeployWorkflow Run with missing tools
func TestDeployWorkflowMissingTools(t *testing.T) {
	tests := []struct {
		name        string
		gitOps      string
		registry   string
		expectError bool
		errorSubstr string
	}{
		{
			name:        "flux not installed",
			gitOps:      "flux",
			registry:   "modelpack",
			expectError: true,
			errorSubstr: "flux not installed",
		},
		{
			name:        "argo not installed",
			gitOps:      "argo",
			registry:   "modelpack",
			expectError: true,
			errorSubstr: "argocd not installed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wf, err := NewDeployWorkflow(tt.gitOps, tt.registry)
			if err != nil {
				t.Fatalf("NewDeployWorkflow error = %v", err)
			}

			wf.SetModelInfo("test-model", "https://github.com/test/repo", "./manifests")
			err = wf.Run()
			if (err != nil) != tt.expectError {
				t.Errorf("Run() error = %v, expectError %v", err, tt.expectError)
				return
			}

			if tt.expectError && !strings.Contains(err.Error(), tt.errorSubstr) {
				t.Errorf("Run() error = %v, expected to contain %q", err, tt.errorSubstr)
			}
		})
	}
}

// Test DeployWorkflow Run with model info
func TestDeployWorkflowWithModelInfo(t *testing.T) {
	// Using modelpack for registry (which doesn't require external tools to be installed)
	// and flux for gitops (which will fail the check, but we're testing SetModelInfo)
	wf, err := NewDeployWorkflow("flux", "modelpack")
	if err != nil {
		t.Fatalf("NewDeployWorkflow error: %v", err)
	}

	wf.SetModelInfo("test-model", "https://github.com/test/repo", "./manifests")

	// Verify model info was set correctly
	if wf.modelName != "test-model" {
		t.Errorf("modelName = %q, want %q", wf.modelName, "test-model")
	}
	if wf.repoURL != "https://github.com/test/repo" {
		t.Errorf("repoURL = %q, want %q", wf.repoURL, "https://github.com/test/repo")
	}
	if wf.manifestPath != "./manifests" {
		t.Errorf("manifestPath = %q, want %q", wf.manifestPath, "./manifests")
	}

	// Run will fail due to flux not being installed, but that's expected
	// We're just testing that SetModelInfo works and the data is stored
	err = wf.Run()
	if err == nil {
		t.Error("Expected error from flux not being installed")
	}
	if !strings.Contains(err.Error(), "flux not installed") {
		t.Errorf("Expected flux not installed error, got: %v", err)
	}
}

func TestDeployWorkflowRunWithoutSetModelInfo(t *testing.T) {
	wf, err := NewDeployWorkflow("flux", "oras")
	if err != nil {
		t.Fatalf("NewDeployWorkflow error: %v", err)
	}
	err = wf.Run()
	if err == nil {
		t.Fatal("Run() without SetModelInfo expected error, got nil")
	}
	if !strings.Contains(err.Error(), "model info not set") {
		t.Errorf("Run() error = %q, expected to contain \"model info not set\"", err.Error())
	}
}
