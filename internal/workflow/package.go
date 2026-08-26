package workflow

import (
	"fmt"
	"path/filepath"
)

// PackageWorkflow orchestrates the packaging of models as OCI artifacts:
// manifest, layers, push and parity check. SBOM generation and MOF
// classification are deliberately not part of it; they are the separate
// hardening step (HardenWorkflow) that runs on the packaged model. Signing
// and SLSA provenance are also deliberately not part of it; they are the
// separate Phase 1, Step 3 (Supply Chain Check) that runs with `model-cli
// sign` on the packaged artifact.
type PackageWorkflow struct {
	registry         string
	registryProvider RegistryProvider
	modelName        string
	modelPath        string
	artifactName     string
	registryURL      string
	includeRAG       bool
	ragPath          string
	annotations      *AnnotationSet
	manifestPath     string
	isSkill          bool

	// Local parity verification
	localDigest  string // digest of the manifest written locally
	pushedDigest string // digest the registry tool reported after pushing
	verifyParity bool
}

// NewPackageWorkflow creates a new packaging workflow
func NewPackageWorkflow(registry string) (*PackageWorkflow, error) {
	registryProvider, err := GetRegistryProvider(registry)
	if err != nil {
		return nil, err
	}

	return &PackageWorkflow{
		registry:         registry,
		registryProvider: registryProvider,
		annotations:      NewAnnotationSet(), // Default annotations
		verifyParity:     true,               // Default to verifying the pushed digest
	}, nil
}

// SetPackageInfo sets the packaging details
func (w *PackageWorkflow) SetPackageInfo(modelName, modelPath, artifactName, registryURL string, includeRAG bool, ragPath string) {
	w.modelName = modelName
	w.modelPath = modelPath
	w.artifactName = artifactName
	w.registryURL = registryURL
	w.includeRAG = includeRAG
	w.ragPath = ragPath
}

// SetAnnotations sets the CNCF AI Interoperability Profile annotations
func (w *PackageWorkflow) SetAnnotations(annotations *AnnotationSet) {
	w.annotations = annotations
}

// SetIsSkill sets whether this is a skill package
func (w *PackageWorkflow) SetIsSkill(isSkill bool) {
	w.isSkill = isSkill
}

// ManifestPath returns the path to the OCI manifest written by Run(), or an
// empty string if Run() has not been called yet.
func (w *PackageWorkflow) ManifestPath() string {
	return w.manifestPath
}

// LocalDigest returns the digest of the manifest written locally by Run().
func (w *PackageWorkflow) LocalDigest() string {
	return w.localDigest
}

// PushedDigest returns the manifest digest reported by the registry tool
// after pushing, or "" when nothing was pushed or the tool did not report one.
func (w *PackageWorkflow) PushedDigest() string {
	return w.pushedDigest
}

// SetLocalDigest sets the digest of the local artifact for parity verification
func (w *PackageWorkflow) SetLocalDigest(digest string) {
	w.localDigest = digest
}

// SetVerifyParity enables or disables local parity verification
func (w *PackageWorkflow) SetVerifyParity(verify bool) {
	w.verifyParity = verify
}

// Run executes the packaging workflow
func (w *PackageWorkflow) Run() error {
	fmt.Printf("Packaging model '%s' from '%s' as '%s'\n", w.modelName, w.modelPath, w.artifactName)

	// Check if registry provider is available
	if !w.registryProvider.IsInstalled() {
		return fmt.Errorf("%s not installed. Install with: %s", w.registryProvider.Name(), w.registryProvider.InstallInstructions())
	}

	fullArtifact := w.artifactName
	if w.registryURL != "" {
		fullArtifact = w.registryURL + "/" + w.artifactName
	}

	// === Packaging Steps ===

	fmt.Println("\n=== Packaging ===")

	// Display annotations that will be included
	if w.annotations != nil {
		w.annotations.Print()
		fmt.Println()
	}

	// Create OCI artifact manifest
	fmt.Println("→ Creating OCI artifact manifest...")

	// Inject CNCF AI Interoperability Profile annotations into the manifest
	// and write it to disk alongside the packaged model files.
	fmt.Println("→ Injecting CNCF AI Interoperability Profile annotations...")

	var manifestAnnotations map[string]string
	if w.annotations != nil {
		manifestAnnotations = w.annotations.ToMap()
	} else {
		manifestAnnotations = make(map[string]string)
	}

	artifactType := ArtifactTypeModel
	if w.isSkill {
		artifactType = ArtifactTypeSkill
	} else if t := manifestAnnotations[AnnotationArtifactType]; t != "" {
		artifactType = ArtifactType(t)
	}

	manifest, err := NewManifestFromDirectory(artifactType, w.artifactName, w.modelPath, manifestAnnotations)
	if err != nil {
		return fmt.Errorf("failed to build OCI manifest: %v", err)
	}
	manifestAnnotations = manifest.Annotations
	w.manifestPath = filepath.Join(w.modelPath, "manifest.json")
	if err := WriteUnifiedOCIManifest(manifest, w.manifestPath); err != nil {
		return fmt.Errorf("failed to write OCI manifest: %v", err)
	}
	fmt.Printf("  Manifest written to: %s (%d layer(s))\n", w.manifestPath, len(manifest.Layers))

	w.localDigest = ComputeManifestDigest(w.manifestPath)

	if w.includeRAG && w.ragPath != "" {
		fmt.Printf("→ Adding RAG context from '%s'...\n", w.ragPath)
	}

	// Package model files
	fmt.Printf("→ Packaging model files from '%s'...\n", w.modelPath)
	fmt.Printf("→ Packaged as OCI artifact: %s\n", fullArtifact)

	// Push to registry if URL is provided
	if w.registryURL != "" {
		fmt.Printf("→ Pushing to registry '%s'...\n", w.registryURL)
		digest, err := w.registryProvider.Push(w.artifactName, w.registryURL, w.modelPath, manifestAnnotations)
		if err != nil {
			return fmt.Errorf("failed to push artifact: %v", err)
		}
		w.pushedDigest = digest
		fmt.Printf("✓ Successfully pushed %s to %s\n", fullArtifact, w.registryURL)
		if digest != "" {
			fmt.Printf("  Digest: %s\n", digest)
		}

		// Verify parity: the manifest stored in the registry must be the one we just pushed.
		if w.verifyParity {
			fmt.Println("\n=== Local Parity Verification ===")
			if digest == "" {
				fmt.Printf("  Skipped: %s did not report a digest for the pushed artifact\n", w.registryProvider.Name())
			} else {
				fmt.Println("→ Verifying that the registry holds the pushed manifest...")
				verifier := NewLocalParityVerifier(w.registryProvider, digest)
				result, err := verifier.Verify(w.artifactName, w.registryURL)
				if err != nil {
					return fmt.Errorf("failed to verify local parity: %v", err)
				}
				fmt.Println(result.String())
				if !result.Match {
					return fmt.Errorf("local parity check failed: %s", result.String())
				}
			}
		}
	} else {
		fmt.Println("✓ Saved locally (not pushed to registry)")
		fmt.Printf("  Artifact ready at: %s\n", fullArtifact)
	}

	// Signing and provenance are not part of packaging. They are Phase 1,
	// Step 3 (Supply Chain Check), run afterwards with `model-cli sign`,
	// which signs the finished artifact and attests it.
	fmt.Println("\n=== Summary ===")
	fmt.Println("Packaging complete!")

	fmt.Println("\nNext steps:")
	fmt.Printf("  - Harden (SBOM + MOF) with: model-cli harden --model %s --model-path %s --artifact %s\n", w.modelName, w.modelPath, w.artifactName)
	fmt.Println("  - Sign and record provenance with: model-cli sign --artifact " + fullArtifact)
	fmt.Println("  - Verify with: model-cli verify --artifact " + fullArtifact)
	fmt.Println("  - Deploy with: model-cli deploy")

	return nil
}
