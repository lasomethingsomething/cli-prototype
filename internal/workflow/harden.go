package workflow

import (
	"fmt"
	"path/filepath"
)

// HardenWorkflow orchestrates the local hardening and compliance checks
type HardenWorkflow struct {
	// Inputs
	modelName    string
	modelPath    string
	artifactName string
	registry    string

	// Options
	generateSBOM bool
	includeMOF   bool

	// Results
	sbomPath     string
	mofClass     string
	annotations  *AnnotationSet

	// State
	workflowErr error
}

// NewHardenWorkflow creates a new hardening workflow
func NewHardenWorkflow(registry string) *HardenWorkflow {
	return &HardenWorkflow{
		registry:    registry,
		generateSBOM: true,
		includeMOF:   true,
		annotations:  NewAnnotationSet(),
	}
}

// SetHardenInfo sets the hardening details
func (w *HardenWorkflow) SetHardenInfo(modelName, modelPath, artifactName string) {
	w.modelName = modelName
	w.modelPath = modelPath
	w.artifactName = artifactName
}

// SetOptions configures hardening options
func (w *HardenWorkflow) SetOptions(generateSBOM, includeMOF bool) {
	w.generateSBOM = generateSBOM
	w.includeMOF = includeMOF
}

// SetAnnotations sets the annotation set to update
func (w *HardenWorkflow) SetAnnotations(annotations *AnnotationSet) {
	w.annotations = annotations
}

// Run executes the hardening workflow
func (w *HardenWorkflow) Run() error {
	fmt.Printf("Hardening artifact '%s' from '%s'\n", w.modelName, w.modelPath)
	fmt.Println()

	// Step 1: Generate SBOM if requested
	if w.generateSBOM {
		fmt.Println("→ Generating SBOM (Software Bill of Materials)...")
		
		sbomGen, err := GetSBOMGenerator("syft")
		if err != nil {
			fmt.Printf("  Warning: SBOM generator not available: %v\n", err)
		} else {
			w.sbomPath = filepath.Join(w.modelPath, "sbom.spdx.json")
			if err := sbomGen.Generate(w.modelPath, w.sbomPath); err != nil {
				w.workflowErr = fmt.Errorf("SBOM generation failed: %v", err)
				fmt.Printf("  ✗ SBOM generation failed: %v\n", err)
			} else {
				fmt.Printf("  ✓ SBOM generated: %s\n", w.sbomPath)
				// Add SBOM annotation
				if w.annotations != nil {
					w.annotations.SBOMFormat = "spdx-json"
				}
			}
		}
		fmt.Println()
	}

	// Step 2: Apply MOF classification if requested
	if w.includeMOF {
		fmt.Println("→ Applying MOF (Model Openness Framework) classification...")
		
		mofClassifier := GetMOFClassifier()
		var err error
		w.mofClass, err = mofClassifier.Classify(w.modelPath)
		if err != nil {
			w.workflowErr = fmt.Errorf("MOF classification failed: %v", err)
			fmt.Printf("  ✗ MOF classification failed: %v\n", err)
		} else {
			fmt.Printf("  ✓ MOF Class: %s\n", w.mofClass)
			// Add MOF annotations
			if w.annotations != nil {
				w.annotations.MOFClass = w.mofClass
				w.annotations.MOFVersion = "1.0"
			}
		}
		fmt.Println()
	}

	// Step 3: Apply security annotations
	if w.annotations != nil {
		fmt.Println("→ Applying security annotations...")
		w.annotations.SigningFramework = "sigstore-cosign"
		w.annotations.ProvenanceType = "slsa-v1.0"
		fmt.Println("  ✓ Security annotations applied")
		fmt.Println()
	}

	if w.workflowErr != nil {
		return w.workflowErr
	}

	fmt.Println("✓ Hardening & Compliance checks complete")
	return nil
}

// SBOMPath returns the path to the generated SBOM
func (w *HardenWorkflow) SBOMPath() string {
	return w.sbomPath
}

// MOFClass returns the MOF classification result
func (w *HardenWorkflow) MOFClass() string {
	return w.mofClass
}

// Annotations returns the updated annotation set
func (w *HardenWorkflow) Annotations() *AnnotationSet {
	return w.annotations
}

// GetHardeningProvider is a factory for hardening providers (future extensibility)
func GetHardeningProvider(name string) (interface{}, error) {
	// Currently only one provider (local hardening)
	return nil, fmt.Errorf("hardening provider not implemented")
}
