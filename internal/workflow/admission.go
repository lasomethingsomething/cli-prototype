package workflow

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// AdmissionRequest represents a Kubernetes-style admission webhook request
type AdmissionRequest struct {
	// Standard Kubernetes admission request fields
	Kind       Kind   `json:"kind"`
	APIVersion string `json:"apiVersion"`
	Namespace  string `json:"namespace"`
	Operation  string `json:"operation"` // CREATE, UPDATE, DELETE, CONNECT

	// Object is the new object being admitted
	Object json.RawMessage `json:"object"`

	// OldObject is the existing object for UPDATE operations
	OldObject json.RawMessage `json:"oldObject,omitempty"`

	// UserInfo contains information about the requesting user
	UserInfo UserInfo `json:"userInfo"`

	// Additional context
	Resource Resource `json:"resource"`
}

// Kind represents the Kubernetes resource kind
type Kind struct {
	Group   string `json:"group"`
	Version string `json:"version"`
	Kind    string `json:"kind"`
}

// UserInfo represents user information from the admission request
type UserInfo struct {
	Username string              `json:"username"`
	UID      string              `json:"uid"`
	Groups   []string            `json:"groups"`
	Extra    map[string][]string `json:"extra,omitempty"`
}

// Resource represents the resource being admitted
type Resource struct {
	Group    string `json:"group"`
	Version  string `json:"version"`
	Resource string `json:"resource"`
}

// AdmissionResponse represents a Kubernetes-style admission webhook response
type AdmissionResponse struct {
	// Allowed indicates whether the request is allowed
	Allowed bool `json:"allowed"`

	// Result contains the reason if the request is denied
	Result *Metav1Status `json:"status,omitempty"`

	// Patch contains JSON Patch operations to mutate the object
	Patch []PatchOperation `json:"patch,omitempty"`

	// PatchType indicates the type of patch
	PatchType *string `json:"patchType,omitempty"`

	// Warnings contains warning messages
	Warnings []string `json:"warnings,omitempty"`

	// AuditAnnotations contains additional audit information
	AuditAnnotations map[string]string `json:"auditAnnotations,omitempty"`
}

// Metav1Status represents a Kubernetes status object
type Metav1Status struct {
	Code    int32  `json:"code"`
	Message string `json:"message"`
	Reason  string `json:"reason,omitempty"`
	Details string `json:"details,omitempty"`
}

// PatchOperation represents a JSON Patch operation
type PatchOperation struct {
	Op    string      `json:"op"`    // add, remove, replace
	Path  string      `json:"path"`  // JSON path
	Value interface{} `json:"value"` // Value to apply
}

// OCIManifestAdmissionRequest represents an admission request for an OCI manifest push
// This is a simplified version for registry-level admission control
type OCIManifestAdmissionRequest struct {
	// Artifact reference (e.g., ghcr.io/my-org/my-model:v1)
	Artifact string `json:"artifact"`

	// The OCI manifest being pushed
	Manifest json.RawMessage `json:"manifest"`

	// Manifest annotations extracted from the manifest
	Annotations map[string]string `json:"annotations,omitempty"`

	// Operation type (push, pull, delete)
	Operation string `json:"operation"`

	// Registry information
	Registry string `json:"registry"`

	// User information
	User string `json:"user,omitempty"`
}

// OCIManifestAdmissionResponse represents the response for an OCI manifest admission request
type OCIManifestAdmissionResponse struct {
	// Allowed indicates whether the manifest can be accepted
	Allowed bool `json:"allowed"`

	// Message contains a human-readable message
	Message string `json:"message,omitempty"`

	// Reason contains a machine-readable reason
	Reason string `json:"reason,omitempty"`

	// Errors contains validation errors
	Errors []string `json:"errors,omitempty"`

	// Warnings contains validation warnings
	Warnings []string `json:"warnings,omitempty"`

	// ValidationResult contains the detailed validation result
	Validation *ContractValidationResult `json:"validation,omitempty"`
}

// MetadataContractAdmissionWebhook implements an admission webhook for validating
// the Standardized Metadata Contract at manifest level
type MetadataContractAdmissionWebhook struct {
	// StrictMode determines whether to reject on warnings
	StrictMode bool

	// AllowedArtifactTypes defines which artifact types are allowed
	AllowedArtifactTypes []ArtifactType

	// AllowedRegistries defines which registries are allowed
	AllowedRegistries []string

	// Validator is a custom validation function
	Validator func(request *OCIManifestAdmissionRequest) (*ContractValidationResult, error)
}

// NewMetadataContractAdmissionWebhook creates a new admission webhook
func NewMetadataContractAdmissionWebhook() *MetadataContractAdmissionWebhook {
	return &MetadataContractAdmissionWebhook{
		StrictMode:           false,
		AllowedArtifactTypes: []ArtifactType{ArtifactTypeModel, ArtifactTypeSkill, ArtifactTypePipeline},
		AllowedRegistries:    []string{}, // Empty means all registries are allowed
	}
}

// HandleHTTPRequest handles an HTTP admission webhook request
func (w *MetadataContractAdmissionWebhook) HandleHTTPRequest(writer http.ResponseWriter, request *http.Request) {
	// Read the request body
	body, err := io.ReadAll(request.Body)
	if err != nil {
		w.writeErrorResponse(writer, http.StatusBadRequest, fmt.Sprintf("failed to read request body: %v", err))
		return
	}

	// Parse the admission request
	var req OCIManifestAdmissionRequest
	if err := json.Unmarshal(body, &req); err != nil {
		w.writeErrorResponse(writer, http.StatusBadRequest, fmt.Sprintf("failed to parse admission request: %v", err))
		return
	}

	// Validate the request
	response := w.Admit(&req)

	// Write the response
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(writer).Encode(response); err != nil {
		// Log error but don't fail the request
		w.writeErrorResponse(writer, http.StatusInternalServerError, fmt.Sprintf("failed to write response: %v", err))
		return
	}
}

// Admit validates an OCI manifest admission request
func (w *MetadataContractAdmissionWebhook) Admit(request *OCIManifestAdmissionRequest) *OCIManifestAdmissionResponse {
	response := &OCIManifestAdmissionResponse{
		Allowed: true,
		Message: "Manifest admitted",
	}

	// Validate operation type
	if request.Operation != "push" && request.Operation != "CREATE" {
		// For non-push operations, we might not need validation
		// But we can still check if it's a delete or other operation
		if request.Operation == "delete" || request.Operation == "DELETE" {
			response.Allowed = true
			response.Message = "Delete operations are always allowed"
			return response
		}
	}

	// Check registry allowlist
	if len(w.AllowedRegistries) > 0 {
		allowed := false
		for _, reg := range w.AllowedRegistries {
			if request.Registry == reg {
				allowed = true
				break
			}
		}
		if !allowed {
			return &OCIManifestAdmissionResponse{
				Allowed: false,
				Message: fmt.Sprintf("registry %s is not in the allowlist", request.Registry),
				Reason:  "ForbiddenRegistry",
			}
		}
	}

	// Determine artifact type from annotations
	artifactType := determineArtifactType(request.Annotations)
	if artifactType == "" {
		// Try to determine from the artifact reference
		artifactType = inferArtifactTypeFromReference(request.Artifact)
	}

	// Validate artifact type
	if artifactType == "" {
		return &OCIManifestAdmissionResponse{
			Allowed: false,
			Message: "unable to determine artifact type from manifest or reference",
			Reason:  "MissingArtifactType",
			Errors:  []string{"artifact type annotation is required: " + AnnotationArtifactType},
		}
	}

	// Check if artifact type is allowed
	if len(w.AllowedArtifactTypes) > 0 {
		allowed := false
		for _, allowedType := range w.AllowedArtifactTypes {
			if ArtifactType(artifactType) == allowedType {
				allowed = true
				break
			}
		}
		if !allowed {
			return &OCIManifestAdmissionResponse{
				Allowed: false,
				Message: fmt.Sprintf("artifact type %s is not allowed", artifactType),
				Reason:  "ForbiddenArtifactType",
			}
		}
	}

	// Validate the metadata contract
	result := ValidateManifestMetadata(request.Annotations, ArtifactType(artifactType))
	response.Validation = result

	if !result.Valid {
		return &OCIManifestAdmissionResponse{
			Allowed:    false,
			Message:    result.String(),
			Reason:     "InvalidMetadataContract",
			Errors:     result.Errors,
			Warnings:   result.Warnings,
			Validation: result,
		}
	}

	// Check for warnings in strict mode
	if w.StrictMode && len(result.Warnings) > 0 {
		return &OCIManifestAdmissionResponse{
			Allowed:    false,
			Message:    fmt.Sprintf("strict mode: %d warning(s) found", len(result.Warnings)),
			Reason:     "MetadataWarningsInStrictMode",
			Warnings:   result.Warnings,
			Validation: result,
		}
	}

	// If we have warnings but not in strict mode, include them
	if len(result.Warnings) > 0 {
		response.Warnings = result.Warnings
		response.Message = fmt.Sprintf("Manifest admitted with %d warning(s)", len(result.Warnings))
	}

	return response
}

// determineArtifactType determines the artifact type from annotations
func determineArtifactType(annotations map[string]string) string {
	if annotations == nil {
		return ""
	}

	// Check for the standard artifact type annotation
	if val, ok := annotations[AnnotationArtifactType]; ok {
		return val
	}

	// Check for legacy annotation
	if val, ok := annotations["org.cncf.ai.artifact.type"]; ok {
		return val
	}

	return ""
}

// inferArtifactTypeFromReference tries to infer artifact type from the artifact reference
func inferArtifactTypeFromReference(artifact string) string {
	// This is a best-effort inference based on naming conventions
	// In production, this should be explicitly set in annotations

	lower := strings.ToLower(artifact)

	// Check for common patterns
	if strings.Contains(lower, "model") || strings.Contains(lower, "-model") || strings.Contains(lower, "_model") {
		return string(ArtifactTypeModel)
	}
	if strings.Contains(lower, "skill") || strings.Contains(lower, "-skill") || strings.Contains(lower, "_skill") {
		return string(ArtifactTypeSkill)
	}
	if strings.Contains(lower, "pipeline") || strings.Contains(lower, "-pipeline") || strings.Contains(lower, "_pipeline") {
		return string(ArtifactTypePipeline)
	}

	// Default to model
	return string(ArtifactTypeModel)
}

// writeErrorResponse writes an error response
func (w *MetadataContractAdmissionWebhook) writeErrorResponse(writer http.ResponseWriter, statusCode int, message string) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(statusCode)

	response := &OCIManifestAdmissionResponse{
		Allowed: false,
		Message: message,
		Reason:  http.StatusText(statusCode),
	}

	json.NewEncoder(writer).Encode(response)
}

// ValidateOCIManifestForPush validates an OCI manifest before allowing push
// This is the main function to be called by registry middleware or admission proxies
func ValidateOCIManifestForPush(manifest *UnifiedOCIManifest) (*ContractValidationResult, error) {
	if manifest == nil {
		return nil, fmt.Errorf("manifest is nil")
	}

	// Determine artifact type
	artifactTypeStr := manifest.Annotations[AnnotationArtifactType]
	if artifactTypeStr == "" {
		return &ContractValidationResult{
			Valid:         false,
			Errors:        []string{"artifact type annotation is required: " + AnnotationArtifactType},
			ArtifactType:  "unknown",
			MissingFields: []string{AnnotationArtifactType},
		}, nil
	}

	artifactType := ArtifactType(artifactTypeStr)

	// Extract metadata contract from annotations
	// The contract can be in a separate annotation or embedded in the manifest
	contractJSON := manifest.Annotations[AnnotationMetadataContract]

	if contractJSON != "" {
		// Parse and validate the contract
		var contract MetadataContract
		if err := json.Unmarshal([]byte(contractJSON), &contract); err != nil {
			return &ContractValidationResult{
				Valid:         false,
				Errors:        []string{fmt.Sprintf("failed to parse metadata contract: %v", err)},
				ArtifactType:  string(artifactType),
				MissingFields: []string{AnnotationMetadataContract},
			}, nil
		}

		return ValidateContract(&contract, artifactType), nil
	}

	// If no explicit contract, try to build one from the AIConfig
	contract := buildContractFromAIConfig(manifest)
	if contract != nil {
		return ValidateContract(contract, artifactType), nil
	}

	// No contract found - validate based on AIConfig presence
	return ValidateManifestMetadata(manifest.Annotations, artifactType), nil
}

// buildContractFromAIConfig builds a metadata contract from the AIConfig in the manifest
func buildContractFromAIConfig(manifest *UnifiedOCIManifest) *MetadataContract {
	if manifest.AIConfig == nil {
		return nil
	}

	contract := &MetadataContract{
		Assets: ContractAssetMetadata{},
	}

	artifactType := manifest.Annotations[AnnotationArtifactType]

	switch artifactType {
	case string(ArtifactTypeModel):
		if config, ok := manifest.AIConfig.(AIModelConfig); ok {
			contract.Assets.Model = &ContractModelMetadata{
				Type:          config.ModelType,
				Framework:     config.ModelFormat,
				Input:         config.InputFormat,
				Output:        config.OutputFormat,
				Capabilities:  config.Capabilities,
				Runtime:       config.Runtime,
				Accelerator:   config.Accelerator,
				Relationships: config.Relationships,
				Description:   config.Description,
				Version:       config.Version,
				Author:        config.Author,
				License:       config.License,
			}
		}
	case string(ArtifactTypeSkill):
		if config, ok := manifest.AIConfig.(AISkillConfig); ok {
			contract.Assets.Skill = &ContractSkillMetadata{
				Type:         config.SkillType,
				PipelineRef:  "", // Not available in AISkillConfig
				Dependencies: config.Dependencies,
				Runtime:      config.Runtime,
				Accelerator:  config.Accelerator,
				Description:  config.Description,
				Version:      config.Version,
				Author:       config.Author,
			}
		}
	case string(ArtifactTypePipeline):
		if config, ok := manifest.AIConfig.(AIPipelineConfig); ok {
			stages := make([]string, len(config.Components))
			for i, comp := range config.Components {
				stages[i] = comp.Type
			}
			// Convert pipeline components
			var contractComps []ContractPipelineComponent
			for _, comp := range config.Components {
				contractComps = append(contractComps, ContractPipelineComponent{
					Name:      comp.Name,
					Type:      comp.Type,
					Reference: comp.Reference,
				})
			}
			contract.Assets.Pipeline = &ContractPipelineMetadata{
				Type:         config.PipelineType,
				Stages:       stages,
				Dependencies: config.Dependencies,
				Components:   contractComps,
				Description:  config.Description,
				Version:      config.Version,
				Author:       config.Author,
			}
		}
	}

	return contract
}

// CreateAdmissionMiddleware creates a middleware function for HTTP handlers
// This can be used with registry implementations to validate manifests on push
func CreateAdmissionMiddleware(webhook *MetadataContractAdmissionWebhook) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			// Skip validation for non-push operations
			if request.Method != http.MethodPost && request.Method != http.MethodPut {
				next.ServeHTTP(writer, request)
				return
			}

			// Check if this is a manifest push
			if !isManifestPushRequest(request) {
				next.ServeHTTP(writer, request)
				return
			}

			// Parse and validate the manifest
			manifest, err := parseManifestFromRequest(request)
			if err != nil {
				// If we can't parse, let the request through
				// (the registry will handle malformed manifests)
				next.ServeHTTP(writer, request)
				return
			}

			// Validate using the webhook
			result, err := ValidateOCIManifestForPush(manifest)
			if err != nil {
				// Log error but don't block
				next.ServeHTTP(writer, request)
				return
			}

			if !result.Valid {
				// Reject the push
				writer.Header().Set("Content-Type", "application/json")
				writer.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(writer).Encode(&OCIManifestAdmissionResponse{
					Allowed:  false,
					Message:  result.String(),
					Reason:   "InvalidMetadataContract",
					Errors:   result.Errors,
					Warnings: result.Warnings,
				})
				return
			}

			// If strict mode and warnings, reject
			if webhook.StrictMode && len(result.Warnings) > 0 {
				writer.Header().Set("Content-Type", "application/json")
				writer.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(writer).Encode(&OCIManifestAdmissionResponse{
					Allowed:  false,
					Message:  fmt.Sprintf("strict mode: %d warning(s)", len(result.Warnings)),
					Reason:   "MetadataWarningsInStrictMode",
					Warnings: result.Warnings,
				})
				return
			}

			// Allow the request through
			next.ServeHTTP(writer, request)
		})
	}
}

// isManifestPushRequest checks if a request is a manifest push
func isManifestPushRequest(request *http.Request) bool {
	// Check content type
	contentType := request.Header.Get("Content-Type")
	if contentType != "application/vnd.oci.image.manifest.v1+json" &&
		contentType != "application/json" {
		return false
	}

	// Check path patterns common in registry APIs
	path := request.URL.Path
	return strings.Contains(path, "/v2/") && strings.Contains(path, "/manifests")
}

// parseManifestFromRequest parses an OCI manifest from an HTTP request
func parseManifestFromRequest(request *http.Request) (*UnifiedOCIManifest, error) {
	body, err := io.ReadAll(request.Body)
	if err != nil {
		return nil, err
	}

	// Reset the body for the next handler
	request.Body = io.NopCloser(strings.NewReader(string(body)))

	var manifest UnifiedOCIManifest
	if err := json.Unmarshal(body, &manifest); err != nil {
		return nil, err
	}

	return &manifest, nil
}

// RegistryAdmissionProxy simulates a registry admission proxy
// This can be used as a standalone service or integrated with a registry
type RegistryAdmissionProxy struct {
	Webhook *MetadataContractAdmissionWebhook
	Port    int
}

// NewRegistryAdmissionProxy creates a new registry admission proxy
func NewRegistryAdmissionProxy(port int) *RegistryAdmissionProxy {
	return &RegistryAdmissionProxy{
		Webhook: NewMetadataContractAdmissionWebhook(),
		Port:    port,
	}
}

// Run starts the admission proxy server
func (p *RegistryAdmissionProxy) Run() error {
	http.HandleFunc("/validate", p.Webhook.HandleHTTPRequest)

	addr := fmt.Sprintf(":%d", p.Port)
	fmt.Printf("Starting registry admission proxy on %s\n", addr)

	return http.ListenAndServe(addr, nil)
}
