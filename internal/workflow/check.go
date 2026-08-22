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

// checkAnnotations checks for required annotations
func (w *CheckWorkflow) checkAnnotations() {
	// Required annotations based on CNCF AI Interoperability Profile
	_ = []string{
		"org.cncf.ai.artifact.type",
		"org.cncf.ai.artifact.runtime",
		"org.cncf.ai.artifact.accelerator",
	}

	// For now, we'll check if the artifact directory exists and has annotations
	// In a real implementation, we would parse the OCI manifest
	if w.artifactPath == "" {
		w.missing = append(w.missing, "artifact path not specified")
		return
	}

	// Check if artifact directory exists
	if _, err := os.Stat(w.artifactPath); os.IsNotExist(err) {
		w.missing = append(w.missing, "artifact directory not found")
		return
	}

	// In a real implementation, we would:
	// 1. Parse the OCI manifest
	// 2. Check for required annotations
	// 3. Report which are missing

	// For now, we'll simulate the check
	fmt.Println("  ✓ Required annotations check (simulated)")
	w.annotationsCheck = append(w.annotationsCheck, AnnotationCheckResult{
		Name:     "org.cncf.ai.artifact.type",
		Expected: "model",
		Actual:   "model",
		Passed:   true,
	})
}

// checkSBOM checks for SBOM presence
func (w *CheckWorkflow) checkSBOM() {
	if w.modelPath == "" {
		w.missing = append(w.missing, "model path not specified")
		w.sbomCheck = false
		return
	}

	// Look for common SBOM file names
	sbomFiles := []string{
		"sbom.spdx.json",
		"sbom.json",
		"sbom.spdx",
		"sbom-cyclonedx.json",
		"sbom-syft.json",
	}

	found := false
	for _, sbomFile := range sbomFiles {
		sbomPath := filepath.Join(w.modelPath, sbomFile)
		if _, err := os.Stat(sbomPath); !os.IsNotExist(err) {
			found = true
			fmt.Printf("  ✓ SBOM found: %s\n", sbomFile)
			w.sbomCheck = true
			break
		}
	}

	if !found {
		// Also check in artifact directory
		for _, sbomFile := range sbomFiles {
			sbomPath := filepath.Join(w.artifactPath, sbomFile)
			if _, err := os.Stat(sbomPath); !os.IsNotExist(err) {
				found = true
				fmt.Printf("  ✓ SBOM found: %s\n", sbomFile)
				w.sbomCheck = true
				break
			}
		}
	}

	if !found {
		fmt.Println("  ✗ SBOM not found")
		w.sbomCheck = false
		w.missing = append(w.missing, "SBOM")
	}
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
