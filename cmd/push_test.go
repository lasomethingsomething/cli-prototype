package cmd

import (
	"testing"

	"github.com/lasomethingsomething/cli-prototype/internal/workflow"
)

// TestMergeManifestAnnotationsPrefersManifestFile pins the precedence rule
// of issue #98: what package/harden wrote into manifest.json describes the
// model, the values push generates are placeholders that must only fill
// keys the file lacks.
func TestMergeManifestAnnotationsPrefersManifestFile(t *testing.T) {
	fromManifest := map[string]string{
		workflow.AnnotationAccelerator:   "cpu",
		workflow.AnnotationMOFClass:      "III",
		workflow.AnnotationMOFComponents: "weights",
	}
	generated := map[string]string{
		workflow.AnnotationAccelerator:   "nvidia-gpu",
		workflow.AnnotationMOFClass:      "I",
		workflow.AnnotationMOFComponents: "weights,training-data,code",
		workflow.AnnotationRuntime:       "vllm",
	}

	got := mergeManifestAnnotations(fromManifest, generated)

	want := map[string]string{
		workflow.AnnotationAccelerator:   "cpu",
		workflow.AnnotationMOFClass:      "III",
		workflow.AnnotationMOFComponents: "weights",
		workflow.AnnotationRuntime:       "vllm", // absent from the file, so the default fills it
	}
	if len(got) != len(want) {
		t.Fatalf("merged = %v, want %v", got, want)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("merged[%s] = %q, want %q", k, got[k], v)
		}
	}

	// Inputs are not mutated.
	if fromManifest[workflow.AnnotationRuntime] != "" || generated[workflow.AnnotationAccelerator] != "nvidia-gpu" {
		t.Errorf("mergeManifestAnnotations mutated its inputs: %v, %v", fromManifest, generated)
	}
}

func TestMergeManifestAnnotationsWithoutManifestFile(t *testing.T) {
	generated := map[string]string{workflow.AnnotationAccelerator: "nvidia-gpu"}
	got := mergeManifestAnnotations(nil, generated)
	if len(got) != 1 || got[workflow.AnnotationAccelerator] != "nvidia-gpu" {
		t.Errorf("merged = %v, want the generated annotations unchanged", got)
	}
}
