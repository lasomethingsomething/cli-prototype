package workflow

import (
	"strings"
	"testing"
)

func TestSearchQueryMatchesAnnotations(t *testing.T) {
	annotations := map[string]string{
		AnnotationArtifactType: "model",
		"ai.model.type":        "llm",
		AnnotationRuntime:      "vllm",
	}
	cases := []struct {
		name  string
		query func(q *SearchQuery)
		want  bool
	}{
		{"no filters", func(q *SearchQuery) {}, true},
		{"matching type", func(q *SearchQuery) { q.ArtifactType = "model" }, true},
		{"other type", func(q *SearchQuery) { q.ArtifactType = "skill" }, false},
		{"matching metadata", func(q *SearchQuery) { q.MetadataFilters["ai.model.type"] = "llm" }, true},
		{"metadata value differs", func(q *SearchQuery) { q.MetadataFilters["ai.model.type"] = "embedding" }, false},
		{"metadata key missing", func(q *SearchQuery) { q.MetadataFilters["ai.model.framework"] = "pytorch" }, false},
		{"value match is exact", func(q *SearchQuery) { q.MetadataFilters["ai.model.type"] = "ll" }, false},
		{"every pair must match", func(q *SearchQuery) {
			q.MetadataFilters["ai.model.type"] = "llm"
			q.MetadataFilters[AnnotationRuntime] = "kserve"
		}, false},
		{"type and metadata", func(q *SearchQuery) {
			q.ArtifactType = "model"
			q.MetadataFilters[AnnotationRuntime] = "vllm"
		}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			q := NewSearchQuery()
			tc.query(q)
			if got := q.MatchesAnnotations(annotations); got != tc.want {
				t.Errorf("MatchesAnnotations() = %v, want %v", got, tc.want)
			}
		})
	}

	// A manifest without annotations only matches an unfiltered query.
	if !NewSearchQuery().MatchesAnnotations(nil) {
		t.Error("unfiltered query should match nil annotations")
	}
	q := NewSearchQuery()
	q.ArtifactType = "model"
	if q.MatchesAnnotations(nil) {
		t.Error("type filter should not match nil annotations")
	}
}

func TestNewManifestCandidate(t *testing.T) {
	const ref = "localhost:5000/e2e/model:v1"

	got, err := NewManifestCandidate(ref, testDigest, []byte(`{"schemaVersion":2,"annotations":{"`+AnnotationArtifactType+`":"model","ai.model.type":"llm"}}`))
	if err != nil {
		t.Fatalf("NewManifestCandidate() error = %v", err)
	}
	if got.Reference != ref || got.Digest != testDigest || got.ArtifactType != "model" {
		t.Errorf("candidate = %+v, want reference/digest/type filled in", got)
	}
	if got.Annotations["ai.model.type"] != "llm" || !strings.Contains(string(got.Manifest), "schemaVersion") {
		t.Errorf("candidate lost annotations or manifest: %+v", got)
	}

	// No annotations at all is not an error: Search skips such manifests.
	plain, err := NewManifestCandidate(ref, testDigest, []byte(`{"schemaVersion":2,"layers":[]}`))
	if err != nil {
		t.Fatalf("NewManifestCandidate() on a manifest without annotations: %v", err)
	}
	if plain.Annotations != nil || plain.ArtifactType != "" {
		t.Errorf("candidate without annotations = %+v, want nil annotations and no type", plain)
	}

	if _, err := NewManifestCandidate(ref, testDigest, []byte(`not json`)); err == nil {
		t.Error("NewManifestCandidate() should reject malformed JSON")
	}
}

func TestSearchResultsFromCandidates(t *testing.T) {
	graph := NewRelationshipGraph()
	graph.AddModel("m:v1", "", "llm", "")
	graph.AddPipeline("p:v1", "", "inference", nil)
	if err := graph.AddModelToPipeline("p:v1", "m:v1"); err != nil {
		t.Fatal(err)
	}
	relationships, err := GenerateRelationshipGraphAnnotation(graph)
	if err != nil {
		t.Fatal(err)
	}

	results := SearchResultsFromCandidates([]ManifestCandidate{
		{Reference: "r/m:v1", Digest: "sha256:aa", ArtifactType: "model", Annotations: map[string]string{
			AnnotationArtifactType: "model", "ai.model.type": "llm",
		}},
		{Reference: "r/p:v1", Digest: "sha256:bb", ArtifactType: "pipeline", Annotations: map[string]string{
			AnnotationArtifactType: "pipeline", AnnotationRelationshipGraph: relationships,
		}},
	})

	if results.TotalCount != 2 || len(results.Results) != 2 {
		t.Fatalf("got %d results (total %d), want 2", len(results.Results), results.TotalCount)
	}
	model := results.Results[0]
	if model.Reference != "r/m:v1" || model.Digest != "sha256:aa" || model.ArtifactType != "model" {
		t.Errorf("model result = %+v", model)
	}
	if model.Metadata["ai.model.type"] != "llm" {
		t.Errorf("model metadata = %v, want ai.model.type=llm", model.Metadata)
	}
	if model.RelationshipGraph != nil {
		t.Errorf("model without ai.relationships got a graph: %+v", model.RelationshipGraph)
	}

	pipeline := results.Results[1]
	if pipeline.RelationshipGraph == nil {
		t.Fatalf("pipeline result has no relationship graph: %+v", pipeline)
	}
	if uses := pipeline.RelationshipGraph.Pipelines["p:v1"].UsesModels; len(uses) != 1 || uses[0] != "m:v1" {
		t.Errorf("pipeline uses_models = %v, want [m:v1]", uses)
	}

	if empty := SearchResultsFromCandidates(nil); empty.TotalCount != 0 || empty.Results == nil {
		t.Errorf("empty candidates = %+v, want an empty (non-nil) result list", empty)
	}
}

func TestFilterByRelationshipUsesModel(t *testing.T) {
	withModel := func(pipelineRef, modelRef string) *RelationshipGraph {
		graph := NewRelationshipGraph()
		graph.AddModel(modelRef, "", "llm", "")
		graph.AddPipeline(pipelineRef, "", "inference", nil)
		if err := graph.AddModelToPipeline(pipelineRef, modelRef); err != nil {
			t.Fatal(err)
		}
		return graph
	}
	results := &SearchResults{}
	results.AddResult(SearchResult{Reference: "r/m:v1", ArtifactType: "model"}) // no graph
	results.AddResult(SearchResult{Reference: "r/p:v1", ArtifactType: "pipeline", RelationshipGraph: withModel("r/p:v1", "r/m:v1")})
	results.AddResult(SearchResult{Reference: "r/p:v2", ArtifactType: "pipeline", RelationshipGraph: withModel("r/p:v2", "r/other:v1")})

	references := func(r *SearchResults) string {
		var refs []string
		for _, result := range r.Results {
			refs = append(refs, result.Reference)
		}
		return strings.Join(refs, ",")
	}

	all := results.FilterByRelationship(NewSearchQuery())
	if got := references(all); got != "r/m:v1,r/p:v1,r/p:v2" {
		t.Errorf("no filters kept %q, want everything", got)
	}

	query := NewSearchQuery()
	query.UsesModel = "r/m:v1"
	filtered := results.FilterByRelationship(query)
	if got := references(filtered); got != "r/p:v1" {
		t.Errorf("--uses-model r/m:v1 kept %q, want only r/p:v1", got)
	}
	if filtered.TotalCount != 1 {
		t.Errorf("TotalCount = %d, want 1", filtered.TotalCount)
	}
}
