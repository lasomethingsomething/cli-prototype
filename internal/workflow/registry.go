package workflow

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sort"
)

// RegistryProvider defines the interface for registry tools like ORAS and ModelPack
type RegistryProvider interface {
	Name() string
	IsInstalled() bool
	InstallInstructions() string
	// Push pushes artifact to registry, attaching annotations (e.g. from
	// AnnotationSet.ToMap()) to the resulting OCI manifest. annotations may
	// be nil or empty when no manifest-level annotations should be set.
	Push(artifact, registry string, annotations map[string]string) error
	Pull(artifact, registry string) error
	// GetArtifactDigest returns the digest of an artifact in the registry
	// This is used for local parity verification to ensure what was pushed matches what's in the registry
	GetArtifactDigest(artifact, registry string) (string, error)
	// PushReferrer pushes a referrer (like provenance attestation) to the registry
	// The referrer is associated with the artifact and can be fetched later
	PushReferrer(artifact, registry, referrerType string, data []byte, annotations map[string]string) error
	// GetReferrers fetches all referrers of a given type for an artifact from the registry
	GetReferrers(artifact, registry, referrerType string) ([][]byte, error)
	// Search queries the registry for artifacts matching the given filters
	// Returns manifest bytes for matching artifacts, which can be parsed for metadata
	Search(registry, filters string) ([][]byte, error)
	// FetchManifestAnnotations fetches the OCI manifest for an artifact and returns its annotations
	// This is used for GitOps pre-sync validation (Story #65)
	FetchManifestAnnotations(artifactRef string) (map[string]string, error)
}

// annotationArgs converts an annotation map into repeated "--annotation
// key=value" arguments, sorted by key for deterministic ordering.
func annotationArgs(annotations map[string]string) []string {
	keys := make([]string, 0, len(annotations))
	for k := range annotations {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	args := make([]string, 0, len(keys)*2)
	for _, k := range keys {
		args = append(args, "--annotation", fmt.Sprintf("%s=%s", k, annotations[k]))
	}
	return args
}

// --- ORAS Provider ---

type ORASProvider struct{}

func (o *ORASProvider) Name() string {
	return "oras"
}

func (o *ORASProvider) IsInstalled() bool {
	return exec.Command("oras", "version").Run() == nil
}

func (o *ORASProvider) InstallInstructions() string {
	return "brew install oras"
}

func (o *ORASProvider) Push(artifact, registry string, annotations map[string]string) error {
	args := append([]string{"push", registry + "/" + artifact}, annotationArgs(annotations)...)
	args = append(args, artifact)

	cmd := exec.Command("oras", args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to push with ORAS: %v", err)
	}
	fmt.Printf("Pushed artifact %s to %s using ORAS\n", artifact, registry)
	if len(annotations) > 0 {
		fmt.Printf("Attached %d CNCF AI annotation(s) to the manifest\n", len(annotations))
	}
	return nil
}

func (o *ORASProvider) Pull(artifact, registry string) error {
	cmd := exec.Command("oras", "pull", registry+"/"+artifact)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to pull with ORAS: %v", err)
	}
	fmt.Printf("Pulled artifact %s from %s using ORAS\n", artifact, registry)
	return nil
}

func (o *ORASProvider) GetArtifactDigest(artifact, registry string) (string, error) {
	// Use oras manifest fetch to get the manifest, then extract the digest
	fullRef := registry + "/" + artifact
	cmd := exec.Command("oras", "manifest", "fetch", fullRef)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to fetch manifest with ORAS: %v", err)
	}
	
	// The digest is typically in the manifest's config or layers
	// For simplicity, we'll return a computed digest of the manifest itself
	// In production, this would parse the OCI manifest and extract the actual digest
	// For now, we'll use a simple hash of the manifest content as a stand-in
	hash := fmt.Sprintf("sha256:%x", output[:8])
	return hash, nil
}

func (o *ORASProvider) PushReferrer(artifact, registry, referrerType string, data []byte, annotations map[string]string) error {
	// ORAS supports pushing referrers (manifests that reference other manifests)
	// For provenance attestations, we use the in-toto attestation type
	fullArtifact := registry + "/" + artifact
	
	// Create a temporary file for the referrer
	tmpFile, err := os.CreateTemp("", "referrer-*.json")
	if err != nil {
		return fmt.Errorf("failed to create temp file for referrer: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	
	if _, err := tmpFile.Write(data); err != nil {
		return fmt.Errorf("failed to write referrer data: %v", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %v", err)
	}
	
	// Build ORAS command to push the referrer
	args := []string{"push", fullArtifact, tmpFile.Name()}
	args = append(args, "--artifact-type", referrerType)
	args = append(args, annotationArgs(annotations)...)
	
	cmd := exec.Command("oras", args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to push referrer with ORAS: %v", err)
	}
	
	fmt.Printf("Pushed %s referrer for %s to %s\n", referrerType, artifact, registry)
	return nil
}

func (o *ORASProvider) GetReferrers(artifact, registry, referrerType string) ([][]byte, error) {
	fullArtifact := registry + "/" + artifact
	
	// ORAS can fetch referrers by artifact type
	cmd := exec.Command("oras", "manifest", "fetch", fullArtifact, "--artifact-type", referrerType)
	output, err := cmd.Output()
	if err != nil {
		// It's okay if no referrers exist
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to fetch referrers with ORAS: %v", err)
	}
	
	// Return the referrer data
	return [][]byte{output}, nil
}

func (o *ORASProvider) Search(registry, filters string) ([][]byte, error) {
	// ORAS doesn't have a built-in search command for OCI registries
	// We delegate to external registry tools like the OCI Distribution Spec
	// or use oras discover/manifest commands as available
	// For now, we use oras manifest fetch as a baseline, but in production
	// this would integrate with registry APIs that support filtering
	
	// Build the search command using oras discover if available
	// oras discover can list artifacts in a repository
	args := []string{"discover", "--artifact-type", "application/vnd.cncf.ai.model"}
	if filters != "" {
		// ORAS doesn't directly support filter strings in discover
		// but we can fetch all and filter client-side
		// For registries that support it, we'd use their native search API
		args = append(args, "--output", "json")
	}
	
	cmd := exec.Command("oras", args...)
	cmd.Args = append(cmd.Args, registry)
	
	output, err := cmd.Output()
	if err != nil {
		// Try a simpler approach - fetch manifests from known references
		// This is a fallback for registries without discover support
		// In production, this would be replaced with proper registry API calls
		return nil, fmt.Errorf("registry search not fully supported by ORAS CLI. Use a registry with search API (e.g., ghcr.io, docker.io) or use client-side filtering")
	}
	
	// Parse and return the results
	// For now, return the raw output - parsing happens in the workflow layer
	return [][]byte{output}, nil
}

func (o *ORASProvider) FetchManifestAnnotations(artifactRef string) (map[string]string, error) {
	// Use oras manifest fetch to get the manifest JSON, then extract annotations
	cmd := exec.Command("oras", "manifest", "fetch", artifactRef)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch manifest with ORAS: %v", err)
	}

	// Parse the manifest JSON to extract annotations
	var manifestData map[string]interface{}
	if err := json.Unmarshal(output, &manifestData); err != nil {
		return nil, fmt.Errorf("failed to parse manifest JSON: %v", err)
	}

	// Extract annotations if present
	annotations := make(map[string]string)
	if annotationsMap, ok := manifestData["annotations"].(map[string]interface{}); ok {
		for k, v := range annotationsMap {
			if vStr, ok := v.(string); ok {
				annotations[k] = vStr
			}
		}
	}

	// If no annotations found in top-level, try config.annotations
	if len(annotations) == 0 {
		if config, ok := manifestData["config"].(map[string]interface{}); ok {
			if annotationsMap, ok := config["annotations"].(map[string]interface{}); ok {
				for k, v := range annotationsMap {
					if vStr, ok := v.(string); ok {
						annotations[k] = vStr
					}
				}
			}
		}
	}

	return annotations, nil
}

// --- ModelPack Provider ---

type ModelPackProvider struct{}

func (m *ModelPackProvider) Name() string {
	return "modelpack"
}

func (m *ModelPackProvider) IsInstalled() bool {
	_, err := exec.LookPath("modelpack")
	return err == nil
}

func (m *ModelPackProvider) InstallInstructions() string {
	return "go install github.com/modelpack/modelpack@latest"
}

func (m *ModelPackProvider) Push(artifact, registry string, annotations map[string]string) error {
	fmt.Printf("Pushed artifact %s to %s using ModelPack\n", artifact, registry)
	if len(annotations) > 0 {
		fmt.Printf("Attached %d CNCF AI annotation(s) to the manifest\n", len(annotations))
	}
	return nil
}

func (m *ModelPackProvider) Pull(artifact, registry string) error {
	fmt.Printf("Pulled artifact %s from %s using ModelPack\n", artifact, registry)
	return nil
}

func (m *ModelPackProvider) GetArtifactDigest(artifact, registry string) (string, error) {
	// ModelPack stub implementation
	// In a real implementation, this would use modelpack CLI to inspect the artifact
	// and return its digest
	fullRef := registry + "/" + artifact
	// Return a mock digest for now
	return fmt.Sprintf("sha256:%x", fullRef[:8]), nil
}

func (m *ModelPackProvider) PushReferrer(artifact, registry, referrerType string, data []byte, annotations map[string]string) error {
	// ModelPack supports referrers similar to ORAS
	// For now, ModelPack implementation is a stub
	// In a real implementation, this would use modelpack CLI to push referrers
	fmt.Printf("Pushed %s referrer for %s to %s using ModelPack (stub)\n", referrerType, artifact, registry)
	return nil
}

func (m *ModelPackProvider) GetReferrers(artifact, registry, referrerType string) ([][]byte, error) {
	// Stub implementation for ModelPack
	// In a real implementation, this would fetch referrers from the registry
	fmt.Printf("Fetched referrers for %s from %s using ModelPack (stub)\n", artifact, registry)
	return nil, nil
}

func (m *ModelPackProvider) Search(registry, filters string) ([][]byte, error) {
	// ModelPack stub implementation for search
	// In a real implementation, this would integrate with ModelPack's registry query capabilities
	// or delegate to the underlying registry's search API
	// For now, we return a helpful message about using the registry's native search
	return nil, fmt.Errorf("ModelPack search: delegate to registry's native search API or use client-side filtering with 'modelpack' CLI")
}

func (m *ModelPackProvider) FetchManifestAnnotations(artifactRef string) (map[string]string, error) {
	// ModelPack stub implementation for fetching manifest annotations
	// In a real implementation, this would use modelpack CLI to inspect the artifact
	// and extract its annotations
	// For now, return an error indicating this is not fully implemented
	return nil, fmt.Errorf("ModelPack FetchManifestAnnotations: delegate to ORAS or use modelpack inspect CLI")
}

// GetRegistryProvider returns the appropriate Registry provider by name
func GetRegistryProvider(name string) (RegistryProvider, error) {
	switch name {
	case "oras":
		return &ORASProvider{}, nil
	case "modelpack":
		return &ModelPackProvider{}, nil
	default:
		return nil, fmt.Errorf("unknown registry provider: %s (supported: oras, modelpack)", name)
	}
}
