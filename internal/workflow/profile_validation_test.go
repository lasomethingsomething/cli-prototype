package workflow

import (
	"strings"
	"testing"
)

func conformantAnnotations() map[string]string {
	return map[string]string{
		AnnotationProfileVersion:   "1.0.0",
		AnnotationArtifactType:     "model",
		AnnotationLifecycleStatus:  LifecycleStatusExperimental,
		AnnotationMOFClass:         "I",
		AnnotationMOFVersion:       "1.0",
		AnnotationMOFComponents:    MOFComponentsClassI,
		AnnotationSigningFramework: "sigstore",
		AnnotationSBOMFormat:       "spdx",
		AnnotationProvenanceType:   "slsa-v1.0",
	}
}

func TestValidateProfileConformanceValid(t *testing.T) {
	result := ValidateProfileConformance(conformantAnnotations())
	if !result.Valid() {
		t.Errorf("expected conformant annotations to pass, got errors: %v", result.Errors)
	}
}

func TestValidateProfileConformanceMissingMustKeys(t *testing.T) {
	annotations := conformantAnnotations()
	delete(annotations, AnnotationLifecycleStatus)
	delete(annotations, AnnotationMOFClass)
	delete(annotations, AnnotationMOFComponents)

	result := ValidateProfileConformance(annotations)
	if result.Valid() {
		t.Fatal("expected missing MUST keys to fail")
	}
	joined := ""
	for _, e := range result.Errors {
		joined += e + "\n"
	}
	for _, key := range []string{AnnotationLifecycleStatus, AnnotationMOFClass, AnnotationMOFComponents} {
		if !strings.Contains(joined, key) {
			t.Errorf("expected error to mention %s, got: %v", key, result.Errors)
		}
	}
}

func TestValidateProfileConformanceEnumViolations(t *testing.T) {
	annotations := conformantAnnotations()
	annotations[AnnotationArtifactType] = "pipeline"
	annotations[AnnotationLifecycleStatus] = "beta"
	annotations[AnnotationMOFClass] = "IV"
	annotations[AnnotationSBOMFormat] = "spdx-json"
	annotations[AnnotationProvenanceType] = "slsa"

	result := ValidateProfileConformance(annotations)
	if result.Valid() {
		t.Fatal("expected enum violations to fail")
	}
	if len(result.Errors) != 5 {
		t.Errorf("expected 5 enum errors, got %d: %v", len(result.Errors), result.Errors)
	}
}

func TestValidateProfileConformanceMOFVocabulary(t *testing.T) {
	annotations := conformantAnnotations()
	annotations[AnnotationMOFComponents] = "weights, training-data,code"

	result := ValidateProfileConformance(annotations)
	joined := ""
	for _, e := range result.Errors {
		joined += e + "\n"
	}
	if !strings.Contains(joined, "not in the MOF component vocabulary") {
		t.Errorf("expected out-of-vocabulary component to fail, got: %v", result.Errors)
	}
	if !strings.Contains(joined, "surrounding whitespace") {
		t.Errorf("expected whitespace violation to fail, got: %v", result.Errors)
	}
}

func TestValidateProfileConformanceClassRequiresComponents(t *testing.T) {
	annotations := conformantAnnotations()
	delete(annotations, AnnotationMOFComponents)

	result := ValidateProfileConformance(annotations)
	joined := ""
	for _, e := range result.Errors {
		joined += e + "\n"
	}
	if !strings.Contains(joined, "class I claimed without mof.components") {
		t.Errorf("expected class-without-components error, got: %v", result.Errors)
	}
}

func TestValidateProfileConformanceExtensibleWarnings(t *testing.T) {
	annotations := conformantAnnotations()
	annotations[AnnotationSigningFramework] = "openpubkey"
	annotations[AnnotationPackagingFormat] = PackagingFormatOCI

	result := ValidateProfileConformance(annotations)
	if !result.Valid() {
		t.Errorf("expected extensible signing framework and legacy packaging to pass with warnings, got errors: %v", result.Errors)
	}
	joined := ""
	for _, w := range result.Warnings {
		joined += w + "\n"
	}
	if !strings.Contains(joined, "extensible value") {
		t.Errorf("expected signing framework warning, got: %v", result.Warnings)
	}
	if !strings.Contains(joined, "legacy value") {
		t.Errorf("expected packaging.format legacy warning, got: %v", result.Warnings)
	}
}

func TestValidateProfileConformanceMissingConfigFileWarning(t *testing.T) {
	annotations := conformantAnnotations()
	annotations[AnnotationMOFComponents] = "datasets,model-card"

	result := ValidateProfileConformance(annotations)
	if !result.Valid() {
		t.Errorf("expected valid vocabulary without config file to pass with warning, got errors: %v", result.Errors)
	}
	if len(result.Warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d: %v", len(result.Warnings), result.Warnings)
	}
	if !strings.Contains(result.Warnings[0], "model-openness-config-file") {
		t.Errorf("expected config file warning, got: %v", result.Warnings)
	}
}
