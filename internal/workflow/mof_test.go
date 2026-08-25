package workflow

import (
	"os"
	"path/filepath"
	"testing"
)

// TestMOFClassifier tests the MOF classifier interface
func TestMOFClassifier(t *testing.T) {
	classifier := GetMOFClassifier()

	if classifier.Name() != "mof" {
		t.Errorf("Expected name 'mof', got '%s'", classifier.Name())
	}
}

// TestMOFClassConstants tests the MOF class constants
func TestMOFClassConstants(t *testing.T) {
	if MOFClassI != "I" {
		t.Errorf("Expected MOFClassI to be 'I', got '%s'", MOFClassI)
	}
	if MOFClassII != "II" {
		t.Errorf("Expected MOFClassII to be 'II', got '%s'", MOFClassII)
	}
	if MOFClassIII != "III" {
		t.Errorf("Expected MOFClassIII to be 'III', got '%s'", MOFClassIII)
	}
}

// TestClassifyModelPathClassI tests classification for Class I (fully open)
func TestClassifyModelPathClassI(t *testing.T) {
	// Create a temporary directory with all components
	tmpDir, err := os.MkdirTemp("", "mof-test-class-I")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create weights file
	if err := os.WriteFile(filepath.Join(tmpDir, "model.gguf"), []byte("weights"), 0644); err != nil {
		t.Fatalf("Failed to create weights file: %v", err)
	}

	// Create training script (code)
	if err := os.WriteFile(filepath.Join(tmpDir, "train.py"), []byte("training code"), 0644); err != nil {
		t.Fatalf("Failed to create code file: %v", err)
	}

	// Create training data
	if err := os.WriteFile(filepath.Join(tmpDir, "training_data.json"), []byte("data"), 0644); err != nil {
		t.Fatalf("Failed to create training data: %v", err)
	}

	// Create README (documentation)
	if err := os.WriteFile(filepath.Join(tmpDir, "README.md"), []byte("# Model"), 0644); err != nil {
		t.Fatalf("Failed to create README: %v", err)
	}

	// Create LICENSE
	if err := os.WriteFile(filepath.Join(tmpDir, "LICENSE"), []byte("MIT"), 0644); err != nil {
		t.Fatalf("Failed to create LICENSE: %v", err)
	}

	classifier := &MOFClassifierImpl{}
	classStr, err := classifier.Classify(tmpDir)
	if err != nil {
		t.Fatalf("Classify() error = %v", err)
	}

	if classStr != "I" {
		t.Errorf("Expected class 'I' for fully open model, got '%s'", classStr)
	}
}

// TestClassifyModelPathClassII tests classification for Class II (partially open)
func TestClassifyModelPathClassII(t *testing.T) {
	// Create a temporary directory with only weights and README
	tmpDir, err := os.MkdirTemp("", "mof-test-class-II")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create weights file
	if err := os.WriteFile(filepath.Join(tmpDir, "model.gguf"), []byte("weights"), 0644); err != nil {
		t.Fatalf("Failed to create weights file: %v", err)
	}

	// Create README (documentation)
	if err := os.WriteFile(filepath.Join(tmpDir, "README.md"), []byte("# Model"), 0644); err != nil {
		t.Fatalf("Failed to create README: %v", err)
	}

	// Don't create code, training data, or license

	classifier := &MOFClassifierImpl{}
	classStr, err := classifier.Classify(tmpDir)
	if err != nil {
		t.Fatalf("Classify() error = %v", err)
	}

	if classStr != "II" {
		t.Errorf("Expected class 'II' for partially open model, got '%s'", classStr)
	}
}

// TestClassifyModelPathClassIII tests classification for Class III (closed)
func TestClassifyModelPathClassIII(t *testing.T) {
	// Create a temporary directory with only weights
	tmpDir, err := os.MkdirTemp("", "mof-test-class-III")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create weights file only
	if err := os.WriteFile(filepath.Join(tmpDir, "model.gguf"), []byte("weights"), 0644); err != nil {
		t.Fatalf("Failed to create weights file: %v", err)
	}

	classifier := &MOFClassifierImpl{}
	classStr, err := classifier.Classify(tmpDir)
	if err != nil {
		t.Fatalf("Classify() error = %v", err)
	}

	if classStr != "III" {
		t.Errorf("Expected class 'III' for closed model, got '%s'", classStr)
	}
}

// TestClassifyModelPathWithDirectoryStructure tests classification with directory patterns
func TestClassifyModelPathWithDirectoryStructure(t *testing.T) {
	// Create a temporary directory with directory structure
	tmpDir, err := os.MkdirTemp("", "mof-test-dirs")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create weights file
	if err := os.WriteFile(filepath.Join(tmpDir, "model.gguf"), []byte("weights"), 0644); err != nil {
		t.Fatalf("Failed to create weights file: %v", err)
	}

	// Create training data directory
	trainDir := filepath.Join(tmpDir, "training_data")
	if err := os.MkdirAll(trainDir, 0755); err != nil {
		t.Fatalf("Failed to create training data directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(trainDir, "data.json"), []byte("data"), 0644); err != nil {
		t.Fatalf("Failed to create training data file: %v", err)
	}

	// Create src directory with code
	srcDir := filepath.Join(tmpDir, "src")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatalf("Failed to create src directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "train.py"), []byte("code"), 0644); err != nil {
		t.Fatalf("Failed to create code file: %v", err)
	}

	// Create LICENSE
	if err := os.WriteFile(filepath.Join(tmpDir, "LICENSE"), []byte("MIT"), 0644); err != nil {
		t.Fatalf("Failed to create LICENSE: %v", err)
	}

	// Create README
	if err := os.WriteFile(filepath.Join(tmpDir, "README.md"), []byte("# Model"), 0644); err != nil {
		t.Fatalf("Failed to create README: %v", err)
	}

	classifier := &MOFClassifierImpl{}
	classStr, err := classifier.Classify(tmpDir)
	if err != nil {
		t.Fatalf("Classify() error = %v", err)
	}

	// Should be Class I with all components
	if classStr != "I" {
		t.Errorf("Expected class 'I' with directory structure, got '%s'", classStr)
	}
}

// TestClassifyModelPathNonExistent tests classification with non-existent path
func TestClassifyModelPathNonExistent(t *testing.T) {
	classifier := &MOFClassifierImpl{}
	_, err := classifier.Classify("/nonexistent/path")
	if err == nil {
		t.Error("Expected error for non-existent path")
	}
}

// TestClassifyModelPathEmptyDirectory tests classification with empty directory
func TestClassifyModelPathEmptyDirectory(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mof-test-empty")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	classifier := &MOFClassifierImpl{}
	classStr, err := classifier.Classify(tmpDir)
	if err != nil {
		t.Fatalf("Classify() error = %v", err)
	}

	// Empty directory should be Class III
	if classStr != "III" {
		t.Errorf("Expected class 'III' for empty directory, got '%s'", classStr)
	}
}

// TestClassificationResult tests the ClassificationResult struct
func TestClassificationResult(t *testing.T) {
	result := &ClassificationResult{
		Class:           MOFClassI,
		HasWeights:      true,
		HasCode:         true,
		HasTrainingData: true,
		HasDocs:         true,
		HasLicense:      true,
		Components:      []string{"weights", "code", "training-data", "documentation", "license"},
	}

	if result.Class != "I" {
		t.Errorf("Expected Class I, got %s", result.Class)
	}
}

// TestDetermineMOFClass tests the classification logic
func TestDetermineMOFClass(t *testing.T) {
	classifier := &MOFClassifierImpl{}

	// Test Class I
	resultI := &ClassificationResult{
		HasWeights:      true,
		HasCode:         true,
		HasTrainingData: true,
		HasDocs:         true,
		HasLicense:      true,
	}
	if classifier.determineMOFClass(resultI) != MOFClassI {
		t.Error("Expected Class I for all components")
	}

	// Test Class II
	resultII := &ClassificationResult{
		HasWeights:      true,
		HasCode:         true,
		HasTrainingData: false,
		HasDocs:         false,
		HasLicense:      false,
	}
	if classifier.determineMOFClass(resultII) != MOFClassII {
		t.Error("Expected Class II for partial components")
	}

	// Test Class III
	resultIII := &ClassificationResult{
		HasWeights:      true,
		HasCode:         false,
		HasTrainingData: false,
		HasDocs:         false,
		HasLicense:      false,
	}
	if classifier.determineMOFClass(resultIII) != MOFClassIII {
		t.Error("Expected Class III for weights only")
	}

	// Test Class III with no weights
	resultEmpty := &ClassificationResult{
		HasWeights:      false,
		HasCode:         false,
		HasTrainingData: false,
		HasDocs:         false,
		HasLicense:      false,
	}
	if classifier.determineMOFClass(resultEmpty) != MOFClassIII {
		t.Error("Expected Class III for no components")
	}
}

// TestFileTypeDetection tests the file type detection functions
func TestFileTypeDetection(t *testing.T) {
	// Test weight files
	if !isWeightFile("model.gguf") {
		t.Error("model.gguf should be detected as weight file")
	}
	if !isWeightFile("model.pt") {
		t.Error("model.pt should be detected as weight file")
	}
	if !isWeightFile("model.safetensors") {
		t.Error("model.safetensors should be detected as weight file")
	}

	// Test code files
	if !isCodeFile("train.py") {
		t.Error("train.py should be detected as code file")
	}
	if !isCodeFile("setup.py") {
		t.Error("setup.py should be detected as code file")
	}
	if !isCodeFile("requirements.txt") {
		t.Error("requirements.txt should be detected as code file")
	}

	// Test documentation files
	if !isDocFile("README.md") {
		t.Error("README.md should be detected as documentation")
	}
	if !isDocFile("CHANGELOG.md") {
		t.Error("CHANGELOG.md should be detected as documentation")
	}

	// Test license files
	if !isLicenseFile("LICENSE") {
		t.Error("LICENSE should be detected as license")
	}
	if !isLicenseFile("MIT-LICENSE.txt") {
		t.Error("MIT-LICENSE.txt should be detected as license")
	}

	// Test training data files
	if !isTrainingDataFile("training.json", "/data/training.json") {
		t.Error("training.json should be detected as training data")
	}
	if !isTrainingDataFile("dataset.csv", "/data/dataset.csv") {
		t.Error("dataset.csv should be detected as training data")
	}
}

// TestDirectoryPatternDetection tests the directory pattern detection
func TestDirectoryPatternDetection(t *testing.T) {
	// Test training data directories
	if !isTrainingDataDir("training_data") {
		t.Error("training_data should be detected as training data directory")
	}
	if !isTrainingDataDir("dataset") {
		t.Error("dataset should be detected as training data directory")
	}

	// Test code directories
	if !isCodeDir("src") {
		t.Error("src should be detected as code directory")
	}
	if !isCodeDir("scripts") {
		t.Error("scripts should be detected as code directory")
	}
}

// TestAppendUnique tests the appendUnique helper function
func TestAppendUnique(t *testing.T) {
	slice := []string{"a", "b"}

	// Test appending new value
	result := appendUnique(slice, "c")
	if len(result) != 3 {
		t.Errorf("Expected length 3, got %d", len(result))
	}

	// Test appending duplicate
	result = appendUnique(slice, "a")
	if len(result) != 2 {
		t.Errorf("Expected length 2 (no duplicate), got %d", len(result))
	}
}

// TestClassifyModelPathWithVariousFormats tests various model file formats
func TestClassifyModelPathWithVariousFormats(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mof-test-formats")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create various weight file formats
	weightFiles := []string{
		"model.gguf", "model.pt", "model.safetensors",
		"model.h5", "model.pkl", "model.bin",
		"model.onnx", "model.pb", "model.tflite",
	}

	for _, file := range weightFiles {
		if err := os.WriteFile(filepath.Join(tmpDir, file), []byte("weights"), 0644); err != nil {
			t.Fatalf("Failed to create weight file %s: %v", file, err)
		}
	}

	// Create README and LICENSE
	if err := os.WriteFile(filepath.Join(tmpDir, "README.md"), []byte("# Model"), 0644); err != nil {
		t.Fatalf("Failed to create README: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "LICENSE"), []byte("MIT"), 0644); err != nil {
		t.Fatalf("Failed to create LICENSE: %v", err)
	}

	classifier := &MOFClassifierImpl{}
	classStr, err := classifier.Classify(tmpDir)
	if err != nil {
		t.Fatalf("Classify() error = %v", err)
	}

	// Should detect weights and docs, but not code or training data
	// So should be Class II
	if classStr != "II" {
		t.Errorf("Expected class 'II' for weights+docs only, got '%s'", classStr)
	}
}

// TestGenerateExplanation tests the explanation generation
func TestGenerateExplanation(t *testing.T) {
	classifier := &MOFClassifierImpl{}

	// Test Class I explanation
	resultI := &ClassificationResult{
		Class: MOFClassI,
	}
	explanation := classifier.generateExplanation(resultI)
	if !mofStringContains(explanation, "Fully open") {
		t.Errorf("Expected Class I explanation to contain 'Fully open', got '%s'", explanation)
	}

	// Test Class II explanation
	resultII := &ClassificationResult{
		Class:           MOFClassII,
		HasWeights:      true,
		HasCode:         false,
		HasTrainingData: false,
		HasDocs:         true,
		HasLicense:      false,
		Components:      []string{"weights", "documentation"},
	}
	explanation = classifier.generateExplanation(resultII)
	if !mofStringContains(explanation, "Partially open") {
		t.Errorf("Expected Class II explanation to contain 'Partially open', got '%s'", explanation)
	}

	// Test Class III explanation
	resultIII := &ClassificationResult{
		Class:           MOFClassIII,
		HasWeights:      true,
		HasCode:         false,
		HasTrainingData: false,
		HasDocs:         false,
		HasLicense:      false,
	}
	explanation = classifier.generateExplanation(resultIII)
	if !mofStringContains(explanation, "Closed") {
		t.Errorf("Expected Class III explanation to contain 'Closed', got '%s'", explanation)
	}
}

// mofStringContains is a helper function for string contains
func mofStringContains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 &&
		(s[0:len(substr)] == substr ||
			s[len(s)-len(substr):] == substr ||
			mofStringContainsHelper(s, substr)))
}

func mofStringContainsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestGetMOFClassifierFactory tests the factory function
func TestGetMOFClassifierFactory(t *testing.T) {
	classifier := GetMOFClassifier()
	if classifier.Name() != "mof" {
		t.Errorf("Expected classifier name 'mof', got '%s'", classifier.Name())
	}
}

// TestClassifyModelPathConvenience tests the convenience function
func TestClassifyModelPathConvenience(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mof-test-convenience")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create weights and license
	if err := os.WriteFile(filepath.Join(tmpDir, "model.gguf"), []byte("weights"), 0644); err != nil {
		t.Fatalf("Failed to create weights file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "LICENSE"), []byte("MIT"), 0644); err != nil {
		t.Fatalf("Failed to create LICENSE: %v", err)
	}

	classStr, result, err := ClassifyModelPath(tmpDir)
	if err != nil {
		t.Fatalf("ClassifyModelPath() error = %v", err)
	}

	if classStr != "III" {
		t.Errorf("Expected class 'III', got '%s'", classStr)
	}

	if result == nil {
		t.Error("Expected non-nil result")
	}

	if !result.HasWeights {
		t.Error("Expected HasWeights to be true")
	}

	if !result.HasLicense {
		t.Error("Expected HasLicense to be true")
	}
}
