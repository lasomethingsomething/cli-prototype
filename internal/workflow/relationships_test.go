package workflow

import (
	"encoding/json"
	"testing"
)

func TestNewRelationshipGraph(t *testing.T) {
	graph := NewRelationshipGraph()

	if graph.Models == nil {
		t.Error("Models map should be initialized")
	}
	if graph.Skills == nil {
		t.Error("Skills map should be initialized")
	}
	if graph.Pipelines == nil {
		t.Error("Pipelines map should be initialized")
	}
	if graph.Datasets == nil {
		t.Error("Datasets map should be initialized")
	}
}

func TestAddModel(t *testing.T) {
	graph := NewRelationshipGraph()

	node := graph.AddModel("model:v1", "sha256:abc123", "llm", "pytorch")

	if node == nil {
		t.Fatal("AddModel should return a node")
	}

	if node.Reference != "model:v1" {
		t.Errorf("Expected reference 'model:v1', got '%s'", node.Reference)
	}
	if node.SHA256 != "sha256:abc123" {
		t.Errorf("Expected sha256 'sha256:abc123', got '%s'", node.SHA256)
	}
	if node.Type != "llm" {
		t.Errorf("Expected type 'llm', got '%s'", node.Type)
	}
	if node.Framework != "pytorch" {
		t.Errorf("Expected framework 'pytorch', got '%s'", node.Framework)
	}

	if _, exists := graph.Models["model:v1"]; !exists {
		t.Error("Model should be in the graph")
	}
}

func TestAddSkill(t *testing.T) {
	graph := NewRelationshipGraph()

	node := graph.AddSkill("skill:v1", "sha256:def456", "rag")

	if node == nil {
		t.Fatal("AddSkill should return a node")
	}

	if node.Reference != "skill:v1" {
		t.Errorf("Expected reference 'skill:v1', got '%s'", node.Reference)
	}
	if node.SHA256 != "sha256:def456" {
		t.Errorf("Expected sha256 'sha256:def456', got '%s'", node.SHA256)
	}
	if node.Type != "rag" {
		t.Errorf("Expected type 'rag', got '%s'", node.Type)
	}

	if _, exists := graph.Skills["skill:v1"]; !exists {
		t.Error("Skill should be in the graph")
	}
}

func TestAddPipeline(t *testing.T) {
	graph := NewRelationshipGraph()

	stages := []string{"preprocess", "inference", "postprocess"}
	node := graph.AddPipeline("pipeline:v1", "sha256:ghi789", "inference", stages)

	if node == nil {
		t.Fatal("AddPipeline should return a node")
	}

	if node.Reference != "pipeline:v1" {
		t.Errorf("Expected reference 'pipeline:v1', got '%s'", node.Reference)
	}
	if node.SHA256 != "sha256:ghi789" {
		t.Errorf("Expected sha256 'sha256:ghi789', got '%s'", node.SHA256)
	}
	if node.Type != "inference" {
		t.Errorf("Expected type 'inference', got '%s'", node.Type)
	}
	if len(node.Stages) != 3 {
		t.Errorf("Expected 3 stages, got %d", len(node.Stages))
	}

	if _, exists := graph.Pipelines["pipeline:v1"]; !exists {
		t.Error("Pipeline should be in the graph")
	}
}

func TestAddDataset(t *testing.T) {
	graph := NewRelationshipGraph()

	node := graph.AddDataset("dataset:v1", "sha256:jkl012", "training")

	if node == nil {
		t.Fatal("AddDataset should return a node")
	}

	if node.Reference != "dataset:v1" {
		t.Errorf("Expected reference 'dataset:v1', got '%s'", node.Reference)
	}
	if node.Type != "training" {
		t.Errorf("Expected type 'training', got '%s'", node.Type)
	}

	if _, exists := graph.Datasets["dataset:v1"]; !exists {
		t.Error("Dataset should be in the graph")
	}
}

func TestAddModelToSkill(t *testing.T) {
	graph := NewRelationshipGraph()
	graph.AddModel("model:v1", "", "", "")
	graph.AddSkill("skill:v1", "", "")

	err := graph.AddModelToSkill("skill:v1", "model:v1")
	if err != nil {
		t.Fatalf("AddModelToSkill failed: %v", err)
	}

	skill := graph.Skills["skill:v1"]
	if len(skill.RequiresModels) != 1 {
		t.Errorf("Expected 1 required model, got %d", len(skill.RequiresModels))
	}
	if skill.RequiresModels[0] != "model:v1" {
		t.Errorf("Expected 'model:v1', got '%s'", skill.RequiresModels[0])
	}
	if len(skill.DependsOn) != 1 {
		t.Errorf("Expected 1 dependency, got %d", len(skill.DependsOn))
	}

	model := graph.Models["model:v1"]
	if len(model.UsedBy) != 1 {
		t.Errorf("Expected 1 user, got %d", len(model.UsedBy))
	}
	if model.UsedBy[0] != "skill:v1" {
		t.Errorf("Expected 'skill:v1', got '%s'", model.UsedBy[0])
	}
}

func TestAddSkillToPipeline(t *testing.T) {
	graph := NewRelationshipGraph()
	graph.AddSkill("skill:v1", "", "")
	graph.AddPipeline("pipeline:v1", "", "", []string{})

	err := graph.AddSkillToPipeline("pipeline:v1", "skill:v1")
	if err != nil {
		t.Fatalf("AddSkillToPipeline failed: %v", err)
	}

	pipeline := graph.Pipelines["pipeline:v1"]
	if len(pipeline.UsesSkills) != 1 {
		t.Errorf("Expected 1 used skill, got %d", len(pipeline.UsesSkills))
	}
	if pipeline.UsesSkills[0] != "skill:v1" {
		t.Errorf("Expected 'skill:v1', got '%s'", pipeline.UsesSkills[0])
	}

	skill := graph.Skills["skill:v1"]
	if len(skill.UsedBy) != 1 {
		t.Errorf("Expected 1 user, got %d", len(skill.UsedBy))
	}
	if skill.UsedBy[0] != "pipeline:v1" {
		t.Errorf("Expected 'pipeline:v1', got '%s'", skill.UsedBy[0])
	}
}

func TestAddModelToPipeline(t *testing.T) {
	graph := NewRelationshipGraph()
	graph.AddModel("model:v1", "", "", "")
	graph.AddPipeline("pipeline:v1", "", "", []string{})

	err := graph.AddModelToPipeline("pipeline:v1", "model:v1")
	if err != nil {
		t.Fatalf("AddModelToPipeline failed: %v", err)
	}

	pipeline := graph.Pipelines["pipeline:v1"]
	if len(pipeline.UsesModels) != 1 {
		t.Errorf("Expected 1 used model, got %d", len(pipeline.UsesModels))
	}
	if pipeline.UsesModels[0] != "model:v1" {
		t.Errorf("Expected 'model:v1', got '%s'", pipeline.UsesModels[0])
	}

	model := graph.Models["model:v1"]
	if len(model.UsedBy) != 1 {
		t.Errorf("Expected 1 user, got %d", len(model.UsedBy))
	}
}

func TestToJSONString(t *testing.T) {
	graph := NewRelationshipGraph()
	graph.AddModel("model:v1", "sha256:abc123", "llm", "pytorch")
	graph.AddSkill("skill:v1", "sha256:def456", "rag")
	graph.AddPipeline("pipeline:v1", "sha256:ghi789", "inference", []string{"preprocess", "inference"})
	graph.AddModelToSkill("skill:v1", "model:v1")
	graph.AddSkillToPipeline("pipeline:v1", "skill:v1")

	jsonStr, err := graph.ToJSONString()
	if err != nil {
		t.Fatalf("ToJSONString failed: %v", err)
	}

	// Verify it's valid JSON
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		t.Fatalf("Generated string is not valid JSON: %v", err)
	}

	// Check that it has the expected structure
	if _, ok := data["model"]; !ok {
		t.Error("Expected 'model' key in JSON")
	}
	if _, ok := data["skill"]; !ok {
		t.Error("Expected 'skill' key in JSON")
	}
	if _, ok := data["pipeline"]; !ok {
		t.Error("Expected 'pipeline' key in JSON")
	}
}

func TestFromJSONString(t *testing.T) {
	jsonStr := `{
		"model": {
			"model:v1": {
				"ref": "model:v1",
				"sha256": "sha256:abc123",
				"type": "llm",
				"framework": "pytorch"
			}
		},
		"skill": {
			"skill:v1": {
				"ref": "skill:v1",
				"type": "rag",
				"requires_models": ["model:v1"]
			}
		}
	}`

	graph, err := FromJSONString(jsonStr)
	if err != nil {
		t.Fatalf("FromJSONString failed: %v", err)
	}

	if len(graph.Models) != 1 {
		t.Errorf("Expected 1 model, got %d", len(graph.Models))
	}
	if _, exists := graph.Models["model:v1"]; !exists {
		t.Error("Expected model 'model:v1' to exist")
	}

	if len(graph.Skills) != 1 {
		t.Errorf("Expected 1 skill, got %d", len(graph.Skills))
	}
	if _, exists := graph.Skills["skill:v1"]; !exists {
		t.Error("Expected skill 'skill:v1' to exist")
	}

	skill := graph.Skills["skill:v1"]
	if len(skill.RequiresModels) != 1 {
		t.Errorf("Expected 1 required model, got %d", len(skill.RequiresModels))
	}
}

func TestGenerateRelationshipGraphAnnotation(t *testing.T) {
	graph := NewRelationshipGraph()
	graph.AddModel("model:v1", "sha256:abc123", "llm", "pytorch")

	annotation, err := GenerateRelationshipGraphAnnotation(graph)
	if err != nil {
		t.Fatalf("GenerateRelationshipGraphAnnotation failed: %v", err)
	}

	if annotation == "" {
		t.Error("Expected non-empty annotation")
	}

	// Verify it's valid JSON
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(annotation), &data); err != nil {
		t.Error("Generated annotation is not valid JSON")
	}
}

func TestGenerateRelationshipGraphAnnotationEmpty(t *testing.T) {
	graph := NewRelationshipGraph()

	annotation, err := GenerateRelationshipGraphAnnotation(graph)
	if err != nil {
		t.Fatalf("GenerateRelationshipGraphAnnotation failed: %v", err)
	}

	if annotation != "" {
		t.Error("Expected empty annotation for empty graph")
	}
}

func TestGenerateRelationshipGraphAnnotationNil(t *testing.T) {
	annotation, err := GenerateRelationshipGraphAnnotation(nil)
	if err != nil {
		t.Fatalf("GenerateRelationshipGraphAnnotation failed: %v", err)
	}

	if annotation != "" {
		t.Error("Expected empty annotation for nil graph")
	}
}

func TestParseRelationshipGraphFromAnnotation(t *testing.T) {
	jsonStr := `{"model":{"model:v1":{"ref":"model:v1","type":"llm"}}}`

	graph, err := ParseRelationshipGraphFromAnnotation(jsonStr)
	if err != nil {
		t.Fatalf("ParseRelationshipGraphFromAnnotation failed: %v", err)
	}

	if len(graph.Models) != 1 {
		t.Errorf("Expected 1 model, got %d", len(graph.Models))
	}
}

func TestParseRelationshipGraphFromAnnotationEmpty(t *testing.T) {
	graph, err := ParseRelationshipGraphFromAnnotation("")
	if err != nil {
		t.Fatalf("ParseRelationshipGraphFromAnnotation failed: %v", err)
	}

	if graph == nil {
		t.Error("Expected non-nil graph")
	}
	if len(graph.Models) != 0 {
		t.Error("Expected empty graph")
	}
}

func TestString(t *testing.T) {
	graph := NewRelationshipGraph()
	graph.AddModel("model:v1", "sha256:abc123", "llm", "pytorch")
	graph.AddSkill("skill:v1", "sha256:def456", "rag")
	graph.AddModelToSkill("skill:v1", "model:v1")

	str := graph.String()

	if !relStringContains(str, "Models:") {
		t.Error("Expected 'Models:' in string output")
	}
	if !relStringContains(str, "model:v1") {
		t.Error("Expected 'model:v1' in string output")
	}
	if !relStringContains(str, "Skills:") {
		t.Error("Expected 'Skills:' in string output")
	}
	if !relStringContains(str, "skill:v1") {
		t.Error("Expected 'skill:v1' in string output")
	}
	if !relStringContains(str, "Requires models") {
		t.Error("Expected relationship info in string output")
	}
}

func relStringContains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && relStringContainsHelper(s, substr))
}

func relStringContainsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestErrorHandling(t *testing.T) {
	graph := NewRelationshipGraph()

	// Test error when skill not found
	err := graph.AddModelToSkill("nonexistent:v1", "model:v1")
	if err == nil {
		t.Error("Expected error when skill not found")
	}

	// Test error when model not found
	graph.AddSkill("skill:v1", "", "")
	err = graph.AddModelToSkill("skill:v1", "nonexistent:v1")
	if err == nil {
		t.Error("Expected error when model not found")
	}

	// Test error when pipeline not found
	err = graph.AddSkillToPipeline("nonexistent:v1", "skill:v1")
	if err == nil {
		t.Error("Expected error when pipeline not found")
	}

	// Test error when skill not found for pipeline
	graph.AddPipeline("pipeline:v1", "", "", []string{})
	err = graph.AddSkillToPipeline("pipeline:v1", "nonexistent:v1")
	if err == nil {
		t.Error("Expected error when skill not found")
	}
}

func TestComplexGraph(t *testing.T) {
	// Test a complex relationship graph matching the story example
	// {
	//   "ai.relationships": {
	//     "model": { "sha256": "abc123", "used_by": ["skill:sha256:def456"] },
	//     "skill": { "sha256": "def456", "used_by": ["pipeline:sha256:ghi789"] }
	//   }
	// }

	graph := NewRelationshipGraph()

	// Add model
	modelNode := graph.AddModel("model:sha256:abc123", "sha256:abc123", "", "")

	// Add skill
	skillNode := graph.AddSkill("skill:sha256:def456", "sha256:def456", "")

	// Add pipeline
	pipelineNode := graph.AddPipeline("pipeline:sha256:ghi789", "sha256:ghi789", "", []string{})

	// Create relationships: skill requires model, pipeline uses skill
	graph.AddModelToSkill("skill:sha256:def456", "model:sha256:abc123")
	graph.AddSkillToPipeline("pipeline:sha256:ghi789", "skill:sha256:def456")

	// Verify model is used by skill
	if len(modelNode.UsedBy) != 1 || modelNode.UsedBy[0] != "skill:sha256:def456" {
		t.Error("Model should be used by skill")
	}

	// Verify skill requires model and is used by pipeline
	if len(skillNode.RequiresModels) != 1 || skillNode.RequiresModels[0] != "model:sha256:abc123" {
		t.Error("Skill should require model")
	}
	if len(skillNode.UsedBy) != 1 || skillNode.UsedBy[0] != "pipeline:sha256:ghi789" {
		t.Error("Skill should be used by pipeline")
	}

	// Verify pipeline uses skill
	if len(pipelineNode.UsesSkills) != 1 || pipelineNode.UsesSkills[0] != "skill:sha256:def456" {
		t.Error("Pipeline should use skill")
	}

	// Generate JSON and verify structure
	jsonStr, err := graph.ToJSONString()
	if err != nil {
		t.Fatalf("ToJSONString failed: %v", err)
	}

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}

	// Check structure
	if models, ok := data["model"].(map[string]interface{}); ok {
		if model, ok := models["model:sha256:abc123"].(map[string]interface{}); ok {
			if usedBy, ok := model["used_by"].([]interface{}); ok {
				if len(usedBy) != 1 {
					t.Errorf("Expected 1 user for model, got %d", len(usedBy))
				}
			}
		}
	}
}
