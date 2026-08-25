package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestORASPushUploadsSourcePathWithAnnotations(t *testing.T) {
	calls := installFakeOras(t, "", 0)

	modelDir := filepath.Join(t.TempDir(), "my-model")
	if err := os.MkdirAll(modelDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(modelDir, "model.txt"), []byte("weights"), 0644); err != nil {
		t.Fatal(err)
	}

	p := &ORASProvider{}
	digest, err := p.Push("my-model:v1", "ghcr.io/my-org", modelDir, map[string]string{
		AnnotationArtifactType: "model",
		AnnotationRuntime:      "vllm",
	})
	if err != nil {
		t.Fatalf("Push() error = %v", err)
	}
	if digest != "" {
		t.Errorf("Push() digest = %q, want empty when oras prints nothing", digest)
	}

	got := calls()
	if len(got) != 1 {
		t.Fatalf("expected exactly one oras invocation, got %d: %v", len(got), got)
	}
	cwd, args, _ := strings.Cut(got[0], "\t")

	if cwd != filepath.Dir(modelDir) {
		t.Errorf("oras ran in %q, want the model's parent directory %q", cwd, filepath.Dir(modelDir))
	}
	want := "push ghcr.io/my-org/my-model:v1 --artifact-type application/vnd.cncf.ai.model --format json " +
		"--annotation org.cncf.ai.artifact.type=model --annotation org.cncf.ai.runtime=vllm my-model"
	if args != want {
		t.Errorf("oras args =\n  %s\nwant\n  %s", args, want)
	}
}

func TestORASPushRejectsMissingSource(t *testing.T) {
	calls := installFakeOras(t, "", 0)

	p := &ORASProvider{}
	_, err := p.Push("my-model:v1", "ghcr.io/my-org", filepath.Join(t.TempDir(), "does-not-exist"), nil)
	if err == nil {
		t.Fatal("Push() with a missing source path should fail")
	}
	if len(calls()) != 0 {
		t.Errorf("oras should not be invoked when the source path is missing, got %v", calls())
	}
}

func TestORASPushReportsToolFailure(t *testing.T) {
	installFakeOras(t, "", 2)

	p := &ORASProvider{}
	_, err := p.Push("my-model:v1", "ghcr.io/my-org", t.TempDir(), nil)
	if err == nil || !strings.Contains(err.Error(), "failed to push with ORAS") {
		t.Errorf("Push() error = %v, want an ORAS failure", err)
	}
}

const testDigest = "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func TestORASPushReturnsReportedDigest(t *testing.T) {
	t.Run("json output", func(t *testing.T) {
		installFakeOras(t, `{"reference":"ghcr.io/my-org/my-model:v1","digest":"`+testDigest+`","size":512}`, 0)
		digest, err := (&ORASProvider{}).Push("my-model:v1", "ghcr.io/my-org", t.TempDir(), nil)
		if err != nil {
			t.Fatalf("Push() error = %v", err)
		}
		if digest != testDigest {
			t.Errorf("Push() digest = %q, want %q", digest, testDigest)
		}
	})
	t.Run("plain output from older oras", func(t *testing.T) {
		installFakeOras(t, "Pushed ghcr.io/my-org/my-model:v1\nDigest: "+testDigest+"\n", 0)
		digest, err := (&ORASProvider{}).Push("my-model:v1", "ghcr.io/my-org", t.TempDir(), nil)
		if err != nil {
			t.Fatalf("Push() error = %v", err)
		}
		if digest != testDigest {
			t.Errorf("Push() digest = %q, want %q", digest, testDigest)
		}
	})
}

func TestORASGetArtifactDigestUsesDescriptor(t *testing.T) {
	calls := installFakeOras(t, `{"mediaType":"application/vnd.oci.image.manifest.v1+json","digest":"`+testDigest+`","size":512}`, 0)

	digest, err := (&ORASProvider{}).GetArtifactDigest("my-model:v1", "ghcr.io/my-org")
	if err != nil {
		t.Fatalf("GetArtifactDigest() error = %v", err)
	}
	if digest != testDigest {
		t.Errorf("GetArtifactDigest() = %q, want %q", digest, testDigest)
	}
	_, args, _ := strings.Cut(calls()[0], "\t")
	if args != "manifest fetch --descriptor ghcr.io/my-org/my-model:v1" {
		t.Errorf("oras args = %q, want descriptor fetch", args)
	}
}

func TestORASGetArtifactDigestRejectsMissingDigest(t *testing.T) {
	installFakeOras(t, `{"mediaType":"application/vnd.oci.image.manifest.v1+json"}`, 0)
	if _, err := (&ORASProvider{}).GetArtifactDigest("my-model:v1", "ghcr.io/my-org"); err == nil {
		t.Error("GetArtifactDigest() should fail when the descriptor has no digest")
	}
}

func TestRepositoryOf(t *testing.T) {
	cases := map[string]string{
		"my-model:v1":                   "my-model",
		"my-org/my-model:v1":            "my-org/my-model",
		"my-model@sha256:abc":           "my-model",
		"my-model":                      "my-model",
		"localhost:5000/my-model:v1":    "localhost:5000/my-model",
		"localhost:5000/my-model":       "localhost:5000/my-model",
		"my-org/my-model:v1@sha256:abc": "my-org/my-model:v1",
	}
	for in, want := range cases {
		if got := repositoryOf(in); got != want {
			t.Errorf("repositoryOf(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestORASGetReferrersDiscoversAndFetchesBlobs(t *testing.T) {
	const refDigest = "sha256:1111111111111111111111111111111111111111111111111111111111111111"
	const blobDigest = "sha256:2222222222222222222222222222222222222222222222222222222222222222"
	attestation := `{"_type":"https://in-toto.io/Statement/v1"}`
	calls := installFakeOrasSequence(t, []string{
		`{"manifests":[{"digest":"` + refDigest + `","artifactType":"` + AttestationTypeProvenance + `"}]}`,
		`{"schemaVersion":2,"layers":[{"mediaType":"application/json","digest":"` + blobDigest + `","size":42}]}`,
		attestation,
	}, 0)

	blobs, err := (&ORASProvider{}).GetReferrers("my-model:v1", "ghcr.io/my-org", AttestationTypeProvenance)
	if err != nil {
		t.Fatalf("GetReferrers() error = %v", err)
	}
	if len(blobs) != 1 || string(blobs[0]) != attestation {
		t.Fatalf("GetReferrers() = %q, want the attestation blob", blobs)
	}

	got := calls()
	wantArgs := []string{
		"discover --artifact-type " + AttestationTypeProvenance + " --format json ghcr.io/my-org/my-model:v1",
		"manifest fetch ghcr.io/my-org/my-model@" + refDigest,
		"blob fetch --output - ghcr.io/my-org/my-model@" + blobDigest,
	}
	if len(got) != len(wantArgs) {
		t.Fatalf("expected %d oras invocations, got %d: %v", len(wantArgs), len(got), got)
	}
	for i, want := range wantArgs {
		_, args, _ := strings.Cut(got[i], "\t")
		if args != want {
			t.Errorf("call %d args = %q, want %q", i, args, want)
		}
	}
}

func TestORASGetReferrersNoneFound(t *testing.T) {
	calls := installFakeOras(t, `{"manifests":[]}`, 0)
	blobs, err := (&ORASProvider{}).GetReferrers("my-model:v1", "ghcr.io/my-org", AttestationTypeProvenance)
	if err != nil {
		t.Fatalf("GetReferrers() error = %v", err)
	}
	if len(blobs) != 0 {
		t.Errorf("GetReferrers() = %v, want none", blobs)
	}
	if len(calls()) != 1 {
		t.Errorf("expected only the discover call, got %v", calls())
	}
}
