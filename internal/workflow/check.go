package workflow

import (
	"fmt"
	"os"
	"path/filepath"
)

// CheckWorkflow validates a local artifact for compliance before push
type CheckWorkflow struct {
	// Inputs
	modelPath    string
	artifactPath string

	// Results
	passed           bool
	missing          []string
	annotationsCheck []AnnotationCheckResult
	sbomCheck        bool
	mofCheck         bool

	// State
	workflowErr error
}

// AnnotationCheckResult represents the result of checking an annotation
type AnnotationCheckResult struct {
	Name     string
	Expected string
	Actual   string
	Passed   bool
}

// NewCheckWorkflow creates a new compliance check workflow
func NewCheckWorkflow() *CheckWorkflow {
	return &CheckWorkflow{
		passed:  true,
		missing: make([]string, 0),
	}
}

// SetCheckInfo sets the paths to check
func (w *CheckWorkflow) SetCheckInfo(modelPath, artifactPath string) {
	w.modelPath = modelPath
	w.artifactPath = artifactPath
}

// Run executes the compliance check workflow
func (w *CheckWorkflow) Run() error {
	fmt.Println("Running local compliance check...")
	fmt.Println()

	// Check 1: Required annotations present
	fmt.Println("→ Checking required annotations...")
	w.checkAnnotations()

	// Check 2: SBOM present
	fmt.Println("→ Checking SBOM presence...")
	w.checkSBOM()

	// Check 3: MOF classification present
	fmt.Println("→ Checking MOF classification...")
	w.checkMOF()

	if w.workflowErr != nil {
		return w.workflowErr
	}

	// Determine overall pass/fail
	w.passed = len(w.missing) == 0 && w.sbomCheck && w.mofCheck

	if w.passed {
		fmt.Println("\n✓ All compliance checks passed")
	} else {
		fmt.Println("\n✗ Compliance checks failed")
		if len(w.missing) > 0 {
			fmt.Println("\nMissing required items:")
			for _, item := range w.missing {
				fmt.Printf("  - %s\n", item)
			}
		}
		if !w.sbomCheck {
			fmt.Println("\n✗ SBOM not found or invalid")
		}
		if !w.mofCheck {
			fmt.Println("\n✗ MOF classification not found or invalid")
		}
		missingCount := len(w.missing)
		if !w.sbomCheck {
			missingCount++
		}
		if !w.mofCheck {
			missingCount++
		}
		w.workflowErr = fmt.Errorf("compliance check failed: %d issues found", missingCount)
	}

	return w.workflowErr
}

// checkAnnotations checks for required annotations in the OCI manifest
func (w *CheckWorkflow) checkAnnotations() {
	// Determine which path to check - prefer artifactPath if provided, otherwise modelPath
	checkPath := w.artifactPath
	if checkPath == "" {
		checkPath = w.modelPath
	}
	if checkPath == "" {
		w.missing = append(w.missing, "model/artifact path not specified")
		return
	}

	// Check if the path exists
	if _, err := os.Stat(checkPath); os.IsNotExist(err) {
		w.missing = append(w.missing, fmt.Sprintf("path not found: %s", checkPath))
		return
	}

	// Try to read manifest.json from the path
	manifestPath := filepath.Join(checkPath, "manifest.json")
	manifest, err := ReadUnifiedOCIManifest(manifestPath)
	if err != nil {
		// If manifest.json doesn't exist, this might be an unpackaged directory
		// For now, we'll check if it's a valid model/artifact directory
		w.missing = append(w.missing, "OCI manifest.json not found")
		fmt.Println("  ✗ Required annotations check: manifest.json not found")
		return
	}

	// Required annotations based on what the package step actually writes
	// Source of truth: the annotations in annotations.go that package uses
	requiredAnnotations := []string{
		AnnotationProfileVersion,
		AnnotationArtifactType,
		AnnotationRuntime,
		AnnotationAccelerator,
		AnnotationPackagingFormat,
		AnnotationSBOMFormat,
		AnnotationSigningFramework,
		AnnotationProvenanceType,
		AnnotationMOFClass,
		AnnotationMOFVersion,
		AnnotationMOFComponents,
	}

	// Check each required annotation
	allPassed := true
	for _, requiredKey := range requiredAnnotations {
		actualValue, ok := manifest.Annotations[requiredKey]
		if !ok || actualValue == "" {
			w.missing = append(w.missing, requiredKey)
			w.annotationsCheck = append(w.annotationsCheck, AnnotationCheckResult{
				Name:     requiredKey,
				Expected: "(any)",
				Actual:   "",
				Passed:   false,
			})
			allPassed = false
			fmt.Printf("  ✗ Missing required annotation: %s\n", requiredKey)
		} else {
			w.annotationsCheck = append(w.annotationsCheck, AnnotationCheckResult{
				Name:     requiredKey,
				Expected: "(any)",
				Actual:   actualValue,
				Passed:   true,
			})
			fmt.Printf("  ✓ %s: %s\n", requiredKey, actualValue)
		}
	}

	if allPassed && len(w.missing) == 0 {
		fmt.Println("  ✓ Required annotations check")
	}
}

// checkSBOM checks for SBOM presence
func (w *CheckWorkflow) checkSBOM() {
	if w.modelPath == "" {
		w.missing = append(w.missing, "model path not specified")
		w.sbomCheck = false
		return
	}

	for _, dir := range w.searchDirs() {
		for _, sbomFile := range w.sbomFileCandidates(dir) {
			if _, err := os.Stat(filepath.Join(dir, sbomFile)); err == nil {
				fmt.Printf("  ✓ SBOM found: %s\n", sbomFile)
				w.sbomCheck = true
				return
			}
		}
	}

	fmt.Println("  ✗ SBOM not found")
	w.sbomCheck = false
	w.missing = append(w.missing, "SBOM")
}

// searchDirs returns the directories to look for artifacts in: the model
// path first, then the artifact path if it is a different directory.
func (w *CheckWorkflow) searchDirs() []string {
	dirs := []string{w.modelPath}
	if w.artifactPath != "" && w.artifactPath != w.modelPath {
		dirs = append(dirs, w.artifactPath)
	}
	return dirs
}

// sbomFileCandidates returns the SBOM file names to look for in dir, most
// specific first: the file `harden` wrote according to the SBOM format
// recorded in dir's manifest.json, then `harden`'s naming for every known
// format, then names other tools commonly use.
func (w *CheckWorkflow) sbomFileCandidates(dir string) []string {
	var candidates []string
	if format := manifestSBOMFormat(filepath.Join(dir, "manifest.json")); format != "" {
		candidates = append(candidates, SBOMFileName(format))
	}
	for _, format := range AllSBOMFormats {
		candidates = append(candidates, SBOMFileName(format))
	}
	candidates = append(candidates,
		"sbom.spdx.json",
		"sbom.json",
		"sbom.spdx",
		"sbom-cyclonedx.json",
		"sbom-syft.json",
	)
	return candidates
}

// manifestSBOMFormat returns the SBOM format `harden` recorded in the
// manifest at path, or "" if there is no readable manifest or no SBOM
// annotation in it.
func manifestSBOMFormat(path string) SBOMFormat {
	manifest, err := ReadUnifiedOCIManifest(path)
	if err != nil {
		return ""
	}
	return SBOMFormat(manifest.Annotations[AnnotationSBOMFormat])
}

// checkMOF checks for MOF classification
func (w *CheckWorkflow) checkMOF() {
	if w.modelPath == "" {
		w.missing = append(w.missing, "model path not specified")
		w.mofCheck = false
		return
	}

	// Look for MOF metadata files
	mofFiles := []string{
		"mof-classification.json",
		"mof.json",
		"classification.json",
	}

	found := false
	for _, mofFile := range mofFiles {
		mofPath := filepath.Join(w.modelPath, mofFile)
		if _, err := os.Stat(mofPath); !os.IsNotExist(err) {
			found = true
			fmt.Printf("  ✓ MOF classification found: %s\n", mofFile)
			w.mofCheck = true
			break
		}
	}

	if !found {
		// Also check in artifact directory
		for _, mofFile := range mofFiles {
			mofPath := filepath.Join(w.artifactPath, mofFile)
			if _, err := os.Stat(mofPath); !os.IsNotExist(err) {
				found = true
				fmt.Printf("  ✓ MOF classification found: %s\n", mofFile)
				w.mofCheck = true
				break
			}
		}
	}

	if !found {
		fmt.Println("  ✗ MOF classification not found")
		w.mofCheck = false
		w.missing = append(w.missing, "MOF classification")
	}
}

// Passed returns whether all checks passed
func (w *CheckWorkflow) Passed() bool {
	return w.passed
}

// Missing returns the list of missing required items
func (w *CheckWorkflow) Missing() []string {
	return w.missing
}

// SBOMCheck returns whether SBOM check passed
func (w *CheckWorkflow) SBOMCheck() bool {
	return w.sbomCheck
}

// MOFCheck returns whether MOF check passed
func (w *CheckWorkflow) MOFCheck() bool {
	return w.mofCheck
}

// AnnotationsCheck returns the annotation check results
func (w *CheckWorkflow) AnnotationsCheck() []AnnotationCheckResult {
	return w.annotationsCheck
}
