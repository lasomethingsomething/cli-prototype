package workflow

import "fmt"

// Annotation constants for the CNCF AI Interoperability Profile (v0.1 draft).
// Spec: https://github.com/cncf/toc/blob/main/tags/tag-developer-experience/initiatives/cloud-native-oci-compliant-inner-loop/annotations.md
// Keys outside the org.cncf.ai namespace (ai.node.*, ai.runtime.*, ai.skill.*)
// are model-cli extensions and not part of the interoperability contract.

// Profile annotations (MUST)
const (
	// ProfileVersion identifies the interoperability profile version
	// Value: semantic version (e.g., "1.0.0")
	AnnotationProfileVersion = "org.cncf.ai.interop.profile.version"

	// ArtifactType identifies the type of AI artifact
	// Value: "model" (v1), future: "skill", "rag-context", "workflow", etc.
	AnnotationArtifactType = "org.cncf.ai.artifact.type"

	// LifecycleStatus is the maturity/promotion status of the artifact,
	// independent of structural conformance (a fully compliant artifact can
	// still be experimental).
	// Value: one of LifecycleStatusExperimental, LifecycleStatusValidated,
	// LifecycleStatusDeprecated, LifecycleStatusProductReady
	AnnotationLifecycleStatus = "org.cncf.ai.lifecycle.status"
)

// Values of AnnotationLifecycleStatus
const (
	LifecycleStatusExperimental = "experimental"
	LifecycleStatusValidated    = "validated"
	LifecycleStatusDeprecated   = "deprecated"
	LifecycleStatusProductReady = "product-ready"
)

// Reused OCI image-spec annotations. The profile does not redefine these:
// "The OCI Image Spec defines a set of annotations that can be applied to
// several components of the specification. These predefined keys should
// be reused wherever possible."
// Reference: https://specs.opencontainers.org/image-spec/annotations/
const (
	// OCIImageVersion is the version of the model artifact itself (not the
	// profile version).
	OCIImageVersion = "org.opencontainers.image.version"
	// OCIImageTitle is the model artifact name.
	OCIImageTitle = "org.opencontainers.image.title"
	// OCIImageDescription describes the artifact.
	OCIImageDescription = "org.opencontainers.image.description"
	// OCIImageAuthors is the authorship of the model artifact.
	OCIImageAuthors = "org.opencontainers.image.authors"
	// OCIImageSource is the source repository/location of the model.
	OCIImageSource = "org.opencontainers.image.source"
	// OCIImageCreated is the creation timestamp (RFC 3339).
	OCIImageCreated = "org.opencontainers.image.created"
	// OCILicenses is the license identifier(s) of the model artifact.
	OCILicenses = "org.opencontainers.image.licenses"
)

// MOF (Model Openness Framework) annotations (MUST when applicable)
const (
	// MOFClass identifies the openness class of the model
	// Value: "I", "II", or "III"
	AnnotationMOFClass = "org.cncf.ai.model.mof.class"

	// MOFVersion identifies the MOF specification version
	// Value: semantic version (e.g., "1.0")
	AnnotationMOFVersion = "org.cncf.ai.model.mof.version"

	// MOFComponents lists which MOF components are present.
	// Value: comma-separated, no surrounding whitespace, drawn from the
	// profile vocabulary (datasets, data-preprocessing-code, model-architecture,
	// final-model-parameters, intermediate-model-parameters, model-metadata,
	// training-code, inference-code, evaluation-code, evaluation-data,
	// evaluation-results, supporting-libraries-and-tools, model-card, data-card,
	// technical-report, research-paper, sample-model-outputs,
	// model-openness-config-file). model-openness-config-file is always
	// required regardless of class.
	AnnotationMOFComponents = "org.cncf.ai.model.mof.components"
)

// MOFComponentsClassI is the full enumerated component list consistent with a
// Class I (most open) claim under the v1 profile. Used as the default MOF
// claim so the annotation at least conforms to the vocabulary; harden/detect
// refines it from the actual model files.
const MOFComponentsClassI = "datasets,data-preprocessing-code,model-architecture," +
	"final-model-parameters,intermediate-model-parameters,model-metadata,training-code," +
	"inference-code,evaluation-code,evaluation-data,evaluation-results," +
	"supporting-libraries-and-tools,model-card,data-card,technical-report,research-paper," +
	"sample-model-outputs,model-openness-config-file"

// Security annotations (MUST/SHOULD)
const (
	// SigningFramework identifies the signing framework used
	// Value: "sigstore", "notation" (extensible per profile open question #2)
	AnnotationSigningFramework = "org.cncf.ai.security.signing.framework"

	// SBOMFormat identifies the SBOM format
	// Value: "spdx", "cyclonedx"
	AnnotationSBOMFormat = "org.cncf.ai.security.sbom.format"

	// ProvenanceType identifies the provenance attestation type
	// Value: "slsa-v1.0", "in-toto", etc.
	AnnotationProvenanceType = "org.cncf.ai.security.provenance.type"

	// PackagingFormat identifies the packaging format
	// Value: PackagingFormatOCI or PackagingFormatModelPack, set by the
	// package workflow from the registry tool that pushed the artifact.
	AnnotationPackagingFormat = "org.cncf.ai.packaging.format"
)

// Values of AnnotationPackagingFormat. Each registry tool produces one of
// them (RegistryProvider.PackagingFormat); the annotation is deliberately not
// defaulted so a manifest never claims a format that was not used.
const (
	// PackagingFormatOCI is a plain OCI artifact, as pushed with ORAS. It uses
	// no model packaging format, so the annotation value is "none" per the
	// profile's allowed values ("modelpack", "none").
	PackagingFormatOCI = "none"
	// PackagingFormatModelPack is a CNCF ModelPack Model Spec artifact, as
	// built and pushed with modctl.
	PackagingFormatModelPack = "modelpack"
)

// Runtime annotations for deployment
const (
	// Runtime identifies the serving runtime
	// Value: "vllm", "kserve", "tensorrt-llm", etc.
	AnnotationRuntime = "org.cncf.ai.runtime"

	// Accelerator identifies the hardware accelerator requirement
	// Value: "nvidia-gpu", "amd-gpu", "intel-gpu", "cpu", "none"
	AnnotationAccelerator = "org.cncf.ai.accelerator"

	// CUDAVersionMin identifies minimum CUDA version required
	// Value: semantic version (e.g., "12.1")
	AnnotationCUDAVersionMin = "org.cncf.ai.accelerator.cuda.min"

	// MemoryMin identifies minimum memory required
	// Value: string with unit (e.g., "24GiB")
	AnnotationMemoryMin = "org.cncf.ai.resource.memory.min"

	// DataResidency identifies data residency requirement for hybrid-cloud
	// Value: region identifier (e.g., "us-east-1", "eu-west-1")
	AnnotationDataResidency = "org.cncf.ai.data.residency"

	// NetworkAccess identifies network access requirement
	// Value: "internal", "private", "public"
	AnnotationNetworkAccess = "org.cncf.ai.network.access"

	// Node requirement annotations for infrastructure orchestration (Story #68)
	// GPUType identifies the specific GPU type required
	// Value: e.g., "nvidia-a100", "nvidia-h100", "nvidia-l40s"
	AnnotationGPUType = "ai.node.gpu.type"

	// VRAMLabel identifies the minimum vRAM requirement per GPU
	// Value: string with unit (e.g., "40GiB", "80GiB")
	AnnotationVRAMMin = "ai.node.vram.min"

	// GPUTopology identifies the GPU topology requirement
	// Value: e.g., "8xH100", "4xA100", "2xL40S"
	AnnotationGPUTopology = "ai.node.gpu.topology"

	// Runtime execution annotations (Story #69)
	// RuntimeType identifies the specific runtime for serving
	// Value: e.g., "vllm", "kserve", "tensorrt-llm"
	AnnotationRuntimeType = "ai.runtime.type"

	// LayerDeduplication identifies if layer deduplication optimization is enabled
	// Value: "true" or "false"
	AnnotationLayerDeduplication = "ai.runtime.optimization.layer-dedup"

	// ReferenceSkillDLC identifies the endpoint for dynamic skill loading
	// Value: URL or connection string for Reference Skill DLC
	AnnotationReferenceSkillDLC = "ai.skill.dlc-endpoint"

	// SkillReferences lists skill dependencies for agentic workflows
	// Value: comma-separated list of skill references (e.g., "skill:sha256:abc,skill:sha256:def")
	AnnotationSkillReferences = "ai.skill.references"
)

// AnnotationSet represents a collection of annotations for an OCI artifact
type AnnotationSet struct {
	// Core interop profile
	ProfileVersion  string
	ArtifactType    string
	LifecycleStatus string

	// MOF classification
	MOFClass      string
	MOFVersion    string
	MOFComponents string

	// Security
	SigningFramework string
	SBOMFormat       string
	ProvenanceType   string
	PackagingFormat  string

	// Runtime requirements
	Runtime     string
	Accelerator string
	CUDAMin     string
	MemoryMin   string

	// Node requirements for infrastructure orchestration (Story #68)
	GPUType     string
	VRAMMin     string
	GPUTopology string

	// Runtime execution for Story #69
	RuntimeType        string
	LayerDeduplication string
	ReferenceSkillDLC  string
	SkillReferences    string
}

// NewAnnotationSet creates a new annotation set with sensible defaults
// Note: Runtime, Accelerator, CUDAMin, and MemoryMin are intentionally empty now.
// They must be explicitly set based on the model format to prevent silently
// defaulting to vllm/nvidia-gpu for models like sklearn that require CPU.
func NewAnnotationSet() *AnnotationSet {
	return &AnnotationSet{
		ProfileVersion:   "1.0.0",
		ArtifactType:     "model",
		LifecycleStatus:  LifecycleStatusExperimental,
		MOFVersion:       "1.0",
		SigningFramework: "sigstore", // Default, can be overridden by user choice
		SBOMFormat:       "spdx",
		ProvenanceType:   "slsa-v1.0",
		Runtime:          "",
		Accelerator:      "",
		CUDAMin:          "",
		MemoryMin:        "",
	}
}

// ToMap converts the annotation set to a map for OCI manifest
func (a *AnnotationSet) ToMap() map[string]string {
	annotations := make(map[string]string)

	// Profile annotations (MUST)
	if a.ProfileVersion != "" {
		annotations[AnnotationProfileVersion] = a.ProfileVersion
	}
	if a.ArtifactType != "" {
		annotations[AnnotationArtifactType] = a.ArtifactType
	}
	if a.LifecycleStatus != "" {
		annotations[AnnotationLifecycleStatus] = a.LifecycleStatus
	}

	// MOF annotations (MUST when applicable)
	if a.MOFClass != "" {
		annotations[AnnotationMOFClass] = a.MOFClass
	}
	if a.MOFVersion != "" {
		annotations[AnnotationMOFVersion] = a.MOFVersion
	}
	if a.MOFComponents != "" {
		annotations[AnnotationMOFComponents] = a.MOFComponents
	}

	// Security annotations
	if a.SigningFramework != "" {
		annotations[AnnotationSigningFramework] = a.SigningFramework
	}
	if a.SBOMFormat != "" {
		annotations[AnnotationSBOMFormat] = a.SBOMFormat
	}
	if a.ProvenanceType != "" {
		annotations[AnnotationProvenanceType] = a.ProvenanceType
	}
	if a.PackagingFormat != "" {
		annotations[AnnotationPackagingFormat] = a.PackagingFormat
	}

	// Runtime annotations
	if a.Runtime != "" {
		annotations[AnnotationRuntime] = a.Runtime
	}
	if a.Accelerator != "" {
		annotations[AnnotationAccelerator] = a.Accelerator
	}
	if a.CUDAMin != "" {
		annotations[AnnotationCUDAVersionMin] = a.CUDAMin
	}
	if a.MemoryMin != "" {
		annotations[AnnotationMemoryMin] = a.MemoryMin
	}

	// Node requirement annotations (Story #68)
	if a.GPUType != "" {
		annotations[AnnotationGPUType] = a.GPUType
	}
	if a.VRAMMin != "" {
		annotations[AnnotationVRAMMin] = a.VRAMMin
	}
	if a.GPUTopology != "" {
		annotations[AnnotationGPUTopology] = a.GPUTopology
	}

	// Runtime execution annotations (Story #69)
	if a.RuntimeType != "" {
		annotations[AnnotationRuntimeType] = a.RuntimeType
	}
	if a.LayerDeduplication != "" {
		annotations[AnnotationLayerDeduplication] = a.LayerDeduplication
	}
	if a.ReferenceSkillDLC != "" {
		annotations[AnnotationReferenceSkillDLC] = a.ReferenceSkillDLC
	}
	if a.SkillReferences != "" {
		annotations[AnnotationSkillReferences] = a.SkillReferences
	}

	return annotations
}

// Print returns a human-readable representation of the annotations
func (a *AnnotationSet) Print() {
	fmt.Println("OCI Annotations (CNCF AI Interoperability Profile):")
	fmt.Println("  Profile:")
	fmt.Printf("    %s: %s\n", AnnotationProfileVersion, a.ProfileVersion)
	fmt.Printf("    %s: %s\n", AnnotationArtifactType, a.ArtifactType)
	fmt.Printf("    %s: %s\n", AnnotationLifecycleStatus, a.LifecycleStatus)

	fmt.Println("  MOF:")
	fmt.Printf("    %s: %s\n", AnnotationMOFClass, a.MOFClass)
	fmt.Printf("    %s: %s\n", AnnotationMOFVersion, a.MOFVersion)
	fmt.Printf("    %s: %s\n", AnnotationMOFComponents, a.MOFComponents)

	fmt.Println("  Security:")
	fmt.Printf("    %s: %s\n", AnnotationSigningFramework, a.SigningFramework)
	fmt.Printf("    %s: %s\n", AnnotationSBOMFormat, a.SBOMFormat)
	fmt.Printf("    %s: %s\n", AnnotationProvenanceType, a.ProvenanceType)
	fmt.Printf("    %s: %s\n", AnnotationPackagingFormat, a.PackagingFormat)

	fmt.Println("  Runtime:")
	fmt.Printf("    %s: %s\n", AnnotationRuntime, a.Runtime)
	fmt.Printf("    %s: %s\n", AnnotationAccelerator, a.Accelerator)
	fmt.Printf("    %s: %s\n", AnnotationCUDAVersionMin, a.CUDAMin)
	fmt.Printf("    %s: %s\n", AnnotationMemoryMin, a.MemoryMin)

	fmt.Println("  Node Requirements:")
	fmt.Printf("    %s: %s\n", AnnotationGPUType, a.GPUType)
	fmt.Printf("    %s: %s\n", AnnotationVRAMMin, a.VRAMMin)
	fmt.Printf("    %s: %s\n", AnnotationGPUTopology, a.GPUTopology)

	fmt.Println("  Runtime Execution:")
	fmt.Printf("    %s: %s\n", AnnotationRuntimeType, a.RuntimeType)
	fmt.Printf("    %s: %s\n", AnnotationLayerDeduplication, a.LayerDeduplication)
	fmt.Printf("    %s: %s\n", AnnotationReferenceSkillDLC, a.ReferenceSkillDLC)
	fmt.Printf("    %s: %s\n", AnnotationSkillReferences, a.SkillReferences)
}

// ProfileSBOMFormat maps an internal SBOM tool format (syft/trivy output
// format) to the profile's annotation vocabulary ("spdx" or "cyclonedx").
func ProfileSBOMFormat(format SBOMFormat) string {
	switch format {
	case SPDXJSON, SPDXTagValue:
		return "spdx"
	case CycloneDXJSON, CycloneDXXML:
		return "cyclonedx"
	default:
		return "spdx"
	}
}
