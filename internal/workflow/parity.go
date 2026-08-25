package workflow

import (
	"fmt"
)

// LocalParityResult represents the result of a local parity verification
type LocalParityResult struct {
	LocalDigest    string
	RegistryDigest string
	Artifact       string
	Registry       string
	Match          bool
}

// String returns a human-readable representation of the parity result
func (r *LocalParityResult) String() string {
	if r.Match {
		return fmt.Sprintf("✓ Local parity VERIFIED: %s matches in registry %s", r.Artifact, r.Registry)
	}
	return fmt.Sprintf("✗ Local parity FAILED: %s does not match in registry %s (local: %s, registry: %s)",
		r.Artifact, r.Registry, r.LocalDigest, r.RegistryDigest)
}

// LocalParityVerifier checks that a local artifact matches what's in the registry
type LocalParityVerifier struct {
	registryProvider RegistryProvider
	localDigest      string
}

// NewLocalParityVerifier creates a new local parity verifier
func NewLocalParityVerifier(registryProvider RegistryProvider, localDigest string) *LocalParityVerifier {
	return &LocalParityVerifier{
		registryProvider: registryProvider,
		localDigest:      localDigest,
	}
}

// Verify compares the local artifact digest with the registry version
func (v *LocalParityVerifier) Verify(artifact, registry string) (*LocalParityResult, error) {
	// Get the digest from the registry
	registryDigest, err := v.registryProvider.GetArtifactDigest(artifact, registry)
	if err != nil {
		return nil, fmt.Errorf("failed to get artifact digest from registry: %v", err)
	}

	// Compare digests
	match := v.localDigest == registryDigest

	return &LocalParityResult{
		LocalDigest:    v.localDigest,
		RegistryDigest: registryDigest,
		Artifact:       artifact,
		Registry:       registry,
		Match:          match,
	}, nil
}

// VerifyWithDigest is a convenience function that creates a verifier and verifies in one step
func VerifyWithDigest(registryProvider RegistryProvider, localDigest, artifact, registry string) (*LocalParityResult, error) {
	verifier := NewLocalParityVerifier(registryProvider, localDigest)
	return verifier.Verify(artifact, registry)
}
