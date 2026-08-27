package workflow

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// TestHardenWorkflowDefaults verifies default values
func TestHardenWorkflowDefaults(t *testing.T) {
	hw := NewHardenWorkflow("oras")

	if !hw.generateSBOM {
		t.Error("generateSBOM default = false, want true")
	}
	if !hw.includeMOF {
		t.Error("includeMOF default = false, want true")
	}
	if hw.sbomTool != "syft" {
		t.Errorf("sbomTool default = %q, want %q", hw.sbomTool, "syft")
	}
	if hw.sbomFormat != SPDXJSON {
		t.Errorf("sbomFormat default = %q, want %q", hw.sbomFormat, SPDXJSON)
	}
	if hw.annotations == nil {
		t.Error("annotations default is nil, want non-nil")
	}
}

// fakeSBOMGenerator writes a placeholder SBOM instead of shelling out.
type fakeSBOMGenerator struct {
	calls int
	fail  bool
}

func (f *fakeSBOMGenerator) Name() string                { return "fake" }
func (f *fakeSBOMGenerator) IsInstalled() bool           { return true }
func (f *fakeSBOMGenerator) InstallInstructions() string { return "n/a" }
func (f *fakeSBOMGenerator) DefaultFormat() SBOMFormat   { return SPDXJSON }
func (f *fakeSBOMGenerator) Generate(modelPath, outputPath string, format SBOMFormat) error {
	f.calls++
	if f.fail {
		return os.ErrPermission
	}
	return os.WriteFile(outputPath, []byte(`{"spdxVersion":"SPDX-2.3"}`), 0644)
}

// packagedModelDir returns a model directory (weights + README, i.e. MOF
// class II) that `package` has already written a manifest.json for.
func packagedModelDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for name, content := range map[string]string{"model.safetensors": "weights", "README.md": "# model"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	pf := &PackageWorkflow{registry: "fake", registryProvider: &fakeRegistryProvider{installed: true}, annotations: NewAnnotationSet()}
	pf.annotations.Runtime = "kserve"
	pf.SetPackageInfo("m", dir, "m:v1", "", false, "")
	if err := pf.Run(); err != nil {
		t.Fatalf("package Run() error = %v", err)
	}
	return dir
}

func newTestHardenWorkflow(dir string) *HardenWorkflow {
	hw := NewHardenWorkflow("")
	hw.SetHardenInfo("m", dir, "m:v1")
	hw.SetOptions(false, true) // SBOM needs an external tool; covered separately
	return hw
}

// TestHardenWorkflowRequiresPackagedManifest verifies hardening refuses to
// run before packaging, and says what to do.
func TestHardenWorkflowRequiresPackagedManifest(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "model.safetensors"), []byte("weights"), 0644); err != nil {
		t.Fatal(err)
	}
	hw := newTestHardenWorkflow(dir)

	err := hw.Run()
	if err == nil || !strings.Contains(err.Error(), "model-cli package") {
		t.Fatalf("Run() error = %v, want an error telling the user to run `model-cli package` first", err)
	}
	if _, statErr := os.Stat(filepath.Join(dir, "mof.json")); !os.IsNotExist(statErr) {
		t.Error("harden wrote mof.json although it refused to run")
	}
}

// TestHardenWorkflowSBOMToolUnavailableBlocks verifies that an unavailable
// SBOM tool blocks hardening before MOF classification runs, naming the
// prerequisite (Issue #89).
func TestHardenWorkflowSBOMToolUnavailableBlocks(t *testing.T) {
	dir := packagedModelDir(t)
	hw := newTestHardenWorkflow(dir)
	hw.SetOptions(true, true) // generateSBOM=true, includeMOF=true
	hw.SetSBOMTool("nonexistent-tool", SPDXJSON)

	err := hw.Run()
	if err == nil {
		t.Fatal("expected SBOM generation to fail and block workflow, got nil")
	}
	if !strings.Contains(err.Error(), "SBOM") || !strings.Contains(err.Error(), "prerequisite") {
		t.Errorf("expected error to mention SBOM prerequisite, got: %v", err)
	}
	if hw.MOFClass() != "" || hw.MOFConfigPath() != "" {
		t.Errorf("MOF step ran despite SBOM failure: class %q, config %q", hw.MOFClass(), hw.MOFConfigPath())
	}
}

// TestHardenWorkflowSBOMGenerateFailureBlocks verifies that a failure inside
// the SBOM generator itself (tool installed, scan fails) blocks hardening
// before MOF classification runs (Issue #89).
func TestHardenWorkflowSBOMGenerateFailureBlocks(t *testing.T) {
	fakeSyft(t, "echo 'boom' >&2; exit 1")

	dir := packagedModelDir(t)
	hw := newTestHardenWorkflow(dir)
	hw.SetOptions(true, true)
	hw.SetSBOMTool("syft", SPDXJSON)

	err := hw.Run()
	if err == nil {
		t.Fatal("expected SBOM generation failure to block workflow, got nil")
	}
	if !strings.Contains(err.Error(), "syft generation failed") || !strings.Contains(err.Error(), "prerequisite") {
		t.Errorf("error = %v, want SBOM generation failure naming the prerequisite", err)
	}
	if hw.MOFClass() != "" || hw.MOFConfigPath() != "" {
		t.Errorf("MOF step ran despite SBOM failure: class %q, config %q", hw.MOFClass(), hw.MOFConfigPath())
	}
}

// TestHardenWorkflowAppliesMOFClassification verifies the detected MOF class
// and components land in the packaged manifest when the user left them
// empty, and that an explicit class is kept.
func TestHardenWorkflowAppliesMOFClassification(t *testing.T) {
	t.Run("detected when empty", func(t *testing.T) {
		dir := packagedModelDir(t)
		hw := newTestHardenWorkflow(dir)
		if err := hw.Run(); err != nil {
			t.Fatalf("Run() error = %v", err)
		}

		if hw.MOFClass() != "II" {
			t.Errorf("MOFClass() = %q, want detected II", hw.MOFClass())
		}
		m, err := ReadUnifiedOCIManifest(filepath.Join(dir, "manifest.json"))
		if err != nil {
			t.Fatal(err)
		}
		for key, want := range map[string]string{
			AnnotationMOFClass:      "II",
			AnnotationMOFVersion:    "1.0",
			AnnotationMOFComponents: "weights,documentation",
			AnnotationRuntime:       "kserve", // written by package, must survive
		} {
			if got := m.Annotations[key]; got != want {
				t.Errorf("manifest annotation %s = %q, want %q", key, got, want)
			}
		}
		if _, ok := hw.AppliedAnnotations()[AnnotationSBOMFormat]; ok {
			t.Error("harden claimed an SBOM format without generating an SBOM")
		}
		if err := ValidateOCIManifest(m); err != nil {
			t.Errorf("updated manifest does not validate: %v", err)
		}
		if hw.MOFConfigPath() != filepath.Join(dir, "mof.json") {
			t.Errorf("MOFConfigPath() = %q, want mof.json next to the model", hw.MOFConfigPath())
		}
		if _, err := os.Stat(hw.MOFConfigPath()); err != nil {
			t.Errorf("expected MOF config file: %v", err)
		}
	})

	t.Run("explicit class kept", func(t *testing.T) {
		dir := packagedModelDir(t)
		hw := newTestHardenWorkflow(dir)
		annotations := NewAnnotationSet()
		annotations.MOFClass = "I"
		annotations.MOFComponents = "weights,code"
		hw.SetAnnotations(annotations)
		if err := hw.Run(); err != nil {
			t.Fatalf("Run() error = %v", err)
		}

		m, err := ReadUnifiedOCIManifest(filepath.Join(dir, "manifest.json"))
		if err != nil {
			t.Fatal(err)
		}
		if m.Annotations[AnnotationMOFClass] != "I" || m.Annotations[AnnotationMOFComponents] != "weights,code" {
			t.Errorf("explicit MOF annotations were overwritten: class=%q components=%q", m.Annotations[AnnotationMOFClass], m.Annotations[AnnotationMOFComponents])
		}
		if hw.MOFClass() != "I" {
			t.Errorf("MOFClass() = %q, want declared I", hw.MOFClass())
		}
	})

	t.Run("rerun is stable", func(t *testing.T) {
		dir := packagedModelDir(t)
		for i := 0; i < 2; i++ {
			hw := newTestHardenWorkflow(dir)
			if err := hw.Run(); err != nil {
				t.Fatalf("Run() #%d error = %v", i+1, err)
			}
			// mof.json from the previous run must not count as training data
			if hw.MOFClass() != "II" {
				t.Errorf("Run() #%d MOFClass() = %q, want II", i+1, hw.MOFClass())
			}
		}
	})
}

// TestHardenWorkflowRecordsSBOM verifies a generated SBOM is recorded in the
// packaged manifest, and a failed generation blocks the workflow (Issue #89)
// without being claimed in the manifest.
func TestHardenWorkflowRecordsSBOM(t *testing.T) {
	t.Run("generated", func(t *testing.T) {
		dir := packagedModelDir(t)
		gen := &fakeSBOMGenerator{}
		hw := newTestHardenWorkflow(dir)
		hw.SetOptions(true, false)
		hw.SetSBOMTool("fake", CycloneDXJSON)
		hw.sbomGenerator = gen
		if err := hw.Run(); err != nil {
			t.Fatalf("Run() error = %v", err)
		}

		wantPath := filepath.Join(dir, "sbom.cyclonedx-json")
		if gen.calls != 1 || hw.SBOMPath() != wantPath {
			t.Errorf("generator calls = %d, SBOMPath() = %q, want one call writing %s", gen.calls, hw.SBOMPath(), wantPath)
		}
		if _, err := os.Stat(wantPath); err != nil {
			t.Errorf("expected SBOM file: %v", err)
		}
		m, err := ReadUnifiedOCIManifest(filepath.Join(dir, "manifest.json"))
		if err != nil {
			t.Fatal(err)
		}
		if got := m.Annotations[AnnotationSBOMFormat]; got != "cyclonedx-json" {
			t.Errorf("manifest annotation %s = %q, want cyclonedx-json", AnnotationSBOMFormat, got)
		}
		if _, ok := m.Annotations[AnnotationMOFClass]; ok {
			t.Error("MOF class recorded although MOF classification was disabled")
		}
	})

	t.Run("generation failure", func(t *testing.T) {
		dir := packagedModelDir(t)
		hw := newTestHardenWorkflow(dir)
		hw.SetOptions(true, false)
		hw.sbomGenerator = &fakeSBOMGenerator{fail: true}

		err := hw.Run()
		if err == nil || !strings.Contains(err.Error(), "SBOM generation failed") || !strings.Contains(err.Error(), "prerequisite") {
			t.Fatalf("Run() error = %v, want SBOM generation failure naming the prerequisite", err)
		}
		if hw.SBOMPath() != "" {
			t.Errorf("SBOMPath() = %q, want empty after a failed generation", hw.SBOMPath())
		}
		if _, ok := hw.AppliedAnnotations()[AnnotationSBOMFormat]; ok {
			t.Error("SBOM format claimed in the manifest although generation failed")
		}
	})
}

func TestCanonicalMOFComponents(t *testing.T) {
	got := canonicalMOFComponents([]string{"license", "documentation", "weights", "training-data", "code"})
	want := []string{"weights", "code", "training-data", "documentation", "license"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("canonicalMOFComponents() = %v, want %v", got, want)
	}
	if got := canonicalMOFComponents([]string{"documentation", "weights"}); !reflect.DeepEqual(got, []string{"weights", "documentation"}) {
		t.Errorf("canonicalMOFComponents(subset) = %v, want [weights documentation]", got)
	}
}
