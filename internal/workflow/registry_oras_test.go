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
	err := p.Push("my-model:v1", "ghcr.io/my-org", modelDir, map[string]string{
		AnnotationArtifactType: "model",
		AnnotationRuntime:      "vllm",
	})
	if err != nil {
		t.Fatalf("Push() error = %v", err)
	}

	got := calls()
	if len(got) != 1 {
		t.Fatalf("expected exactly one oras invocation, got %d: %v", len(got), got)
	}
	cwd, args, _ := strings.Cut(got[0], "\t")

	if cwd != filepath.Dir(modelDir) {
		t.Errorf("oras ran in %q, want the model's parent directory %q", cwd, filepath.Dir(modelDir))
	}
	want := "push ghcr.io/my-org/my-model:v1 --artifact-type application/vnd.cncf.ai.model " +
		"--annotation org.cncf.ai.artifact.type=model --annotation org.cncf.ai.runtime=vllm my-model"
	if args != want {
		t.Errorf("oras args =\n  %s\nwant\n  %s", args, want)
	}
}

func TestORASPushRejectsMissingSource(t *testing.T) {
	calls := installFakeOras(t, "", 0)

	p := &ORASProvider{}
	err := p.Push("my-model:v1", "ghcr.io/my-org", filepath.Join(t.TempDir(), "does-not-exist"), nil)
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
	err := p.Push("my-model:v1", "ghcr.io/my-org", t.TempDir(), nil)
	if err == nil || !strings.Contains(err.Error(), "failed to push with ORAS") {
		t.Errorf("Push() error = %v, want an ORAS failure", err)
	}
}
