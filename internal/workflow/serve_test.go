package workflow

import (
	"testing"
)

// Test NewServeWorkflow
func TestNewServeWorkflow(t *testing.T) {
	// Test with valid runtime
	wf, err := NewServeWorkflow("vllm")
	if err != nil {
		t.Fatalf("NewServeWorkflow(\"vllm\") error = %v", err)
	}
	if wf == nil {
		t.Fatal("NewServeWorkflow(\"vllm\") returned nil")
	}
	if wf.runtime != "vllm" {
		t.Errorf("runtime = %q, want %q", wf.runtime, "vllm")
	}

	// Test with invalid runtime
	_, err = NewServeWorkflow("invalid-runtime")
	if err == nil {
		t.Error("NewServeWorkflow(\"invalid-runtime\") expected error, got nil")
	}
}

// Test SetServeInfo
func TestSetServeInfo(t *testing.T) {
	wf, err := NewServeWorkflow("vllm")
	if err != nil {
		t.Fatalf("NewServeWorkflow error: %v", err)
	}

	wf.SetServeInfo("/path/to/model", "0.0.0.0", "8080")

	if wf.modelPath != "/path/to/model" {
		t.Errorf("modelPath = %q, want %q", wf.modelPath, "/path/to/model")
	}
	if wf.host != "0.0.0.0" {
		t.Errorf("host = %q, want %q", wf.host, "0.0.0.0")
	}
	if wf.port != "8080" {
		t.Errorf("port = %q, want %q", wf.port, "8080")
	}
}

// Test SetModelInfo
func TestServeSetModelInfo(t *testing.T) {
	wf, err := NewServeWorkflow("kserve")
	if err != nil {
		t.Fatalf("NewServeWorkflow error: %v", err)
	}

	wf.SetModelInfo("70GB", []string{"skill-1:v1", "skill-2:v2"})

	if wf.modelSize != "70GB" {
		t.Errorf("modelSize = %q, want %q", wf.modelSize, "70GB")
	}
	if len(wf.skillRefs) != 2 {
		t.Errorf("skillRefs length = %d, want 2", len(wf.skillRefs))
	}
	if wf.skillRefs[0] != "skill-1:v1" {
		t.Errorf("skillRefs[0] = %q, want %q", wf.skillRefs[0], "skill-1:v1")
	}
	if wf.skillRefs[1] != "skill-2:v2" {
		t.Errorf("skillRefs[1] = %q, want %q", wf.skillRefs[1], "skill-2:v2")
	}
}

// Test ServeWorkflow default values
func TestServeWorkflowDefaults(t *testing.T) {
	wf, err := NewServeWorkflow("vllm")
	if err != nil {
		t.Fatalf("NewServeWorkflow error: %v", err)
	}

	if !wf.loadSkills {
		t.Error("loadSkills default = false, want true")
	}
}
