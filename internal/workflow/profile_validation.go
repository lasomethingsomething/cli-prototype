package workflow

import (
	"fmt"
	"strings"
)

// MOFComponentVocabulary is the enumerated component list the profile
// allows in org.cncf.ai.model.mof.components.
var MOFComponentVocabulary = map[string]bool{
	"datasets":                       true,
	"data-preprocessing-code":        true,
	"model-architecture":             true,
	"final-model-parameters":         true,
	"intermediate-model-parameters":  true,
	"model-metadata":                 true,
	"training-code":                  true,
	"inference-code":                 true,
	"evaluation-code":                true,
	"evaluation-data":                true,
	"evaluation-results":             true,
	"supporting-libraries-and-tools": true,
	"model-card":                     true,
	"data-card":                      true,
	"technical-report":               true,
	"research-paper":                 true,
	"sample-model-outputs":           true,
	"model-openness-config-file":     true,
}

// profileMustKeys are the v1 profile keys every conforming model manifest
// must carry. packaging.format is SHOULD (omit when not defined) and is
// validated separately when present.
var profileMustKeys = []string{
	AnnotationProfileVersion,
	AnnotationArtifactType,
	AnnotationLifecycleStatus,
	AnnotationMOFClass,
	AnnotationMOFVersion,
	AnnotationMOFComponents,
	AnnotationSigningFramework,
	AnnotationSBOMFormat,
	AnnotationProvenanceType,
}

// ProfileConformanceResult holds profile-conformance findings. Errors are
// spec violations; Warnings are tolerated oddities (extensible values,
// legacy formats) that do not break conformance.
type ProfileConformanceResult struct {
	Errors   []string
	Warnings []string
}

// Valid reports whether the manifest satisfies the profile's MUSTs.
func (r *ProfileConformanceResult) Valid() bool { return len(r.Errors) == 0 }

// ValidateProfileConformance checks manifest annotations against the CNCF AI
// Interoperability Profile (draft v0.1): MUST keys are present, values match
// the documented enums, and mof.components uses the MOF vocabulary with
// items comma-separated and no surrounding whitespace. The spec's open
// question #2 allows extensible values for signing frameworks, and
// packaging.format has a single documented value ("modelpack") plus "omit if
// not defined", so those produce warnings rather than errors so legacy
// manifests remain readable. Evidentiary claims (signature/SBOM/provenance
// resolvability, MOF card existence) are pointers, not proof: verifying the
// referenced evidence belongs to the verification tools, per the spec.
func ValidateProfileConformance(annotations map[string]string) *ProfileConformanceResult {
	result := &ProfileConformanceResult{}

	for _, key := range profileMustKeys {
		if v, ok := annotations[key]; !ok || v == "" {
			result.Errors = append(result.Errors, fmt.Sprintf("missing required annotation %s", key))
		}
	}

	if v, ok := annotations[AnnotationArtifactType]; ok && v != "" {
		if v != "model" {
			result.Errors = append(result.Errors, fmt.Sprintf(
				"%s: %q is not a valid v1 artifact type (only \"model\")", AnnotationArtifactType, v))
		}
	}

	if v, ok := annotations[AnnotationLifecycleStatus]; ok && v != "" {
		switch v {
		case LifecycleStatusExperimental, LifecycleStatusValidated,
			LifecycleStatusDeprecated, LifecycleStatusProductReady:
		default:
			result.Errors = append(result.Errors, fmt.Sprintf(
				"%s: %q is not one of experimental, validated, deprecated, product-ready",
				AnnotationLifecycleStatus, v))
		}
	}

	if v, ok := annotations[AnnotationMOFClass]; ok && v != "" {
		switch v {
		case "I", "II", "III":
		default:
			result.Errors = append(result.Errors, fmt.Sprintf(
				"%s: %q is not a MOF class (I, II, III)", AnnotationMOFClass, v))
		}
		if components, hasComponents := annotations[AnnotationMOFComponents]; !hasComponents || components == "" {
			result.Errors = append(result.Errors, fmt.Sprintf(
				"%s: class %s claimed without mof.components", AnnotationMOFClass, v))
		}
	}

	if v, ok := annotations[AnnotationMOFComponents]; ok && v != "" {
		hasConfigFile := false
		for _, comp := range strings.Split(v, ",") {
			if comp == "" {
				result.Errors = append(result.Errors, fmt.Sprintf(
					"%s: empty list item", AnnotationMOFComponents))
				continue
			}
			if comp != strings.TrimSpace(comp) {
				result.Errors = append(result.Errors, fmt.Sprintf(
					"%s: list item %q has surrounding whitespace", AnnotationMOFComponents, comp))
			}
			comp = strings.TrimSpace(comp)
			if !MOFComponentVocabulary[comp] {
				result.Errors = append(result.Errors, fmt.Sprintf(
					"%s: %q is not in the MOF component vocabulary", AnnotationMOFComponents, comp))
			}
			if comp == MOFComponentConfigFile {
				hasConfigFile = true
			}
		}
		if _, classClaimed := annotations[AnnotationMOFClass]; classClaimed && !hasConfigFile {
			result.Warnings = append(result.Warnings, fmt.Sprintf(
				"%s: model-openness-config-file is always required by the profile regardless of class",
				AnnotationMOFComponents))
		}
	}

	if v, ok := annotations[AnnotationSBOMFormat]; ok && v != "" {
		if v != "spdx" && v != "cyclonedx" {
			result.Errors = append(result.Errors, fmt.Sprintf(
				"%s: %q is neither spdx nor cyclonedx", AnnotationSBOMFormat, v))
		}
	}

	if v, ok := annotations[AnnotationProvenanceType]; ok && v != "" {
		if v != "slsa-v1.0" && v != "in-toto" {
			result.Errors = append(result.Errors, fmt.Sprintf(
				"%s: %q is neither slsa-v1.0 nor in-toto", AnnotationProvenanceType, v))
		}
	}

	if v, ok := annotations[AnnotationSigningFramework]; ok && v != "" {
		switch v {
		case "sigstore", "notation":
		default:
			// Open question #2: extensible string format allowed for
			// frameworks beyond the required enum.
			result.Warnings = append(result.Warnings, fmt.Sprintf(
				"%s: %q is outside the required enum (sigstore, notation); accepted as an extensible value",
				AnnotationSigningFramework, v))
		}
	}

	if v, ok := annotations[AnnotationPackagingFormat]; ok && v != "" {
		switch v {
		case PackagingFormatModelPack:
			// The only documented value.
		case PackagingFormatOCI:
			result.Warnings = append(result.Warnings, fmt.Sprintf(
				"%s: %q is a legacy value; the spec says omit the annotation when no packaging format is defined",
				AnnotationPackagingFormat, v))
		default:
			result.Warnings = append(result.Warnings, fmt.Sprintf(
				"%s: %q is not a documented value (modelpack)",
				AnnotationPackagingFormat, v))
		}
	}

	return result
}
