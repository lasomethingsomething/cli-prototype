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
	// No config.json/manifest.json in the source: no --config is passed.
	want := "push ghcr.io/my-org/my-model:v1 --artifact-type application/vnd.cncf.ai.model --format json " +
		"--annotation org.cncf.ai.artifact.type=model --annotation org.cncf.ai.runtime=vllm my-model"
	if args != want {
		t.Errorf("oras args =\n  %s\nwant\n  %s", args, want)
	}
}

// packagedModelDir returns a model directory with manifest.json and
// config.json as written by `package`, with the given artifact type.
func packagedModelDir(t *testing.T, artifactType ArtifactType) string {
	t.Helper()
	modelDir := filepath.Join(t.TempDir(), "my-model")
	if err := os.MkdirAll(modelDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(modelDir, "model.txt"), []byte("weights"), 0644); err != nil {
		t.Fatal(err)
	}
	m, err := NewManifestFromDirectory(artifactType, "my-model:v1", modelDir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := WriteUnifiedOCIManifest(m, filepath.Join(modelDir, "manifest.json")); err != nil {
		t.Fatal(err)
	}
	return modelDir
}

func TestORASPushAttachesConfigBlob(t *testing.T) {
	for _, tc := range []struct {
		artifactType  ArtifactType
		wantMediaType string
	}{
		{ArtifactTypeModel, AIModelConfigMediaType},
		{ArtifactTypeSkill, AISkillConfigMediaType},
		{ArtifactTypePipeline, AIPipelineConfigMediaType},
	} {
		t.Run(string(tc.artifactType), func(t *testing.T) {
			calls := installFakeOras(t, "", 0)
			modelDir := packagedModelDir(t, tc.artifactType)

			_, err := (&ORASProvider{}).Push("my-model:v1", "ghcr.io/my-org", modelDir, map[string]string{
				AnnotationArtifactType: string(tc.artifactType),
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
			// The config path is relative to the parent directory ORAS runs
			// in, like the source itself, and typed with the manifest's
			// config media type.
			want := "push ghcr.io/my-org/my-model:v1 --artifact-type application/vnd.cncf.ai." + string(tc.artifactType) +
				" --format json --annotation org.cncf.ai.artifact.type=" + string(tc.artifactType) +
				" --config my-model/config.json:" + tc.wantMediaType + " my-model"
			if args != want {
				t.Errorf("oras args =\n  %s\nwant\n  %s", args, want)
			}
		})
	}
}

func TestORASPushSkipsConfigBlobWithoutManifest(t *testing.T) {
	calls := installFakeOras(t, "", 0)

	// A model directory that ships its own config.json (as Hugging Face
	// models do) but was not packaged: nothing says what the blob is.
	modelDir := filepath.Join(t.TempDir(), "my-model")
	if err := os.MkdirAll(modelDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(modelDir, "config.json"), []byte(`{"hidden_size":128}`), 0644); err != nil {
		t.Fatal(err)
	}

	if _, err := (&ORASProvider{}).Push("my-model:v1", "ghcr.io/my-org", modelDir, nil); err != nil {
		t.Fatalf("Push() error = %v", err)
	}
	_, args, _ := strings.Cut(calls()[0], "\t")
	if strings.Contains(args, "--config") {
		t.Errorf("oras args = %q, want no --config without a manifest.json", args)
	}
}

func TestORASPushRejectsUnreadableManifestNextToConfigBlob(t *testing.T) {
	calls := installFakeOras(t, "", 0)

	modelDir := filepath.Join(t.TempDir(), "my-model")
	if err := os.MkdirAll(modelDir, 0755); err != nil {
		t.Fatal(err)
	}
	for name, content := range map[string]string{"config.json": "{}", "manifest.json": "not json"} {
		if err := os.WriteFile(filepath.Join(modelDir, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	if _, err := (&ORASProvider{}).Push("my-model:v1", "ghcr.io/my-org", modelDir, nil); err == nil {
		t.Fatal("Push() should fail when manifest.json next to config.json cannot be read")
	}
	if len(calls()) != 0 {
		t.Errorf("oras should not be invoked, got %v", calls())
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

func TestORASPushReferrerUsesAttach(t *testing.T) {
	calls := installFakeOras(t, "", 0)

	err := (&ORASProvider{}).PushReferrer("my-model:v1", "ghcr.io/my-org", AttestationTypeProvenance,
		[]byte(`{"_type":"https://in-toto.io/Statement/v1"}`),
		map[string]string{"org.opencontainers.image.title": "provenance-attestation"})
	if err != nil {
		t.Fatalf("PushReferrer() error = %v", err)
	}
	got := calls()
	if len(got) != 1 {
		t.Fatalf("expected one oras invocation, got %v", got)
	}
	_, args, _ := strings.Cut(got[0], "\t")
	want := "attach ghcr.io/my-org/my-model:v1 --artifact-type " + AttestationTypeProvenance +
		" --annotation org.opencontainers.image.title=provenance-attestation referrer.json:" + AttestationTypeProvenance
	if args != want {
		t.Errorf("oras args =\n  %s\nwant\n  %s", args, want)
	}
}

func TestParseDiscoverOutputAcceptsBothKeys(t *testing.T) {
	for name, in := range map[string]string{
		"oras 1.2 manifests": `{"manifests":[{"digest":"sha256:aa","artifactType":"t"}]}`,
		"oras 1.3 referrers": `{"reference":"r","referrers":[{"digest":"sha256:aa","artifactType":"t"}]}`,
	} {
		refs, err := parseDiscoverOutput([]byte(in))
		if err != nil || len(refs) != 1 || refs[0].Digest != "sha256:aa" {
			t.Errorf("%s: refs=%v err=%v", name, refs, err)
		}
	}
	if refs, err := parseDiscoverOutput([]byte(`{"referrers":[]}`)); err != nil || len(refs) != 0 {
		t.Errorf("empty: refs=%v err=%v", refs, err)
	}
}

const (
	searchDigestModel    = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	searchDigestPipeline = "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	searchDigestPlain    = "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
)

// installSearchRegistry fakes a registry with three repositories as the
// ORAS CLI reports them: e2e/model has a model (v1) and a pipeline (v2),
// e2e/broken cannot list its tags, and plain holds a manifest without
// annotations. It returns the recorded oras invocations.
func installSearchRegistry(t *testing.T) func() []string {
	t.Helper()
	ok := func(stdout string) fakeOrasResponse { return fakeOrasResponse{stdout: stdout} }
	return installFakeOrasResponses(t, []fakeOrasResponse{
		ok("e2e/model\ne2e/broken\nplain\n"), // repo ls
		ok("v1\nv2\n"),                       // repo tags e2e/model
		ok(`{"mediaType":"application/vnd.oci.image.manifest.v1+json","digest":"` + searchDigestModel + `","size":1}`),
		ok(`{"schemaVersion":2,"artifactType":"application/vnd.cncf.ai.model","annotations":{"` + AnnotationArtifactType + `":"model","ai.model.type":"llm","` + AnnotationRuntime + `":"vllm"}}`),
		ok(`{"mediaType":"application/vnd.oci.image.manifest.v1+json","digest":"` + searchDigestPipeline + `","size":1}`),
		ok(`{"schemaVersion":2,"artifactType":"application/vnd.cncf.ai.pipeline","annotations":{"` + AnnotationArtifactType + `":"pipeline","ai.pipeline.type":"inference"}}`),
		{stdout: "", exitCode: 1}, // repo tags e2e/broken
		ok("latest\n"),            // repo tags plain
		ok(`{"mediaType":"application/vnd.oci.image.manifest.v1+json","digest":"` + searchDigestPlain + `","size":1}`),
		ok(`{"schemaVersion":2,"layers":[]}`), // no annotations at all
	})
}

func TestORASSearchWalksRegistryWithORAS(t *testing.T) {
	calls := installSearchRegistry(t)

	got, err := (&ORASProvider{}).Search("localhost:5000", NewSearchQuery())
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	want := []ManifestCandidate{
		{Reference: "localhost:5000/e2e/model:v1", Digest: searchDigestModel, ArtifactType: "model"},
		{Reference: "localhost:5000/e2e/model:v2", Digest: searchDigestPipeline, ArtifactType: "pipeline"},
	}
	if len(got) != len(want) {
		t.Fatalf("Search() returned %d candidates, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i].Reference != want[i].Reference || got[i].Digest != want[i].Digest || got[i].ArtifactType != want[i].ArtifactType {
			t.Errorf("candidate %d = {%s %s %s}, want {%s %s %s}", i,
				got[i].Reference, got[i].Digest, got[i].ArtifactType,
				want[i].Reference, want[i].Digest, want[i].ArtifactType)
		}
	}
	if got[0].Annotations[AnnotationRuntime] != "vllm" || len(got[0].Manifest) == 0 {
		t.Errorf("candidate 0 lost its annotations or manifest: %+v", got[0])
	}

	wantArgs := []string{
		"repo ls localhost:5000",
		"repo tags localhost:5000/e2e/model",
		"manifest fetch --descriptor localhost:5000/e2e/model:v1",
		"manifest fetch localhost:5000/e2e/model:v1",
		"manifest fetch --descriptor localhost:5000/e2e/model:v2",
		"manifest fetch localhost:5000/e2e/model:v2",
		"repo tags localhost:5000/e2e/broken",
		"repo tags localhost:5000/plain",
		"manifest fetch --descriptor localhost:5000/plain:latest",
		"manifest fetch localhost:5000/plain:latest",
	}
	recorded := calls()
	if len(recorded) != len(wantArgs) {
		t.Fatalf("expected %d oras invocations, got %d: %v", len(wantArgs), len(recorded), recorded)
	}
	for i, want := range wantArgs {
		_, args, _ := strings.Cut(recorded[i], "\t")
		if args != want {
			t.Errorf("call %d args = %q, want %q", i, args, want)
		}
	}
}

func TestORASSearchAppliesQueryFilters(t *testing.T) {
	cases := []struct {
		name  string
		query func(q *SearchQuery)
		want  []string
	}{
		{"type model", func(q *SearchQuery) { q.ArtifactType = "model" }, []string{"localhost:5000/e2e/model:v1"}},
		{"type pipeline", func(q *SearchQuery) { q.ArtifactType = "pipeline" }, []string{"localhost:5000/e2e/model:v2"}},
		{"type without matches", func(q *SearchQuery) { q.ArtifactType = "skill" }, nil},
		{"metadata match", func(q *SearchQuery) { q.MetadataFilters["ai.model.type"] = "llm" }, []string{"localhost:5000/e2e/model:v1"}},
		{"metadata value mismatch", func(q *SearchQuery) { q.MetadataFilters["ai.model.type"] = "embedding" }, nil},
		{"all metadata pairs must match", func(q *SearchQuery) {
			q.MetadataFilters["ai.model.type"] = "llm"
			q.MetadataFilters["ai.pipeline.type"] = "inference"
		}, nil},
		{"type and metadata", func(q *SearchQuery) {
			q.ArtifactType = "model"
			q.MetadataFilters[AnnotationRuntime] = "vllm"
		}, []string{"localhost:5000/e2e/model:v1"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			installSearchRegistry(t)
			query := NewSearchQuery()
			tc.query(query)

			got, err := (&ORASProvider{}).Search("localhost:5000", query)
			if err != nil {
				t.Fatalf("Search() error = %v", err)
			}
			var refs []string
			for _, c := range got {
				refs = append(refs, c.Reference)
			}
			if strings.Join(refs, ",") != strings.Join(tc.want, ",") {
				t.Errorf("Search() = %v, want %v", refs, tc.want)
			}
		})
	}
}

func TestORASSearchFailsWhenRepositoriesCannotBeListed(t *testing.T) {
	installFakeOras(t, "", 1)
	_, err := (&ORASProvider{}).Search("localhost:5000", NewSearchQuery())
	if err == nil || !strings.Contains(err.Error(), "failed to list repositories of localhost:5000") {
		t.Errorf("Search() error = %v, want a repository listing failure", err)
	}
}

func TestORASSearchSkipsUnfetchableManifest(t *testing.T) {
	ok := func(stdout string) fakeOrasResponse { return fakeOrasResponse{stdout: stdout} }
	installFakeOrasResponses(t, []fakeOrasResponse{
		ok("e2e/model\n"),
		ok("v1\nv2\n"),
		{stdout: "", exitCode: 1}, // descriptor of v1 fails
		ok(`{"digest":"` + searchDigestPipeline + `"}`),
		ok(`{"annotations":{"` + AnnotationArtifactType + `":"pipeline"}}`),
	})

	got, err := (&ORASProvider{}).Search("localhost:5000", NewSearchQuery())
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(got) != 1 || got[0].Reference != "localhost:5000/e2e/model:v2" {
		t.Errorf("Search() = %+v, want only e2e/model:v2", got)
	}
}
