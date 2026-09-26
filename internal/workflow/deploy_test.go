package workflow

import (
	"strings"
	"testing"
)

// --- Fakes so tests don't depend on the developer's PATH ---

// fakeGitOpsProvider lets tests control IsInstalled directly.
type fakeGitOpsProvider struct {
	name      string
	installed bool
}

func (f *fakeGitOpsProvider) Name() string                { return f.name }
func (f *fakeGitOpsProvider) IsInstalled() bool           { return f.installed }
func (f *fakeGitOpsProvider) InstallInstructions() string { return "brew install " + f.name }
func (f *fakeGitOpsProvider) Deploy(modelName, repoURL, path, modelPath string) error {
	return nil
}

// fakeDeployRegistry embeds RegistryProvider so every method is satisfied
// without external tools. Only the methods the deploy tests use are real;
// unimplemented ones panic if called (they should not be).
type fakeDeployRegistry struct {
	RegistryProvider
	installed bool
}

func (f *fakeDeployRegistry) Name() string                { return "fake-registry" }
func (f *fakeDeployRegistry) IsInstalled() bool           { return f.installed }
func (f *fakeDeployRegistry) InstallInstructions() string { return "n/a" }
func (f *fakeDeployRegistry) PackagingFormat() string     { return "oci" }
func (f *fakeDeployRegistry) Push(artifact, registry, sourcePath string, annotations map[string]string) (string, error) {
	return "sha256:fake", nil
}
func (f *fakeDeployRegistry) FetchManifestAnnotations(artifactRef string) (map[string]string, error) {
	return nil, nil
}

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
		t.Fatalf("NewDeployWorkflow error = %v", err)
	}
	wf.SetModelInfo("test-model", "./models/test", "https://github.com/test/repo", "./manifests")

	if wf.modelName != "test-model" {
		t.Errorf("modelName = %q, want %q", wf.modelName, "test-model")
	}
	if wf.modelPath != "./models/test" {
		t.Errorf("modelPath = %q, want %q", wf.modelPath, "./models/test")
	}
	if wf.repoURL != "https://github.com/test/repo" {
		t.Errorf("repoURL = %q, want %q", wf.repoURL, "https://github.com/test/repo")
	}
	if wf.manifestPath != "./manifests" {
		t.Errorf("manifestPath = %q, want %q", wf.manifestPath, "./manifests")
	}
}

// Test that missing tools produce the right errors, using injected fakes
func TestDeployWorkflowMissingTools(t *testing.T) {
	// GitOps tool missing
	wf, err := NewDeployWorkflow("flux", "modelpack")
	if err != nil {
		t.Fatalf("NewDeployWorkflow error = %v", err)
	}
	wf.gitOpsProvider = &fakeGitOpsProvider{name: "flux", installed: false}
	wf.registryProvider = &fakeDeployRegistry{installed: true}
	wf.SetModelInfo("test-model", "./models/test", "https://github.com/test/repo", "./manifests")

	err = wf.Run()
	if err == nil || !strings.Contains(err.Error(), "flux not installed") {
		t.Errorf("expected 'flux not installed' error, got: %v", err)
	}

	// Registry tool missing
	wf2, err := NewDeployWorkflow("flux", "modelpack")
	if err != nil {
		t.Fatalf("NewDeployWorkflow error = %v", err)
	}
	wf2.gitOpsProvider = &fakeGitOpsProvider{name: "flux", installed: true}
	wf2.registryProvider = &fakeDeployRegistry{installed: false}
	wf2.SetModelInfo("test-model", "./models/test", "https://github.com/test/repo", "./manifests")

	err = wf2.Run()
	if err == nil || !strings.Contains(err.Error(), "fake-registry not installed") {
		t.Errorf("expected registry error, got: %v", err)
	}
}

// Test DeployWorkflow Run with model info
func TestDeployWorkflowWithModelInfo(t *testing.T) {
	wf, err := NewDeployWorkflow("flux", "modelpack")
	if err != nil {
		t.Fatalf("NewDeployWorkflow error: %v", err)
	}
	wf.gitOpsProvider = &fakeGitOpsProvider{name: "flux", installed: false}
	wf.registryProvider = &fakeDeployRegistry{installed: true}
	wf.SetModelInfo("test-model", "./models/test", "https://github.com/test/repo", "./manifests")

	if wf.modelName != "test-model" {
		t.Errorf("modelName = %q, want %q", wf.modelName, "test-model")
	}
	if wf.modelPath != "./models/test" {
		t.Errorf("modelPath = %q, want %q", wf.modelPath, "./models/test")
	}
	if wf.repoURL != "https://github.com/test/repo" {
		t.Errorf("repoURL = %q, want %q", wf.repoURL, "https://github.com/test/repo")
	}
	if wf.manifestPath != "./manifests" {
		t.Errorf("manifestPath = %q, want %q", wf.manifestPath, "./manifests")
	}

	// Run fails on the missing (fake) flux tool
	err = wf.Run()
	if err == nil || !strings.Contains(err.Error(), "flux not installed") {
		t.Errorf("Expected flux not installed error, got: %v", err)
	}
}
