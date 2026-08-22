package workflow

import (
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
