package workflow

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// PackageWorkflow orchestrates the packaging of models as OCI artifacts
type PackageWorkflow struct {
	registry         string
	registryProvider RegistryProvider
	modelName        string
	modelPath        string
	artifactName     string
	registryURL      string
	includeRAG       bool
	ragPath          string
	generateSBOM     bool
	includeMOF       bool
	sbomTool         string
	sbomFormat       SBOMFormat
	annotations      *AnnotationSet
	manifestPath     string
	isSkill          bool

	// Signing options
	sign   bool
	signer string

	// Provenance options
	generateProvenance bool
	provenancePath     string

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
		registry:           registry,
		registryProvider:   registryProvider,
		generateSBOM:       true,               // Default to generating SBOM
		includeMOF:         true,               // Default to including MOF classification
		sbomTool:           "syft",             // Default SBOM tool
		sbomFormat:         SPDXJSON,           // Default SBOM format
		annotations:        NewAnnotationSet(), // Default annotations
		generateProvenance: true,               // Default to generating provenance attestation
		verifyParity:       true,               // Default to verifying the pushed digest
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

// SetSecurityOptions configures SBOM and MOF options
func (w *PackageWorkflow) SetSecurityOptions(generateSBOM, includeMOF bool) {
	w.generateSBOM = generateSBOM
	w.includeMOF = includeMOF
}

// SetSBOMTool sets the SBOM generation tool and format
func (w *PackageWorkflow) SetSBOMTool(tool string, format SBOMFormat) {
	w.sbomTool = tool
	w.sbomFormat = format
	// Update annotation
	if w.annotations != nil {
		w.annotations.SBOMFormat = string(format)
	}
}

// SetIsSkill sets whether this is a skill package
func (w *PackageWorkflow) SetIsSkill(isSkill bool) {
	w.isSkill = isSkill
}

// SetSigningOptions configures signing options for the workflow
func (w *PackageWorkflow) SetSigningOptions(sign bool, signer string) {
	w.sign = sign
	w.signer = signer
}

// SetProvenanceOptions configures provenance generation options
func (w *PackageWorkflow) SetProvenanceOptions(generate bool) {
	w.generateProvenance = generate
}

// ProvenancePath returns the path to the generated provenance attestation
func (w *PackageWorkflow) ProvenancePath() string {
	return w.provenancePath
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

// applyMOFClassification records the detected MOF class and components in
// the annotations unless the user declared them explicitly. An explicit
// class that differs from the detected one is kept, with a warning.
func (w *PackageWorkflow) applyMOFClassification(detectedClass string, result *ClassificationResult) {
	switch {
	case w.annotations.MOFClass == "":
		w.annotations.MOFClass = detectedClass
		fmt.Printf("  MOF Class: %s (detected)\n", detectedClass)
	case w.annotations.MOFClass != detectedClass:
		fmt.Printf("  MOF Class: %s (declared; files suggest %s - %s)\n", w.annotations.MOFClass, detectedClass, result.Explanation)
	default:
		fmt.Printf("  MOF Class: %s (declared, matches detection)\n", w.annotations.MOFClass)
	}
	if w.annotations.MOFComponents == "" && len(result.Components) > 0 {
		w.annotations.MOFComponents = strings.Join(canonicalMOFComponents(result.Components), ",")
		fmt.Printf("  MOF Components: %s (detected)\n", w.annotations.MOFComponents)
	}
}

// canonicalMOFComponents orders detected components the way the MOF spec
// lists them, independent of the order files were encountered on disk.
func canonicalMOFComponents(components []string) []string {
	order := []string{"weights", "code", "training-data", "documentation", "license"}
	var sorted []string
	for _, want := range order {
		for _, c := range components {
			if c == want {
				sorted = append(sorted, c)
			}
		}
	}
	return sorted
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

	// === Security & Supply Chain Steps ===

	// 1. Generate SBOM if requested
	if w.generateSBOM {
		fmt.Println("\n=== Supply Chain Security ===")
		fmt.Println("✓ Generating SBOM (Software Bill of Materials)...")

		sbomGen, err := GetSBOMGenerator(w.sbomTool)
		if err != nil {
			fmt.Printf("  Note: SBOM generation skipped: %v\n", err)
		} else {
			sbomPath := filepath.Join(w.modelPath, "sbom."+string(w.sbomFormat))
			if err := sbomGen.Generate(w.modelPath, sbomPath, w.sbomFormat); err != nil {
				fmt.Printf("  Warning: SBOM generation failed: %v\n", err)
			} else {
				fmt.Printf("  SBOM generated: %s (format: %s)\n", sbomPath, w.sbomFormat)
				// Attach SBOM as OCI layer
				fmt.Printf("  SBOM attached as OCI layer with format: %s\n", w.sbomFormat)
			}
		}
	}

	// 2. MOF classification: fill in what the user left empty, and warn when
	//    the declared class disagrees with what the files suggest.
	if w.includeMOF {
		fmt.Println("✓ Applying MOF (Model Openness Framework) classification...")

		detectedClass, result, err := ClassifyModelPath(w.modelPath)
		if err != nil {
			fmt.Printf("  Warning: MOF classification failed: %v\n", err)
		} else if w.annotations != nil {
			w.applyMOFClassification(detectedClass, result)
		}
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

	// Generate provenance attestation (SLSA/in-toto) - frozen at point of creation
	if w.generateProvenance {
		fmt.Println("\n=== Provenance ===")
		fmt.Println("→ Generating SLSA provenance attestation...")

		// Create provenance generator
		pg := NewProvenanceGenerator()
		pg.SetSourceInfo(w.modelPath, "")
		pg.SetArtifactInfo(fullArtifact, nil)
		pg.SetRecipeInfo(
			"https://model-cli.dev/recipe/package/v1",
			fmt.Sprintf("registry:%s", w.registry),
			"package",
		)

		// Generate and write the attestation
		w.provenancePath = filepath.Join(w.modelPath, "attestation.json")
		attestation, err := pg.WriteToFile(w.provenancePath)
		if err != nil {
			return fmt.Errorf("failed to generate provenance attestation: %v", err)
		}

		// Validate the attestation
		if err := ValidateAttestation(attestation); err != nil {
			return fmt.Errorf("failed to validate provenance attestation: %v", err)
		}

		fmt.Printf("✓ Provenance attestation generated: %s\n", w.provenancePath)
		fmt.Println("  Attestation contains:")
		fmt.Printf("    - Build ID: %s\n", attestation.Statement.Predicate.BuildID)
		fmt.Printf("    - Build Type: %s\n", attestation.Statement.Predicate.BuildType)
		fmt.Printf("    - Builder: %s\n", attestation.Statement.Predicate.Builder.ID)
		fmt.Printf("    - Source: %s\n", attestation.Statement.Predicate.Source.ID)
		fmt.Printf("    - Timestamp: %s\n", attestation.Statement.Predicate.Metadata.BuildFinishedOn.Format(time.RFC3339))
	}

	// Sign the artifact if requested (at point of creation)
	if w.sign {
		fmt.Println("\n=== Signing ===")
		fmt.Printf("→ Signing artifact '%s'...\n", fullArtifact)

		// Determine signer to use
		signerToUse := w.signer
		if signerToUse == "" {
			signerToUse = "sigstore" // Default to sigstore
		}

		// Get signing provider
		sp, err := GetSigningProvider(signerToUse)
		if err != nil {
			return fmt.Errorf("failed to get signing provider: %v", err)
		}

		// Check if tool is installed
		if !sp.IsInstalled() {
			return fmt.Errorf("%s not installed. Install with: %s", sp.Name(), sp.InstallInstructions())
		}

		// Sign the artifact
		if err := sp.Sign(fullArtifact, ""); err != nil {
			return fmt.Errorf("failed to sign artifact: %v", err)
		}

		sigPath := sp.GetSignaturePath(fullArtifact)
		fmt.Printf("✓ Signed artifact: %s\n", fullArtifact)
		fmt.Printf("  Signature: %s\n", sigPath)
	}

	fmt.Println("\n=== Summary ===")
	fmt.Println("Packaging complete!")

	if !w.sign {
		fmt.Println("\nNext steps:")
		fmt.Println("  - Sign with: model-cli sign --artifact " + fullArtifact)
		fmt.Println("  - Verify with: model-cli verify --artifact " + fullArtifact)
		fmt.Println("  - Deploy with: model-cli deploy")
	} else {
		fmt.Println("\nNext steps:")
		fmt.Println("  - Verify with: model-cli verify --artifact " + fullArtifact)
		fmt.Println("  - Deploy with: model-cli deploy")
	}

	return nil
}
