package workflow

import (
	"strings"
	"testing"
)

// TestFullWorkflowIntegration tests the complete flow from package to serve
func TestFullWorkflowIntegration(t *testing.T) {
	// Phase 1: Package
	pf, err := NewPackageWorkflow("oras")
	if err != nil {
		t.Fatalf("Failed to create package workflow: %v", err)
	}

	annotations := NewAnnotationSet()
	annotations.Runtime = "vllm"
	annotations.Accelerator = "nvidia-gpu"
	annotations.MOFClass = "I"

	pf.SetPackageInfo("phi-4-mini", "/models/phi-4-mini", "phi-4-mini:v1", "ghcr.io/my-org", false, "")
	pf.SetAnnotations(annotations)

	if pf.modelName != "phi-4-mini" {
		t.Errorf("Package workflow modelName = %q, want %q", pf.modelName, "phi-4-mini")
	}
	if pf.annotations.Runtime != "vllm" {
		t.Errorf("Package workflow runtime annotation = %q, want %q", pf.annotations.Runtime, "vllm")
	}

	// Phase 2: Registry providers
	registryProvider, err := GetRegistryProvider("oras")
	if err != nil {
		t.Fatalf("Failed to get registry provider: %v", err)
	}
	if registryProvider.Name() != "oras" {
		t.Errorf("Registry provider name = %q, want %q", registryProvider.Name(), "oras")
	}

	// Phase 3: Signing providers
	signingProvider, err := GetSigningProvider("sigstore")
	if err != nil {
		t.Fatalf("Failed to get signing provider: %v", err)
	}
	if signingProvider.Name() != "cosign" {
		t.Errorf("Signing provider name = %q, want %q", signingProvider.Name(), "cosign")
	}

	// Phase 4: Deploy
	wf, err := NewDeployWorkflow("argo", "oras")
	if err != nil {
		t.Fatalf("Failed to create deploy workflow: %v", err)
	}

	wf.SetModelInfo("phi-4-mini", "https://github.com/me/manifests", "/manifests")

	if wf.modelName != "phi-4-mini" {
		t.Errorf("Deploy workflow modelName = %q, want %q", wf.modelName, "phi-4-mini")
	}

	// Phase 5: Serve
	wf2, err := NewServeWorkflow("vllm")
	if err != nil {
		t.Fatalf("Failed to create serve workflow: %v", err)
	}

	wf2.SetServeInfo("/models/phi-4-mini", "0.0.0.0", "8080")
	wf2.SetModelInfo("14GB", []string{"skill-1:v1"})

	if wf2.modelPath != "/models/phi-4-mini" {
		t.Errorf("Serve workflow modelPath = %q, want %q", wf2.modelPath, "/models/phi-4-mini")
	}
	if wf2.modelSize != "14GB" {
		t.Errorf("Serve workflow modelSize = %q, want %q", wf2.modelSize, "14GB")
	}
	if len(wf2.skillRefs) != 1 {
		t.Errorf("Serve workflow skillRefs length = %d, want 1", len(wf2.skillRefs))
	}

	// Verify annotation set
	ann := NewAnnotationSet()
	annMap := ann.ToMap()

	if len(annMap) == 0 {
		t.Error("AnnotationSet ToMap() returned empty map")
	}

	// Verify all annotation constants have correct prefix
	if !strings.HasPrefix(AnnotationRuntime, "org.cncf.ai.") {
		t.Errorf("AnnotationRuntime = %q, should start with 'org.cncf.ai.'", AnnotationRuntime)
	}
}
