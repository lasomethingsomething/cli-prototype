//go:build e2e

// Package e2e drives the built model-cli binary against a real OCI registry.
//
// Run locally with a throwaway zot registry and ORAS and syft on PATH
// (package generates an SBOM with syft and fails without it):
//
//	docker run -d --name zot -p 5000:5000 \
//	  -v "$PWD/test/e2e/zot-config.json:/etc/zot/config.json:ro" \
//	  ghcr.io/project-zot/zot-linux-amd64:latest
//	MODEL_CLI_E2E_REGISTRY=localhost:5000 go test -tags e2e -v ./test/e2e/
//
// Without MODEL_CLI_E2E_REGISTRY the tests are skipped.
package e2e

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lasomethingsomething/cli-prototype/internal/workflow"
)

var (
	registry string // e.g. localhost:5000
	binary   string // built model-cli
)

func TestMain(m *testing.M) {
	registry = os.Getenv("MODEL_CLI_E2E_REGISTRY")
	if registry == "" {
		fmt.Println("MODEL_CLI_E2E_REGISTRY not set; skipping e2e tests")
		os.Exit(0)
	}
	for _, tool := range []string{"oras", "syft"} {
		if _, err := exec.LookPath(tool); err != nil {
			fmt.Printf("%s not on PATH; e2e tests need it\n", tool)
			os.Exit(1)
		}
	}

	dir, err := os.MkdirTemp("", "model-cli-e2e-*")
	if err != nil {
		panic(err)
	}
	binary = filepath.Join(dir, "model-cli")
	build := exec.Command("go", "build", "-o", binary, "../..")
	build.Stdout, build.Stderr = os.Stdout, os.Stderr
	if err := build.Run(); err != nil {
		panic("building model-cli: " + err.Error())
	}

	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

// runCLI runs the binary non-interactively with an isolated HOME and cwd.
func runCLI(t *testing.T, cwd string, args ...string) (string, error) {
	t.Helper()
	cmd := exec.Command(binary, args...)
	cmd.Dir = cwd
	cmd.Env = append(os.Environ(), "HOME="+t.TempDir(), "MODEL_CLI_NO_INTERACTIVE=1")
	out, err := cmd.CombinedOutput()
	t.Logf("$ model-cli %s\n%s", strings.Join(args, " "), out)
	return string(out), err
}

func orasJSON(t *testing.T, dest interface{}, args ...string) {
	t.Helper()
	out, err := exec.Command("oras", args...).Output()
	if err != nil {
		t.Fatalf("oras %s: %v", strings.Join(args, " "), err)
	}
	if err := json.Unmarshal(out, dest); err != nil {
		t.Fatalf("oras %s: invalid JSON: %v\n%s", strings.Join(args, " "), err, out)
	}
}

func newModelDir(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "test-model")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	for name, content := range map[string]string{
		"model.safetensors": "not really weights",
		"README.md":         "# test model",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func uniqueTag() string { return fmt.Sprintf("v%d", time.Now().UnixNano()) }

type pushedManifest struct {
	ArtifactType string            `json:"artifactType"`
	Annotations  map[string]string `json:"annotations"`
	Layers       []struct {
		Annotations map[string]string `json:"annotations"`
	} `json:"layers"`
}

func TestPackageThenValidateAgainstRegistry(t *testing.T) {
	modelDir := newModelDir(t)
	artifact := "e2e/test-model:" + uniqueTag()
	ref := registry + "/" + artifact

	out, err := runCLI(t, filepath.Dir(modelDir), "package",
		"--model", "test-model", "--model-path", modelDir, "--artifact", artifact,
		"--registry", "oras", "--registry-url", registry,
		"--runtime", "vllm", "--accelerator", "cpu", "--memory-min", "1GiB")
	if err != nil {
		t.Fatalf("package failed: %v", err)
	}
	if !strings.Contains(out, "Local parity VERIFIED") {
		t.Errorf("package output lacks a passing parity check")
	}

	// The SBOM is a required prerequisite: package writes it next to the model.
	if _, err := os.Stat(filepath.Join(modelDir, "sbom.spdx-json")); err != nil {
		t.Errorf("package did not write an SBOM next to the model: %v", err)
	}

	// The registry holds a manifest with the CNCF annotations and the
	// artifact type the CLI declared.
	var pushed pushedManifest
	orasJSON(t, &pushed, "manifest", "fetch", ref)
	if pushed.ArtifactType != "application/vnd.cncf.ai.model" {
		t.Errorf("artifactType = %q", pushed.ArtifactType)
	}
	for key, want := range map[string]string{
		workflow.AnnotationArtifactType:   "model",
		workflow.AnnotationRuntime:        "vllm",
		workflow.AnnotationAccelerator:    "cpu",
		workflow.AnnotationMemoryMin:      "1GiB",
		workflow.AnnotationMOFClass:       "II", // weights + README, detected
		workflow.AnnotationProfileVersion: "1.0.0",
	} {
		if got := pushed.Annotations[key]; got != want {
			t.Errorf("pushed annotation %s = %q, want %q", key, got, want)
		}
	}
	if len(pushed.Layers) != 1 || pushed.Layers[0].Annotations["org.opencontainers.image.title"] != "test-model" {
		t.Errorf("pushed layers = %+v, want one layer titled test-model", pushed.Layers)
	}

	// The local preview manifest carries exactly the annotations that were pushed.
	local, err := workflow.ReadUnifiedOCIManifest(filepath.Join(modelDir, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	for key, want := range local.Annotations {
		if key == "org.opencontainers.image.title" {
			continue // ORAS sets its own title
		}
		if got := pushed.Annotations[key]; got != want {
			t.Errorf("annotation %s: local %q, registry %q", key, want, got)
		}
	}

	// validate gitops / admission read the same manifest back through the CLI.
	for _, tc := range []struct {
		name string
		args []string
	}{
		{"gitops", []string{"validate", "gitops", "--artifact", ref, "--registry", "oras", "--json-output"}},
		{"admission air-gapped", []string{"validate", "admission", "--artifact", ref, "--registry", "oras", "--env", "air-gapped", "--json-output"}},
		{"admission hybrid-cloud", []string{"validate", "admission", "--artifact", ref, "--registry", "oras", "--env", "hybrid-cloud", "--region", "eu-west-1", "--json-output"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out, err := runCLI(t, t.TempDir(), tc.args...)
			if err != nil {
				t.Fatalf("%s failed: %v", tc.name, err)
			}
			var report workflow.ValidationReport
			if err := json.Unmarshal([]byte(out[strings.Index(out, "{"):]), &report); err != nil {
				t.Fatalf("invalid JSON report: %v", err)
			}
			if !report.Passed || report.Status != "pass" || report.Artifact != ref {
				t.Errorf("report = %+v, want pass for %s", report, ref)
			}
			if len(report.Checks) == 0 {
				t.Error("report has no checks")
			}
		})
	}

	t.Run("missing artifact fails", func(t *testing.T) {
		_, err := runCLI(t, t.TempDir(), "validate", "gitops", "--artifact", registry+"/e2e/does-not-exist:v0", "--registry", "oras", "--quiet")
		if err == nil {
			t.Error("validate gitops on a missing artifact should exit non-zero")
		}
	})
}

func TestPushAttachesProvenanceReferrer(t *testing.T) {
	modelDir := newModelDir(t)
	artifact := "e2e/pushed-model:" + uniqueTag()
	ref := registry + "/" + artifact

	if _, err := runCLI(t, t.TempDir(), "push",
		"--artifact", artifact, "--destination", registry, "--registry", "oras",
		"--model-path", modelDir, "--generate-provenance"); err != nil {
		t.Fatalf("push failed: %v", err)
	}

	// The subject is still the model, not the attestation.
	var pushed pushedManifest
	orasJSON(t, &pushed, "manifest", "fetch", ref)
	if pushed.Annotations[workflow.AnnotationArtifactType] != "model" {
		t.Errorf("subject manifest was replaced; annotations = %v", pushed.Annotations)
	}

	// The attestation is discoverable as a referrer of the in-toto type...
	// (ORAS 1.2 lists referrers under "manifests", 1.3+ under "referrers".)
	raw, err := exec.Command("oras", "discover", "--format", "json", "--artifact-type", workflow.AttestationTypeProvenance, ref).Output()
	if err != nil {
		t.Fatalf("oras discover: %v", err)
	}
	var discovered struct {
		Manifests []struct{ ArtifactType string } `json:"manifests"`
		Referrers []struct{ ArtifactType string } `json:"referrers"`
	}
	if err := json.Unmarshal(raw, &discovered); err != nil {
		t.Fatalf("oras discover: invalid JSON: %v\n%s", err, raw)
	}
	if n := len(discovered.Manifests) + len(discovered.Referrers); n != 1 {
		t.Fatalf("want exactly one provenance referrer, got %d; raw discover output:\n%s", n, raw)
	}

	// ...and the CLI's own referrer path reads it back as a valid attestation.
	am := workflow.NewAttestationManager(nil, &workflow.ORASProvider{}, registry)
	attestation, err := am.GetAttestation(artifact)
	if err != nil {
		t.Fatalf("GetAttestation() error = %v", err)
	}
	if attestation.Statement.Predicate.Builder.ID != "model-cli" {
		t.Errorf("attestation builder = %q, want model-cli", attestation.Statement.Predicate.Builder.ID)
	}
	if err := workflow.ValidateAttestation(attestation); err != nil {
		t.Errorf("fetched attestation is invalid: %v", err)
	}
}
