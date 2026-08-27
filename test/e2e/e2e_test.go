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
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
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
	ArtifactType string                 `json:"artifactType"`
	Config       workflow.OCIDescriptor `json:"config"`
	Annotations  map[string]string      `json:"annotations"`
	Layers       []struct {
		Annotations map[string]string `json:"annotations"`
	} `json:"layers"`
}

func TestPackageThenValidateAgainstRegistry(t *testing.T) {
	modelDir := newModelDir(t)
	repo := "e2e/test-model"
	artifact := repo + ":" + uniqueTag()
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
		workflow.AnnotationProfileVersion: "1.0.0",
	} {
		if got := pushed.Annotations[key]; got != want {
			t.Errorf("pushed annotation %s = %q, want %q", key, got, want)
		}
	}
	// Packaging is only packaging: SBOM and MOF classification are the
	// separate harden step (issue #87), so nothing of them is produced here.
	if got, ok := pushed.Annotations[workflow.AnnotationMOFClass]; ok {
		t.Errorf("package pushed a MOF class %q; classification belongs to harden", got)
	}
	if sboms, _ := filepath.Glob(filepath.Join(modelDir, "sbom.*")); len(sboms) != 0 {
		t.Errorf("package wrote SBOM files %v; SBOM generation belongs to harden", sboms)
	}
	if len(pushed.Layers) != 1 || pushed.Layers[0].Annotations["org.opencontainers.image.title"] != "test-model" {
		t.Errorf("pushed layers = %+v, want one layer titled test-model", pushed.Layers)
	}

	local, err := workflow.ReadUnifiedOCIManifest(filepath.Join(modelDir, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}

	// The AI config was pushed as the manifest's config blob (#84): the
	// registry's config descriptor is the one in the local manifest.json...
	if pushed.Config.Digest != local.Config.Digest || pushed.Config.Size != local.Config.Size || pushed.Config.MediaType != local.Config.MediaType {
		t.Errorf("registry config descriptor = %+v, want the local manifest's %+v", pushed.Config, local.Config)
	}
	// ...and fetching the blob round-trips the local config.json and the
	// AI config it holds.
	localBlob, err := os.ReadFile(filepath.Join(modelDir, workflow.ConfigBlobFileName))
	if err != nil {
		t.Fatalf("package did not write %s: %v", workflow.ConfigBlobFileName, err)
	}
	blobRef := registry + "/" + repo + "@" + pushed.Config.Digest
	fetched, err := exec.Command("oras", "blob", "fetch", "--output", "-", blobRef).Output()
	if err != nil {
		t.Fatalf("oras blob fetch %s: %v", blobRef, err)
	}
	if !bytes.Equal(fetched, localBlob) {
		t.Errorf("config blob in registry differs from local config.json:\n%s\nvs\n%s", fetched, localBlob)
	}
	var fetchedConfig, localConfig workflow.AIModelConfig
	if err := json.Unmarshal(fetched, &fetchedConfig); err != nil {
		t.Fatalf("fetched config blob is not an AI model config: %v", err)
	}
	inline, err := json.Marshal(local.AIConfig)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(inline, &localConfig); err != nil {
		t.Fatalf("local aiConfig is not an AI model config: %v", err)
	}
	if fetchedConfig.ModelType == "" || !reflect.DeepEqual(fetchedConfig, localConfig) {
		t.Errorf("fetched AI config = %+v, want the local manifest's %+v", fetchedConfig, localConfig)
	}

	// The local manifest carries exactly the annotations that were pushed.
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

	// harden is the step after package: it generates the SBOM, classifies
	// the model and records both in the manifest package wrote.
	t.Run("harden", func(t *testing.T) {
		if _, err := exec.LookPath("syft"); err != nil {
			t.Skip("syft not on PATH; harden e2e needs it")
		}
		out, err := runCLI(t, filepath.Dir(modelDir), "harden",
			"--model", "test-model", "--model-path", modelDir, "--artifact", artifact,
			"--sbom-tool", "syft", "--sbom-format", "spdx-json")
		if err != nil {
			t.Fatalf("harden failed: %v", err)
		}
		if !strings.Contains(out, "Manifest updated") {
			t.Errorf("harden output does not report updating the manifest")
		}
		if _, err := os.Stat(filepath.Join(modelDir, "sbom.spdx-json")); err != nil {
			t.Errorf("expected SBOM next to the model: %v", err)
		}

		hardened, err := workflow.ReadUnifiedOCIManifest(filepath.Join(modelDir, "manifest.json"))
		if err != nil {
			t.Fatal(err)
		}
		for key, want := range map[string]string{
			workflow.AnnotationSBOMFormat:    "spdx-json",
			workflow.AnnotationMOFClass:      "II", // weights + README, detected
			workflow.AnnotationMOFComponents: "weights,documentation",
			workflow.AnnotationMOFVersion:    "1.0",
			workflow.AnnotationRuntime:       "vllm", // from package, kept
		} {
			if got := hardened.Annotations[key]; got != want {
				t.Errorf("hardened manifest annotation %s = %q, want %q", key, got, want)
			}
		}
	})
}

func TestHardenRequiresPackage(t *testing.T) {
	modelDir := newModelDir(t)

	out, err := runCLI(t, filepath.Dir(modelDir), "harden",
		"--model", "test-model", "--model-path", modelDir, "--artifact", "e2e/unpackaged:"+uniqueTag(),
		"--generate-sbom=false")
	if err == nil {
		t.Fatal("harden on an unpackaged model should exit non-zero")
	}
	if !strings.Contains(out, "model-cli package") {
		t.Errorf("harden output = %q, want it to tell the user to run `model-cli package` first", out)
	}
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

func TestSearchFindsPushedArtifacts(t *testing.T) {
	modelDir := newModelDir(t)
	tag := uniqueTag()
	modelArtifact := "e2e/search-test-model:" + tag
	modelRef := registry + "/" + modelArtifact

	out, err := runCLI(t, filepath.Dir(modelDir), "package",
		"--model", "test-model", "--model-path", modelDir, "--artifact", modelArtifact,
		"--registry", "oras", "--registry-url", registry,
		"--runtime", "vllm", "--accelerator", "cpu")
	if err != nil {
		t.Fatalf("package failed: %v", err)
	}
	if !strings.Contains(out, "Local parity VERIFIED") {
		t.Errorf("package output lacks a passing parity check")
	}
	var descriptor struct {
		Digest string `json:"digest"`
	}
	orasJSON(t, &descriptor, "manifest", "fetch", "--descriptor", modelRef)

	// A pipeline that uses the model, carrying the relationship graph that
	// `map` embeds, so --uses-model has something to find.
	pipelineRef := registry + "/e2e/search-test-pipeline:" + tag
	graph := workflow.NewRelationshipGraph()
	graph.AddModel(modelRef, "", "llm", "")
	graph.AddPipeline(pipelineRef, "", "inference", nil)
	if err := graph.AddModelToPipeline(pipelineRef, modelRef); err != nil {
		t.Fatal(err)
	}
	relationships, err := workflow.GenerateRelationshipGraphAnnotation(graph)
	if err != nil {
		t.Fatal(err)
	}
	pipelineDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(pipelineDir, "pipeline.yaml"), []byte("stages: []\n"), 0644); err != nil {
		t.Fatal(err)
	}
	push := exec.Command("oras", "push", pipelineRef,
		"--artifact-type", "application/vnd.cncf.ai.pipeline",
		"--annotation", workflow.AnnotationArtifactType+"=pipeline",
		"--annotation", "ai.pipeline.type=inference",
		"--annotation", workflow.AnnotationRelationshipGraph+"="+relationships,
		"pipeline.yaml")
	push.Dir = pipelineDir
	if out, err := push.CombinedOutput(); err != nil {
		t.Fatalf("oras push pipeline: %v\n%s", err, out)
	}

	// search --output json prints nothing but the SearchResults document.
	search := func(t *testing.T, filters ...string) workflow.SearchResults {
		t.Helper()
		args := append([]string{"search", "--destination", registry, "--registry-tool", "oras", "--output", "json"}, filters...)
		out, err := runCLI(t, t.TempDir(), args...)
		if err != nil {
			t.Fatalf("search failed: %v", err)
		}
		var results workflow.SearchResults
		if err := json.Unmarshal([]byte(out), &results); err != nil {
			t.Fatalf("--output json is not a JSON document: %v\n%s", err, out)
		}
		if results.Registry != registry || results.TotalCount != len(results.Results) {
			t.Errorf("results registry=%q total=%d for %d results", results.Registry, results.TotalCount, len(results.Results))
		}
		return results
	}
	references := func(results workflow.SearchResults) []string {
		var refs []string
		for _, r := range results.Results {
			refs = append(refs, r.Reference)
		}
		return refs
	}
	find := func(results workflow.SearchResults, ref string) *workflow.SearchResult {
		for i := range results.Results {
			if results.Results[i].Reference == ref {
				return &results.Results[i]
			}
		}
		return nil
	}

	t.Run("type model", func(t *testing.T) {
		results := search(t, "--type", "model")
		got := find(results, modelRef)
		if got == nil {
			t.Fatalf("results %v lack the pushed model %s", references(results), modelRef)
		}
		if got.Digest != descriptor.Digest {
			t.Errorf("digest = %q, want %q as stored in the registry", got.Digest, descriptor.Digest)
		}
		if got.ArtifactType != "model" || got.Annotations[workflow.AnnotationRuntime] != "vllm" {
			t.Errorf("result = %+v, want artifact_type model with the pushed annotations", got)
		}
		for _, r := range results.Results {
			if r.ArtifactType != "model" {
				t.Errorf("--type model listed %s of type %q", r.Reference, r.ArtifactType)
			}
		}
	})

	t.Run("metadata", func(t *testing.T) {
		results := search(t, "--type", "pipeline", "--metadata", "ai.pipeline.type=inference")
		if find(results, pipelineRef) == nil {
			t.Errorf("results %v lack the pipeline %s", references(results), pipelineRef)
		}
		if find(results, modelRef) != nil {
			t.Errorf("--type pipeline listed the model %s", modelRef)
		}
		if none := search(t, "--metadata", "ai.pipeline.type=does-not-exist"); len(none.Results) != 0 {
			t.Errorf("unmatched metadata filter returned %v", references(none))
		}
	})

	t.Run("uses-model", func(t *testing.T) {
		results := search(t, "--uses-model", modelRef)
		if refs := references(results); len(refs) != 1 || refs[0] != pipelineRef {
			t.Errorf("--uses-model %s = %v, want exactly [%s]", modelRef, refs, pipelineRef)
		}
	})

	t.Run("table output", func(t *testing.T) {
		out, err := runCLI(t, t.TempDir(), "search", "--destination", registry, "--registry-tool", "oras", "--type", "model")
		if err != nil {
			t.Fatalf("search failed: %v", err)
		}
		for _, want := range []string{
			"Search Results from " + registry + ":\n",
			". " + modelRef + "\n",
			"   Digest: " + descriptor.Digest + "\n",
			"   Type: model\n",
		} {
			if !strings.Contains(out, want) {
				t.Errorf("table output lacks %q", want)
			}
		}
	})
}
