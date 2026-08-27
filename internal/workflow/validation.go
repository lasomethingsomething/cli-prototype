package workflow

import (
	"encoding/json"
	"fmt"
	"strings"
)

// CheckStatus is the outcome of a single validation check.
type CheckStatus string

const (
	CheckPass CheckStatus = "pass"
	CheckFail CheckStatus = "fail"
	CheckWarn CheckStatus = "warn"
	CheckInfo CheckStatus = "info"
)

// Check is one line of a validation report.
type Check struct {
	Section string      `json:"section"`
	Name    string      `json:"name"`
	Status  CheckStatus `json:"status"`
	Detail  string      `json:"detail,omitempty"`
}

// ValidationReport is the shared result type for every validation target
// (gitops pre-sync, admission, cluster nodes, runtime operators). Commands
// build one through the Evaluate* functions and render it once, as text or
// JSON, so all targets look and behave the same.
type ValidationReport struct {
	Target   string   `json:"target"`
	Artifact string   `json:"artifact"`
	Status   string   `json:"status"` // "pass" or "fail"
	Passed   bool     `json:"passed"`
	Missing  []string `json:"missing"`
	Warnings []string `json:"warnings"`
	Checks   []Check  `json:"checks"`
}

// NewValidationReport starts a passing report for artifact under target.
func NewValidationReport(target, artifact string) *ValidationReport {
	return &ValidationReport{
		Target:   target,
		Artifact: artifact,
		Status:   "pass",
		Passed:   true,
		Missing:  []string{},
		Warnings: []string{},
	}
}

func (r *ValidationReport) add(section, name string, status CheckStatus, detail string) {
	r.Checks = append(r.Checks, Check{Section: section, Name: name, Status: status, Detail: detail})
}

// Pass records a satisfied check.
func (r *ValidationReport) Pass(section, name, detail string) {
	r.add(section, name, CheckPass, detail)
}

// Info records a neutral observation.
func (r *ValidationReport) Info(section, name, detail string) {
	r.add(section, name, CheckInfo, detail)
}

// Warn records a non-blocking problem.
func (r *ValidationReport) Warn(section, name, detail string) {
	r.add(section, name, CheckWarn, detail)
	r.Warnings = append(r.Warnings, name+": "+detail)
}

// Fail records a blocking problem. missing, when non-empty, names the
// annotation or requirement that was not satisfied.
func (r *ValidationReport) Fail(section, name, detail, missing string) {
	r.add(section, name, CheckFail, detail)
	if missing != "" {
		r.Missing = append(r.Missing, missing)
	}
	r.Passed = false
	r.Status = "fail"
}

// JSON renders the report for CI consumers.
func (r *ValidationReport) JSON() ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}

// Summary is a one-line result for quiet output and error messages.
func (r *ValidationReport) Summary() string {
	if r.Passed {
		return fmt.Sprintf("PASS: %s (%s)", r.Artifact, r.Target)
	}
	if len(r.Missing) > 0 {
		return fmt.Sprintf("FAIL: %s (%s) - missing: %s", r.Artifact, r.Target, strings.Join(r.Missing, ", "))
	}
	return fmt.Sprintf("FAIL: %s (%s)", r.Artifact, r.Target)
}

// Policy holds the deployment context an artifact is evaluated against.
type Policy struct {
	// Environment is the target environment: development, staging,
	// production, air-gapped or hybrid-cloud. Empty skips environment checks.
	Environment string
	// Region is the target region for hybrid-cloud data residency checks.
	Region string
	// Strict turns data-residency mismatches into failures instead of warnings.
	Strict bool
}

// Section names used by the evaluators.
const (
	SectionTrustProfile   = "Trust Profile"
	SectionInfrastructure = "Infrastructure Requirements"
	SectionEnvironment    = "Environment Safety Policy"
)

// TrustProfileAnnotations must all be present for GitOps admission.
var TrustProfileAnnotations = []string{
	AnnotationSigningFramework,
	AnnotationSBOMFormat,
	AnnotationProvenanceType,
	AnnotationProfileVersion,
	AnnotationArtifactType,
}

// InfrastructureAnnotations must be present so policy engines can match the
// artifact to a destination; ConditionalInfrastructureAnnotations only
// produce a warning when absent.
var (
	InfrastructureAnnotations            = []string{AnnotationRuntime, AnnotationAccelerator}
	ConditionalInfrastructureAnnotations = []string{AnnotationCUDAVersionMin, AnnotationMemoryMin}
)

// EvaluateTrustProfile checks the annotations external policy engines need
// to verify signatures, SBOMs and provenance. Presence is checked here;
// cryptographic verification is delegated to those tools.
func EvaluateTrustProfile(r *ValidationReport, annotations map[string]string) {
	for _, key := range TrustProfileAnnotations {
		if v, ok := annotations[key]; ok {
			r.Pass(SectionTrustProfile, key, v)
		} else {
			r.Fail(SectionTrustProfile, key, "missing", key)
		}
	}
}

// EvaluateInfrastructure checks the runtime/accelerator annotations and
// warns about absent conditional ones (CUDA version, memory).
func EvaluateInfrastructure(r *ValidationReport, annotations map[string]string) {
	for _, key := range InfrastructureAnnotations {
		if v, ok := annotations[key]; ok {
			r.Pass(SectionInfrastructure, key, v)
		} else {
			r.Fail(SectionInfrastructure, key, "missing", key)
		}
	}
	for _, key := range ConditionalInfrastructureAnnotations {
		if v, ok := annotations[key]; ok {
			r.Pass(SectionInfrastructure, key, v)
		} else {
			r.Warn(SectionInfrastructure, key, "not present (optional)")
		}
	}
}

// EvaluateEnvironmentPolicy applies the air-gapped and hybrid-cloud safety
// policies for the target environment in policy.
func EvaluateEnvironmentPolicy(r *ValidationReport, annotations map[string]string, policy Policy) {
	switch policy.Environment {
	case "":
		r.Info(SectionEnvironment, "environment", "not specified, skipping safety policy checks")

	case "development", "staging", "production":
		r.Pass(SectionEnvironment, "environment", policy.Environment)

	case "air-gapped":
		// Both formats model-cli produces are self-contained OCI artifacts
		// that can be mirrored into an air-gapped registry; anything else
		// is unknown to us and worth a look.
		switch format, ok := annotations[AnnotationPackagingFormat]; {
		case !ok:
			r.Warn(SectionEnvironment, AnnotationPackagingFormat, "missing; cannot confirm air-gapped packaging")
		case format == PackagingFormatModelPack:
			r.Pass(SectionEnvironment, AnnotationPackagingFormat, format+" (Model Spec artifact; mirror with `modctl pull`/`push` into the air-gapped registry)")
		case format == PackagingFormatOCI:
			r.Pass(SectionEnvironment, AnnotationPackagingFormat, format+" (plain OCI artifact; mirror with `oras copy` into the air-gapped registry)")
		default:
			r.Warn(SectionEnvironment, AnnotationPackagingFormat, format+" is not a format model-cli produces; confirm it can be mirrored into the air-gapped registry")
		}
		if format, ok := annotations[AnnotationSBOMFormat]; ok {
			r.Pass(SectionEnvironment, AnnotationSBOMFormat, format+" (available for air-gapped compliance)")
		} else {
			r.Warn(SectionEnvironment, AnnotationSBOMFormat, "missing; air-gapped compliance needs an SBOM")
		}
		r.Info(SectionEnvironment, "dependencies", "ensure all dependencies are pre-loaded in the air-gapped registry")

	case "hybrid-cloud":
		if policy.Region == "" {
			r.Warn(SectionEnvironment, "region", "not specified for hybrid-cloud validation")
			return
		}
		if residency, ok := annotations[AnnotationDataResidency]; ok {
			if residency == policy.Region {
				r.Pass(SectionEnvironment, AnnotationDataResidency, residency+" matches target region")
			} else if policy.Strict {
				r.Fail(SectionEnvironment, AnnotationDataResidency,
					fmt.Sprintf("artifact requires %s, target is %s", residency, policy.Region),
					AnnotationDataResidency+"="+policy.Region)
			} else {
				r.Warn(SectionEnvironment, AnnotationDataResidency,
					fmt.Sprintf("artifact requires %s, target is %s", residency, policy.Region))
			}
		} else {
			r.Info(SectionEnvironment, AnnotationDataResidency, "no data residency requirement specified")
		}
		if network, ok := annotations[AnnotationNetworkAccess]; ok {
			if network == "public" {
				r.Warn(SectionEnvironment, AnnotationNetworkAccess, "public; may not suit all hybrid-cloud configurations")
			} else {
				r.Pass(SectionEnvironment, AnnotationNetworkAccess, network)
			}
		} else {
			r.Info(SectionEnvironment, AnnotationNetworkAccess, "no network access requirement specified")
		}
		r.Pass(SectionEnvironment, "region", policy.Region)

	default:
		r.Warn(SectionEnvironment, "environment", "unknown environment "+policy.Environment)
	}
}

// EvaluateArtifact runs the trust profile, infrastructure and environment
// evaluations an artifact must pass before GitOps tools deploy it.
func EvaluateArtifact(target, artifact string, annotations map[string]string, policy Policy) *ValidationReport {
	r := NewValidationReport(target, artifact)
	EvaluateTrustProfile(r, annotations)
	EvaluateInfrastructure(r, annotations)
	EvaluateEnvironmentPolicy(r, annotations, policy)
	return r
}
