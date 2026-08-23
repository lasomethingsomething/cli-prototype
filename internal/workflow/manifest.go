package workflow

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
)

// OCI media types used for the manifest and config produced during packaging.
// See: https://github.com/opencontainers/image-spec/blob/main/manifest.md
const (
	ManifestMediaType = "application/vnd.oci.image.manifest.v1+json"
	ConfigMediaType   = "application/vnd.cncf.ai.model.config.v1+json"
)

// Descriptor is a minimal OCI content descriptor.
type Descriptor struct {
	MediaType string `json:"mediaType"`
}

// Manifest is a minimal OCI image manifest that carries the CNCF AI
// Interoperability Profile annotations produced by an AnnotationSet.
type Manifest struct {
	SchemaVersion int               `json:"schemaVersion"`
	MediaType     string            `json:"mediaType"`
	Config        Descriptor        `json:"config"`
	Annotations   map[string]string `json:"annotations,omitempty"`
}

// NewManifest builds an OCI manifest with the given annotations attached at
// the manifest level.
func NewManifest(annotations map[string]string) *Manifest {
	return &Manifest{
		SchemaVersion: 2,
		MediaType:     ManifestMediaType,
		Config:        Descriptor{MediaType: ConfigMediaType},
		Annotations:   annotations,
	}
}

// WriteManifest marshals the manifest as indented JSON and writes it to path.
func WriteManifest(m *Manifest, path string) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal OCI manifest: %v", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write OCI manifest to %s: %v", path, err)
	}
	return nil
}

// ReadManifest reads and parses an OCI manifest JSON file from path.
func ReadManifest(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read OCI manifest from %s: %v", path, err)
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("failed to parse OCI manifest at %s: %v", path, err)
	}
	return &m, nil
}

// ComputeManifestDigest computes a SHA256 digest of the manifest file at the given path
// This is used for local parity verification to ensure the local artifact matches
// what was pushed to the registry
func ComputeManifestDigest(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	hash := sha256.Sum256(data)
	return fmt.Sprintf("sha256:%x", hash)
}
