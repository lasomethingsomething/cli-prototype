package workflow

import (
	"encoding/json"
	"strings"
	"testing"
)

func fullAnnotations() map[string]string {
	a := NewAnnotationSet()
	a.MOFClass = "II"
	return a.ToMap()
}

func TestEvaluateArtifactPassesWithFullAnnotations(t *testing.T) {
	r := EvaluateArtifact("gitops", "m:v1", fullAnnotations(), Policy{Environment: "production"})
	if !r.Passed || r.Status != "pass" || len(r.Missing) != 0 {
		t.Fatalf("report = %+v, want pass with nothing missing", r)
	}
	if !strings.HasPrefix(r.Summary(), "PASS:") {
		t.Errorf("Summary() = %q", r.Summary())
	}
}

func TestEvaluateArtifactFailsOnMissingTrustAndInfra(t *testing.T) {
	ann := fullAnnotations()
	delete(ann, AnnotationSigningFramework)
	delete(ann, AnnotationRuntime)
	delete(ann, AnnotationCUDAVersionMin) // conditional: warning only

	r := EvaluateArtifact("gitops", "m:v1", ann, Policy{})
	if r.Passed {
		t.Fatal("expected failure")
	}
	if len(r.Missing) != 2 || r.Missing[0] != AnnotationSigningFramework || r.Missing[1] != AnnotationRuntime {
		t.Errorf("Missing = %v, want the two required keys in order", r.Missing)
	}
	if len(r.Warnings) != 1 || !strings.Contains(r.Warnings[0], AnnotationCUDAVersionMin) {
		t.Errorf("Warnings = %v, want one about the conditional annotation", r.Warnings)
	}
	if !strings.Contains(r.Summary(), AnnotationSigningFramework) {
		t.Errorf("Summary() = %q, want the missing keys listed", r.Summary())
	}
}

func TestEvaluateEnvironmentPolicyHybridCloud(t *testing.T) {
	ann := fullAnnotations()
	ann[AnnotationDataResidency] = "eu-west-1"

	t.Run("matching region passes", func(t *testing.T) {
		r := EvaluateArtifact("admission", "m:v1", ann, Policy{Environment: "hybrid-cloud", Region: "eu-west-1"})
		if !r.Passed {
			t.Errorf("report failed: %v", r.Missing)
		}
	})
	t.Run("mismatch warns by default", func(t *testing.T) {
		r := EvaluateArtifact("admission", "m:v1", ann, Policy{Environment: "hybrid-cloud", Region: "us-east-1"})
		if !r.Passed || len(r.Warnings) == 0 {
			t.Errorf("want pass with a warning, got passed=%v warnings=%v", r.Passed, r.Warnings)
		}
	})
	t.Run("mismatch fails in strict mode", func(t *testing.T) {
		r := EvaluateArtifact("admission", "m:v1", ann, Policy{Environment: "hybrid-cloud", Region: "us-east-1", Strict: true})
		if r.Passed || len(r.Missing) != 1 || r.Missing[0] != AnnotationDataResidency+"=us-east-1" {
			t.Errorf("want failure naming the residency requirement, got passed=%v missing=%v", r.Passed, r.Missing)
		}
	})
	t.Run("no region warns", func(t *testing.T) {
		r := EvaluateArtifact("admission", "m:v1", ann, Policy{Environment: "hybrid-cloud"})
		if !r.Passed || len(r.Warnings) == 0 {
			t.Errorf("want pass with a region warning, got passed=%v warnings=%v", r.Passed, r.Warnings)
		}
	})
}

func TestEvaluateEnvironmentPolicyAirGapped(t *testing.T) {
	ann := fullAnnotations()
	ann[AnnotationPackagingFormat] = "oras"
	r := EvaluateArtifact("gitops", "m:v1", ann, Policy{Environment: "air-gapped"})
	if !r.Passed {
		t.Errorf("air-gapped checks are advisory, got failure: %v", r.Missing)
	}
	found := false
	for _, w := range r.Warnings {
		if strings.Contains(w, "modelpack") {
			found = true
		}
	}
	if !found {
		t.Errorf("want a warning suggesting modelpack, got %v", r.Warnings)
	}
}

func TestEvaluateEnvironmentPolicyUnknownEnvironmentWarns(t *testing.T) {
	r := EvaluateArtifact("gitops", "m:v1", fullAnnotations(), Policy{Environment: "moon-base"})
	if !r.Passed || len(r.Warnings) != 1 || !strings.Contains(r.Warnings[0], "moon-base") {
		t.Errorf("unknown environment should warn, got passed=%v warnings=%v", r.Passed, r.Warnings)
	}
}

func TestValidationReportJSON(t *testing.T) {
	r := EvaluateArtifact("nodes", "m:v1", map[string]string{}, Policy{})
	data, err := r.JSON()
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]interface{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	for _, key := range []string{"target", "artifact", "status", "passed", "missing", "warnings", "checks"} {
		if _, ok := decoded[key]; !ok {
			t.Errorf("JSON missing key %q", key)
		}
	}
	if decoded["status"] != "fail" {
		t.Errorf("status = %v, want fail", decoded["status"])
	}
}
