package workflow

import (
	"os"
	"path/filepath"
	"testing"
)

// TestCheckWorkflowFindsHardenSBOM verifies `validate local` finds the SBOM
// `harden` writes for every --sbom-format (issue #99): the file name is
// derived from the format recorded in manifest.json, not guessed.
func TestCheckWorkflowFindsHardenSBOM(t *testing.T) {
	for _, format := range AllSBOMFormats {
		t.Run(string(format), func(t *testing.T) {
			dir := hardenPackagedModelDir(t)
			hw := newTestHardenWorkflow(dir)
			hw.SetOptions(true, true)
			hw.sbomGenerator = &fakeSBOMGenerator{}
			hw.SetSBOMTool("fake", format)
			if err := hw.Run(); err != nil {
				t.Fatalf("harden Run() error = %v", err)
			}

			cw := NewCheckWorkflow()
			cw.SetCheckInfo(dir, dir)
			cw.checkSBOM()
			if !cw.sbomCheck {
				t.Errorf("SBOM %s written by harden was not found; missing = %v", filepath.Base(hw.SBOMPath()), cw.missing)
			}
		})
	}
}

// TestCheckWorkflowSBOMWithoutManifest verifies the SBOM is still found when
// there is no manifest.json to read the format from, and that names other
// tools use are recognised too.
func TestCheckWorkflowSBOMWithoutManifest(t *testing.T) {
	for _, name := range []string{"sbom.spdx-json", "sbom.cyclonedx-json", "sbom.spdx.json", "sbom-cyclonedx.json"} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, name), []byte("{}"), 0644); err != nil {
				t.Fatal(err)
			}
			cw := NewCheckWorkflow()
			cw.SetCheckInfo(dir, dir)
			cw.checkSBOM()
			if !cw.sbomCheck {
				t.Errorf("%s was not found", name)
			}
		})
	}
}

// TestCheckWorkflowSBOMInArtifactPath verifies the artifact path is searched
// when the model path has no SBOM.
func TestCheckWorkflowSBOMInArtifactPath(t *testing.T) {
	modelDir, artifactDir := t.TempDir(), t.TempDir()
	if err := os.WriteFile(filepath.Join(artifactDir, "sbom.spdx-json"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}
	cw := NewCheckWorkflow()
	cw.SetCheckInfo(modelDir, artifactDir)
	cw.checkSBOM()
	if !cw.sbomCheck {
		t.Error("SBOM in artifact path was not found")
	}
}

// TestCheckWorkflowSBOMMissing verifies a missing SBOM is reported.
func TestCheckWorkflowSBOMMissing(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("no sbom here"), 0644); err != nil {
		t.Fatal(err)
	}
	cw := NewCheckWorkflow()
	cw.SetCheckInfo(dir, dir)
	cw.checkSBOM()
	if cw.sbomCheck {
		t.Error("sbomCheck = true, want false")
	}
	if len(cw.missing) != 1 || cw.missing[0] != "SBOM" {
		t.Errorf("missing = %v, want [SBOM]", cw.missing)
	}
}
