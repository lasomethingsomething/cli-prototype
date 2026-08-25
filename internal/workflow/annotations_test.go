package workflow

import (
	"strings"
	"testing"
)

// Test NewAnnotationSet creates default values
func TestNewAnnotationSet(t *testing.T) {
	annotations := NewAnnotationSet()

	if annotations.ProfileVersion != "1.0.0" {
		t.Errorf("ProfileVersion = %q, want %q", annotations.ProfileVersion, "1.0.0")
	}
	if annotations.ArtifactType != "model" {
		t.Errorf("ArtifactType = %q, want %q", annotations.ArtifactType, "model")
	}
	if annotations.MOFVersion != "1.0" {
		t.Errorf("MOFVersion = %q, want %q", annotations.MOFVersion, "1.0")
	}
	if annotations.SigningFramework != "sigstore-cosign" {
		t.Errorf("SigningFramework = %q, want %q", annotations.SigningFramework, "sigstore-cosign")
	}
	if annotations.SBOMFormat != "spdx-json" {
		t.Errorf("SBOMFormat = %q, want %q", annotations.SBOMFormat, "spdx-json")
	}
	if annotations.Runtime != "vllm" {
		t.Errorf("Runtime = %q, want %q", annotations.Runtime, "vllm")
	}
	if annotations.Accelerator != "nvidia-gpu" {
		t.Errorf("Accelerator = %q, want %q", annotations.Accelerator, "nvidia-gpu")
	}
	if annotations.CUDAMin != "12.1" {
		t.Errorf("CUDAMin = %q, want %q", annotations.CUDAMin, "12.1")
	}
}

// Test AnnotationSet ToMap conversion
func TestAnnotationSetToMap(t *testing.T) {
	annotations := NewAnnotationSet()
	annotations.MOFClass = "I"
	annotations.MOFComponents = "weights,training-data"

	m := annotations.ToMap()

	// Check all expected keys are present
	expectedKeys := []string{
		AnnotationProfileVersion,
		AnnotationArtifactType,
		AnnotationMOFClass,
		AnnotationMOFVersion,
		AnnotationMOFComponents,
		AnnotationSigningFramework,
		AnnotationSBOMFormat,
		AnnotationProvenanceType,
		AnnotationPackagingFormat,
		AnnotationRuntime,
		AnnotationAccelerator,
		AnnotationCUDAVersionMin,
	}

	for _, key := range expectedKeys {
		if _, ok := m[key]; !ok {
			t.Errorf("ToMap() missing key: %s", key)
		}
	}

	// Check specific values
	if m[AnnotationProfileVersion] != "1.0.0" {
		t.Errorf("ProfileVersion in map = %q, want %q", m[AnnotationProfileVersion], "1.0.0")
	}
	if m[AnnotationMOFClass] != "I" {
		t.Errorf("MOFClass in map = %q, want %q", m[AnnotationMOFClass], "I")
	}
}

// Test AnnotationSet ToMap with empty values
func TestAnnotationSetToMapEmptyValues(t *testing.T) {
	annotations := &AnnotationSet{}
	m := annotations.ToMap()

	// Empty values should not be in the map
	if len(m) != 0 {
		t.Errorf("ToMap() with empty AnnotationSet should be empty, got %d entries", len(m))
	}

	// Set only one field
	annotations.Runtime = "vllm"
	m = annotations.ToMap()

	if len(m) != 1 {
		t.Errorf("ToMap() with one field should have 1 entry, got %d", len(m))
	}
	if m[AnnotationRuntime] != "vllm" {
		t.Errorf("Runtime in map = %q, want %q", m[AnnotationRuntime], "vllm")
	}
}

// Test annotation constants are correct
func TestAnnotationConstants(t *testing.T) {
	// Test that constants start with the correct prefix
	constants := []string{
		AnnotationProfileVersion,
		AnnotationArtifactType,
		AnnotationMOFClass,
		AnnotationMOFVersion,
		AnnotationMOFComponents,
		AnnotationSigningFramework,
		AnnotationSBOMFormat,
		AnnotationProvenanceType,
		AnnotationPackagingFormat,
		AnnotationRuntime,
		AnnotationAccelerator,
		AnnotationCUDAVersionMin,
		AnnotationMemoryMin,
	}

	for _, c := range constants {
		if !strings.HasPrefix(c, "org.cncf.ai.") {
			t.Errorf("Annotation constant %q does not start with 'org.cncf.ai.'", c)
		}
	}
}

// Test custom annotation values
func TestAnnotationSetCustomValues(t *testing.T) {
	annotations := &AnnotationSet{
		ProfileVersion:   "2.0.0",
		ArtifactType:     "skill",
		Runtime:          "kserve",
		Accelerator:      "amd-gpu",
		CUDAMin:          "11.8",
		MemoryMin:        "32GiB",
		MOFClass:         "II",
		MOFComponents:    "weights,code",
		SigningFramework: "notation",
		SBOMFormat:       "cyclonedx",
	}

	m := annotations.ToMap()

	if m[AnnotationProfileVersion] != "2.0.0" {
		t.Errorf("ProfileVersion = %q, want %q", m[AnnotationProfileVersion], "2.0.0")
	}
	if m[AnnotationArtifactType] != "skill" {
		t.Errorf("ArtifactType = %q, want %q", m[AnnotationArtifactType], "skill")
	}
	if m[AnnotationRuntime] != "kserve" {
		t.Errorf("Runtime = %q, want %q", m[AnnotationRuntime], "kserve")
	}
	if m[AnnotationAccelerator] != "amd-gpu" {
		t.Errorf("Accelerator = %q, want %q", m[AnnotationAccelerator], "amd-gpu")
	}
	if m[AnnotationMOFClass] != "II" {
		t.Errorf("MOFClass = %q, want %q", m[AnnotationMOFClass], "II")
	}
}
