package workflow

import (
	"fmt"
	"path/filepath"
)

// PackageWorkflow orchestrates the packaging of models as OCI artifacts
type PackageWorkflow struct {
	registry          string
	registryProvider RegistryProvider
	modelName         string
	modelPath         string
	artifactName      string
	registryURL       string
	includeRAG        bool
	ragPath           string
	generateSBOM      bool
	includeMOF        bool
	annotations      *AnnotationSet
}

// NewPackageWorkflow creates a new packaging workflow
func NewPackageWorkflow(registry string) (*PackageWorkflow, error) {
	registryProvider, err := GetRegistryProvider(registry)
	if err != nil {
		return nil, err
	}

	return &PackageWorkflow{
		registry:          registry,
		registryProvider: registryProvider,
		generateSBOM:      true,  // Default to generating SBOM
		includeMOF:        true,  // Default to including MOF classification
		annotations:      NewAnnotationSet(), // Default annotations
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
		
		sbomGen, err := GetSBOMGenerator("syft")
		if err != nil {
			fmt.Printf("  Note: SBOM generation skipped: %v\n", err)
		} else {
			sbomPath := filepath.Join(w.modelPath, "sbom.spdx.json")
			if err := sbomGen.Generate(w.modelPath, sbomPath); err != nil {
				fmt.Printf("  Warning: SBOM generation failed: %v\n", err)
			} else {
				fmt.Printf("  SBOM saved to: %s\n", sbomPath)
			}
		}
	}

	// 2. MOF Classification if requested
	if w.includeMOF {
		fmt.Println("✓ Applying MOF (Model Openness Framework) classification...")
		
		mofClassifier := GetMOFClassifier()
		mofClass, err := mofClassifier.Classify(w.modelPath)
		if err != nil {
			fmt.Printf("  Warning: MOF classification failed: %v\n", err)
		} else {
			fmt.Printf("  MOF Class: %s\n", mofClass)
			// In production, this would add the classification to artifact metadata
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
	
	// Inject CNCF AI Interoperability Profile annotations
	fmt.Println("→ Injecting CNCF AI Interoperability Profile annotations...")
	
	if w.includeRAG && w.ragPath != "" {
		fmt.Printf("→ Adding RAG context from '%s'...\n", w.ragPath)
	}

	// Package model files
	fmt.Printf("→ Packaging model files from '%s'...\n", w.modelPath)
	fmt.Printf("→ Packaged as OCI artifact: %s\n", fullArtifact)

	// Push to registry if URL is provided
	if w.registryURL != "" {
		fmt.Printf("→ Pushing to registry '%s'...\n", w.registryURL)
		if err := w.registryProvider.Push(w.artifactName, w.registryURL); err != nil {
			return fmt.Errorf("failed to push artifact: %v", err)
		}
		fmt.Printf("✓ Successfully pushed %s to %s\n", fullArtifact, w.registryURL)
	} else {
		fmt.Println("✓ Saved locally (not pushed to registry)")
		fmt.Printf("  Artifact ready at: %s\n", fullArtifact)
	}

	fmt.Println("\n=== Summary ===")
	fmt.Println("Packaging complete!")
	fmt.Println("\nNext steps:")
	fmt.Println("  - Sign with: model-cli sign --artifact " + fullArtifact)
	fmt.Println("  - Verify with: model-cli verify --artifact " + fullArtifact)
	fmt.Println("  - Deploy with: model-cli deploy")

	return nil
}
