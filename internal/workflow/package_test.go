package workflow

import (
	"testing"
)

// Test NewPackageWorkflow
func TestNewPackageWorkflow(t *testing.T) {
	// Test with valid registry
	pf, err := NewPackageWorkflow("oras")
	if err != nil {
		t.Fatalf("NewPackageWorkflow(\"oras\") error = %v", err)
	}
	if pf == nil {
		t.Fatal("NewPackageWorkflow(\"oras\") returned nil")
	}
	if pf.registry != "oras" {
		t.Errorf("registry = %q, want %q", pf.registry, "oras")
	}

	// Test with invalid registry
	_, err = NewPackageWorkflow("invalid-registry")
	if err == nil {
		t.Error("NewPackageWorkflow(\"invalid-registry\") expected error, got nil")
	}
}

// Test SetPackageInfo
func TestSetPackageInfo(t *testing.T) {
	pf, err := NewPackageWorkflow("oras")
	if err != nil {
		t.Fatalf("NewPackageWorkflow error: %v", err)
	}

	pf.SetPackageInfo("phi-4-mini", "/path/to/model", "my-model:v1", "ghcr.io/my-org", true, "/path/to/rag")

	if pf.modelName != "phi-4-mini" {
		t.Errorf("modelName = %q, want %q", pf.modelName, "phi-4-mini")
	}
	if pf.modelPath != "/path/to/model" {
		t.Errorf("modelPath = %q, want %q", pf.modelPath, "/path/to/model")
	}
	if pf.artifactName != "my-model:v1" {
		t.Errorf("artifactName = %q, want %q", pf.artifactName, "my-model:v1")
	}
	if pf.registryURL != "ghcr.io/my-org" {
		t.Errorf("registryURL = %q, want %q", pf.registryURL, "ghcr.io/my-org")
	}
	if !pf.includeRAG {
		t.Error("includeRAG = false, want true")
	}
	if pf.ragPath != "/path/to/rag" {
		t.Errorf("ragPath = %q, want %q", pf.ragPath, "/path/to/rag")
	}
}

// Test SetAnnotations
func TestSetAnnotations(t *testing.T) {
	pf, err := NewPackageWorkflow("oras")
	if err != nil {
		t.Fatalf("NewPackageWorkflow error: %v", err)
	}

	annotations := NewAnnotationSet()
	annotations.Runtime = "vllm"
	annotations.Accelerator = "nvidia-gpu"

	pf.SetAnnotations(annotations)

	if pf.annotations == nil {
		t.Fatal("annotations is nil after SetAnnotations")
	}
	if pf.annotations.Runtime != "vllm" {
		t.Errorf("annotations.Runtime = %q, want %q", pf.annotations.Runtime, "vllm")
	}
	if pf.annotations.Accelerator != "nvidia-gpu" {
		t.Errorf("annotations.Accelerator = %q, want %q", pf.annotations.Accelerator, "nvidia-gpu")
	}
}

// Test PackageWorkflow default values
func TestPackageWorkflowDefaults(t *testing.T) {
	pf, err := NewPackageWorkflow("oras")
	if err != nil {
		t.Fatalf("NewPackageWorkflow error: %v", err)
	}

	// Check default values
	if !pf.generateSBOM {
		t.Error("generateSBOM default = false, want true")
	}
	if !pf.includeMOF {
		t.Error("includeMOF default = false, want true")
	}
	if pf.annotations == nil {
		t.Error("annotations default is nil, want non-nil")
	}
}
