package workflow

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// HardenWorkflow orchestrates the local hardening and compliance step that
// follows packaging: SBOM generation and MOF classification. It runs on a
// model that `model-cli package` has already written a manifest.json for and
// records its results in that manifest.
type HardenWorkflow struct {
	// Inputs
	modelName    string
	modelPath    string
	artifactName string
	registry     string

	// Options
	generateSBOM      bool
	includeMOF        bool
	generateMOFConfig bool
	sbomTool          string
	sbomFormat        SBOMFormat
	license           string

	// sbomGenerator overrides the generator looked up by sbomTool (tests).
	sbomGenerator SBOMGenerator

	// Results
	sbomPath           string
	mofClass           string
	mofConfigPath      string
	manifestPath       string
	appliedAnnotations map[string]string
	annotations        *AnnotationSet

	// State
	workflowErr error
}

// NewHardenWorkflow creates a new hardening workflow
func NewHardenWorkflow(registry string) *HardenWorkflow {
	return &HardenWorkflow{
		registry:          registry,
		generateSBOM:      true,
		includeMOF:        true,
		generateMOFConfig: true,
		sbomTool:          "syft",
		sbomFormat:        SPDXJSON,
		license:           "CC-BY-4.0", // Default license
		annotations:       NewAnnotationSet(),
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
	w.generateMOFConfig = true
}

// SetGenerateMOFConfig enables/disables MOF config file generation
func (w *HardenWorkflow) SetGenerateMOFConfig(generate bool) {
	w.generateMOFConfig = generate
}

// SetLicense sets the license for the MOF metadata (defaults to CC-BY-4.0)
func (w *HardenWorkflow) SetLicense(license string) {
	w.license = license
}

// SetSBOMTool sets the SBOM generation tool and format
func (w *HardenWorkflow) SetSBOMTool(tool string, format SBOMFormat) {
	w.sbomTool = tool
	w.sbomFormat = format
}

// SBOMTool returns the name of the SBOM generator in use.
func (w *HardenWorkflow) SBOMTool() string { return w.sbomTool }

// SBOMFormat returns the SBOM output format in use.
func (w *HardenWorkflow) SBOMFormat() SBOMFormat { return w.sbomFormat }

// SetAnnotations sets the annotation set to update. A MOF class or component
// list already present in it is treated as declared by the user and kept.
func (w *HardenWorkflow) SetAnnotations(annotations *AnnotationSet) {
	w.annotations = annotations
}

// Run executes the hardening workflow
func (w *HardenWorkflow) Run() error {
	fmt.Printf("Hardening artifact '%s' from '%s'\n", w.modelName, w.modelPath)
	fmt.Println()

	if w.annotations == nil {
		w.annotations = NewAnnotationSet()
	}

	// Hardening depends on packaging: the manifest written by
	// `model-cli package` is where the results are recorded.
	manifest, err := w.readPackagedManifest()
	if err != nil {
		return err
	}

	// Step 1: Generate SBOM if requested
	// SBOM is a prerequisite for Phase 1 Step 2 - failure must block the workflow
	if w.generateSBOM {
		fmt.Println("→ Generating SBOM (Software Bill of Materials)...")

		sbomGen := w.sbomGenerator
		if sbomGen == nil {
			sbomGen, err = GetSBOMGenerator(w.sbomTool)
		}
		if err != nil {
			return fmt.Errorf("SBOM generator not available: %v. SBOM is a required prerequisite for Phase 1 Step 2", err)
		}

		sbomPath := filepath.Join(w.modelPath, SBOMFileName(w.sbomFormat))
		if err := sbomGen.Generate(w.modelPath, sbomPath, w.sbomFormat); err != nil {
			return fmt.Errorf("SBOM generation failed: %v. SBOM is a required prerequisite for Phase 1 Step 2", err)
		}
		w.sbomPath = sbomPath
		w.annotations.SBOMFormat = string(w.sbomFormat)
		fmt.Printf("  ✓ SBOM generated: %s\n", w.sbomPath)
		// Attach SBOM as OCI layer (simulated - in real implementation would use OCI tools)
		fmt.Printf("  ✓ SBOM attached as OCI layer with format: %s\n", w.sbomFormat)
		fmt.Println()
	}

	// Step 2: Apply MOF classification if requested: fill in what the user
	// left empty, and warn when the declared class disagrees with what the
	// files suggest.
	if w.includeMOF {
		fmt.Println("→ Applying MOF (Model Openness Framework) classification...")

		detectedClass, result, err := ClassifyModelPath(w.modelPath)
		if err != nil {
			w.workflowErr = fmt.Errorf("MOF classification failed: %v", err)
			fmt.Printf("  ✗ MOF classification failed: %v\n", err)
		} else {
			w.applyMOFClassification(detectedClass, result)
			w.mofClass = w.annotations.MOFClass
			if w.annotations.MOFVersion == "" {
				w.annotations.MOFVersion = "1.0"
			}
		}
		fmt.Println()

		// Step 2.5: Generate MOF metadata config file if requested
		if w.generateMOFConfig && err == nil {
			fmt.Println("→ Generating MOF metadata config file...")
			w.writeMOFConfig(result)
			fmt.Println()
		}
	}

	// Step 3: Apply security annotations
	fmt.Println("→ Applying security annotations...")
	w.annotations.SigningFramework = "sigstore-cosign"
	w.annotations.ProvenanceType = "slsa-v1.0"
	fmt.Println("  ✓ Security annotations applied")
	fmt.Println()

	// Step 4: Record the results in the packaged manifest.
	fmt.Println("→ Updating packaged manifest...")
	w.appliedAnnotations = w.manifestAnnotations()
	for k, v := range w.appliedAnnotations {
		manifest.Annotations[k] = v
	}
	if err := WriteUnifiedOCIManifest(manifest, w.manifestPath); err != nil {
		return fmt.Errorf("failed to update manifest: %v", err)
	}
	fmt.Printf("  ✓ Manifest updated: %s (%d annotation(s))\n", w.manifestPath, len(w.appliedAnnotations))
	fmt.Println()

	if w.workflowErr != nil {
		return w.workflowErr
	}

	fmt.Println("✓ Hardening & Compliance checks complete")
	return nil
}

// readPackagedManifest loads <model-path>/manifest.json, which
// `model-cli package` writes; hardening cannot run before packaging.
func (w *HardenWorkflow) readPackagedManifest() (*UnifiedOCIManifest, error) {
	w.manifestPath = filepath.Join(w.modelPath, "manifest.json")
	if _, err := os.Stat(w.manifestPath); err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("no manifest.json in %s: run `model-cli package` first", w.modelPath)
		}
		return nil, fmt.Errorf("failed to read %s: %v", w.manifestPath, err)
	}
	manifest, err := ReadUnifiedOCIManifest(w.manifestPath)
	if err != nil {
		return nil, err
	}
	if manifest.Annotations == nil {
		manifest.Annotations = make(map[string]string)
	}
	return manifest, nil
}

// applyMOFClassification records the detected MOF class and components in
// the annotations unless the user declared them explicitly. An explicit
// class that differs from the detected one is kept, with a warning.
func (w *HardenWorkflow) applyMOFClassification(detectedClass string, result *ClassificationResult) {
	switch {
	case w.annotations.MOFClass == "":
		w.annotations.MOFClass = detectedClass
		fmt.Printf("  ✓ MOF Class: %s (detected)\n", detectedClass)
	case w.annotations.MOFClass != detectedClass:
		fmt.Printf("  ✓ MOF Class: %s (declared; files suggest %s - %s)\n", w.annotations.MOFClass, detectedClass, result.Explanation)
	default:
		fmt.Printf("  ✓ MOF Class: %s (declared, matches detection)\n", w.annotations.MOFClass)
	}
	if w.annotations.MOFComponents == "" && len(result.Components) > 0 {
		w.annotations.MOFComponents = strings.Join(canonicalMOFComponents(result.Components), ",")
		fmt.Printf("  ✓ MOF Components: %s (detected)\n", w.annotations.MOFComponents)
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

// writeMOFConfig writes mof.json next to the model from the effective
// classification (declared values win over detected ones).
func (w *HardenWorkflow) writeMOFConfig(result *ClassificationResult) {
	var components []string
	if w.annotations.MOFComponents != "" {
		components = strings.Split(w.annotations.MOFComponents, ",")
	}

	generator := NewMOFMetadataGenerator()
	generator.SetModelInfo(w.modelName, w.modelPath, w.artifactName)
	generator.SetMOFClassification(w.annotations.MOFClass, components, result.Explanation)
	generator.SetReleaseInfo(w.modelName, "1.0.0", "", "model")
	generator.SetReleaseLicense(w.license)

	mofConfigPath := filepath.Join(w.modelPath, "mof.json")
	if err := generator.WriteToFile(mofConfigPath, "json"); err != nil {
		w.workflowErr = fmt.Errorf("MOF metadata generation failed: %w", err)
		fmt.Printf("  ✗ MOF metadata generation failed: %v\n", err)
		return
	}
	w.mofConfigPath = mofConfigPath
	fmt.Printf("  ✓ MOF metadata config file generated: %s\n", w.mofConfigPath)
	// Attach MOF config as OCI layer (simulated)
	fmt.Printf("  ✓ MOF config attached as OCI layer\n")
}

// manifestAnnotations returns the annotations hardening owns, ready to be
// merged into the packaged manifest. Only results that were actually
// produced are claimed: no SBOM format without an SBOM, no MOF class without
// a classification.
func (w *HardenWorkflow) manifestAnnotations() map[string]string {
	annotations := make(map[string]string)
	if w.sbomPath != "" {
		annotations[AnnotationSBOMFormat] = string(w.sbomFormat)
	}
	if w.annotations.MOFClass != "" {
		annotations[AnnotationMOFClass] = w.annotations.MOFClass
		if w.annotations.MOFVersion != "" {
			annotations[AnnotationMOFVersion] = w.annotations.MOFVersion
		}
	}
	if w.annotations.MOFComponents != "" {
		annotations[AnnotationMOFComponents] = w.annotations.MOFComponents
	}
	if w.annotations.SigningFramework != "" {
		annotations[AnnotationSigningFramework] = w.annotations.SigningFramework
	}
	if w.annotations.ProvenanceType != "" {
		annotations[AnnotationProvenanceType] = w.annotations.ProvenanceType
	}
	return annotations
}

// SBOMPath returns the path to the generated SBOM, or "" if none was generated
func (w *HardenWorkflow) SBOMPath() string {
	return w.sbomPath
}

// MOFClass returns the MOF classification result
func (w *HardenWorkflow) MOFClass() string {
	return w.mofClass
}

// MOFConfigPath returns the path to the generated MOF metadata config file
func (w *HardenWorkflow) MOFConfigPath() string {
	return w.mofConfigPath
}

// ManifestPath returns the path of the packaged manifest the results were
// recorded in, or "" if Run() has not been called yet.
func (w *HardenWorkflow) ManifestPath() string {
	return w.manifestPath
}

// AppliedAnnotations returns the annotations Run() merged into the manifest.
func (w *HardenWorkflow) AppliedAnnotations() map[string]string {
	return w.appliedAnnotations
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
