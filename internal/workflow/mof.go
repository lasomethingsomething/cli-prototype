package workflow

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// MOFClassifier defines the interface for Model Openness Framework classification
type MOFClassifier interface {
	Name() string
	Classify(modelPath string) (string, error)
}

// MOFClass represents the Model Openness Framework classification
type MOFClass string

const (
	// MOFClassI - Fully open: weights, code, training data, documentation and license all available
	MOFClassI MOFClass = "I"
	// MOFClassII - Partially open: weights plus at least one of code, training data or documentation
	MOFClassII MOFClass = "II"
	// MOFClassIII - Closed: weights only (a license alone does not add transparency)
	MOFClassIII MOFClass = "III"
)

// ClassificationResult contains detailed MOF classification information
type ClassificationResult struct {
	Class           MOFClass
	HasWeights      bool
	HasCode         bool
	HasTrainingData bool
	HasDocs         bool
	HasLicense      bool
	Components      []string
	Explanation     string
}

// --- MOF Classifier (Model Openness Framework) ---
// Based on: https://github.com/Adopt-MOF/MOF

type MOFClassifierImpl struct{}

func (m *MOFClassifierImpl) Name() string {
	return "mof"
}

// Classify determines the MOF class by inspecting the model directory
func (m *MOFClassifierImpl) Classify(modelPath string) (string, error) {
	fmt.Printf("Classifying model at %s with MOF...\n", modelPath)

	// Check if the path exists
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		return "", fmt.Errorf("model path does not exist: %s", modelPath)
	}

	// Perform classification
	result, err := m.classifyModelPath(modelPath)
	if err != nil {
		return "", err
	}

	fmt.Printf("MOF Classification: %s\n", result.Class)
	fmt.Printf("  Components found: %s\n", strings.Join(result.Components, ", "))
	fmt.Printf("  Explanation: %s\n", result.Explanation)

	return string(result.Class), nil
}

// classifyModelPath inspects the directory and determines MOF class
func (m *MOFClassifierImpl) classifyModelPath(modelPath string) (*ClassificationResult, error) {
	result := &ClassificationResult{
		Components: make([]string, 0),
	}

	// Walk the directory and check for MOF-relevant files
	err := filepath.Walk(modelPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		filename := info.Name()
		lowerName := strings.ToLower(filename)

		// Skip files that model-cli itself writes next to the model.
		if isGeneratedArtifact(lowerName) {
			return nil
		}

		// Each file counts as exactly one component, checked most-specific first,
		// so a weight file such as model.h5 is never also counted as code or data.
		switch {
		case isWeightFile(lowerName):
			result.HasWeights = true
			result.Components = appendUnique(result.Components, "weights")
		case isLicenseFile(lowerName):
			result.HasLicense = true
			result.Components = appendUnique(result.Components, "license")
		case isCodeFile(lowerName):
			result.HasCode = true
			result.Components = appendUnique(result.Components, "code")
		case isDocFile(lowerName):
			result.HasDocs = true
			result.Components = appendUnique(result.Components, "documentation")
		case isTrainingDataFile(lowerName, path):
			result.HasTrainingData = true
			result.Components = appendUnique(result.Components, "training-data")
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk model path: %w", err)
	}

	// Also check directory names for common patterns
	entries, err := os.ReadDir(modelPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read model directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			name := strings.ToLower(entry.Name())
			if !result.HasTrainingData && isTrainingDataDir(name) {
				result.HasTrainingData = true
				result.Components = appendUnique(result.Components, "training-data")
			}
			if !result.HasCode && isCodeDir(name) {
				result.HasCode = true
				result.Components = appendUnique(result.Components, "code")
			}
		}
	}

	// Determine class based on MOF specification
	result.Class = m.determineMOFClass(result)
	result.Explanation = m.generateExplanation(result)

	return result, nil
}

// determineMOFClass applies MOF classification rules
func (m *MOFClassifierImpl) determineMOFClass(result *ClassificationResult) MOFClass {
	// Class I: weights, code, training data, documentation and license all present.
	if result.HasWeights && result.HasCode && result.HasTrainingData && result.HasDocs && result.HasLicense {
		return MOFClassI
	}

	// Class II: weights plus at least one component that adds transparency
	// (code, training data or documentation). A license on its own does not.
	if result.HasWeights && (result.HasCode || result.HasTrainingData || result.HasDocs) {
		return MOFClassII
	}

	// Class III: weights only, or nothing recognised.
	return MOFClassIII
}

// generateExplanation creates a human-readable explanation
func (m *MOFClassifierImpl) generateExplanation(result *ClassificationResult) string {
	switch result.Class {
	case MOFClassI:
		return "Fully open: weights, code, training data, docs, and license all available"
	case MOFClassII:
		missing := []string{}
		if !result.HasWeights {
			missing = append(missing, "weights")
		}
		if !result.HasCode {
			missing = append(missing, "code")
		}
		if !result.HasTrainingData {
			missing = append(missing, "training data")
		}
		if !result.HasDocs {
			missing = append(missing, "documentation")
		}
		if !result.HasLicense {
			missing = append(missing, "license")
		}
		if len(missing) > 0 {
			return fmt.Sprintf("Partially open: has %s but missing %s",
				strings.Join(result.Components, ", "),
				strings.Join(missing, ", "))
		}
		return "Partially open: has multiple components"
	case MOFClassIII:
		if result.HasWeights {
			return "Closed: weights available but no transparency into training process"
		}
		return "Closed: minimal or no transparency"
	default:
		return "Unknown classification"
	}
}

// isWeightFile checks if a filename is a model weight file
func isWeightFile(filename string) bool {
	weightExtensions := []string{
		".bin", ".pt", ".pth", ".ckpt", ".safetensors", ".gguf",
		".h5", ".hdf5", ".pkl", ".pickle", ".npz", ".npy",
		".tflite", ".pb", ".onnx", ".meta", ".params",
	}
	if hasAnySuffix(filename, weightExtensions) {
		return true
	}
	return strings.Contains(filename, "model") || strings.Contains(filename, "weights")
}

// isCodeFile checks if a filename is a code file
func isCodeFile(filename string) bool {
	codeExtensions := []string{
		".py", ".js", ".ts", ".java", ".cpp", ".c", ".h", ".hpp",
		".go", ".rs", ".rb", ".php", ".sh", ".bash",
	}
	if hasAnySuffix(filename, codeExtensions) {
		return true
	}
	codeFiles := []string{
		"makefile", "dockerfile", "requirements.txt",
		"setup.py", "pyproject.toml", "package.json",
	}
	for _, name := range codeFiles {
		if filename == name {
			return true
		}
	}
	return false
}

// hasAnySuffix reports whether filename ends with any of the given suffixes.
func hasAnySuffix(filename string, suffixes []string) bool {
	for _, suffix := range suffixes {
		if strings.HasSuffix(filename, suffix) {
			return true
		}
	}
	return false
}

// isGeneratedArtifact reports whether a file was produced by model-cli itself
// (manifest, attestation, SBOM) and should not influence classification.
func isGeneratedArtifact(filename string) bool {
	return filename == "manifest.json" || filename == "attestation.json" || strings.HasPrefix(filename, "sbom.")
}

// isTrainingDataFile checks if a filename is training data
func isTrainingDataFile(filename string, fullPath string) bool {
	// Check by extension
	trainingExtensions := []string{
		".json", ".jsonl", ".csv", ".txt", ".parquet", ".arrow",
		".npy", ".npz", ".h5", ".hdf5", ".tfrecord", ".pb",
		".tsv", ".xml", ".yaml", ".yml",
	}
	for _, ext := range trainingExtensions {
		if strings.HasSuffix(filename, ext) {
			// Additional checks to distinguish from code/config
			// Skip if it's in a config/parameter directory
			if strings.Contains(fullPath, "config") || strings.Contains(fullPath, "param") {
				continue
			}
			return true
		}
	}

	// Check by name patterns
	trainingPatterns := []string{
		"train", "training", "dataset", "data", "corpus",
		"prompt", "instruction", "fine-tune", "pretrain",
	}
	for _, pattern := range trainingPatterns {
		if strings.Contains(strings.ToLower(filename), pattern) {
			return true
		}
	}

	return false
}

// isTrainingDataDir checks if a directory name indicates training data
func isTrainingDataDir(dirname string) bool {
	trainingDirPatterns := []string{
		"train", "training", "dataset", "data", "datasets",
		"corpus", "corpora", "prompt", "prompts", "instruction",
		"fine-tune", "finetune", "pretrain", "raw", "processed",
	}
	for _, pattern := range trainingDirPatterns {
		if strings.Contains(dirname, pattern) {
			return true
		}
	}
	return false
}

// isCodeDir checks if a directory name indicates code
func isCodeDir(dirname string) bool {
	codeDirPatterns := []string{
		"src", "source", "code", "script", "scripts",
		"train", "training", "inference", "util", "utils",
		"model", "models", "module", "modules",
	}
	for _, pattern := range codeDirPatterns {
		if strings.Contains(dirname, pattern) {
			return true
		}
	}
	return false
}

// isDocFile checks if a filename is documentation
func isDocFile(filename string) bool {
	docFiles := []string{
		"readme", "changelog", "contributing",
		"history", "roadmap", "authors", "acknowledgements",
		"citation", "bibtex",
	}
	for _, doc := range docFiles {
		if strings.Contains(strings.ToLower(filename), doc) {
			return true
		}
	}
	return false
}

// isLicenseFile checks if a filename is a license file
func isLicenseFile(filename string) bool {
	licenseFiles := []string{
		"license", "licence", "copying", "copyright",
		"mit", "apache", "gpl", "bsd", "lgpl", "mpl",
		"unlicense", "eula",
	}
	for _, lic := range licenseFiles {
		if strings.Contains(strings.ToLower(filename), lic) {
			return true
		}
	}

	// Check common license file patterns
	if strings.EqualFold(filename, "license") ||
		strings.EqualFold(filename, "license.md") ||
		strings.EqualFold(filename, "license.txt") ||
		strings.EqualFold(filename, "licence") ||
		strings.EqualFold(filename, "copying") ||
		strings.EqualFold(filename, "copying.md") {
		return true
	}

	return false
}

// appendUnique appends a value to a slice if not already present
func appendUnique(slice []string, value string) []string {
	for _, v := range slice {
		if v == value {
			return slice
		}
	}
	return append(slice, value)
}

// GetMOFClassifier returns the MOF classifier
func GetMOFClassifier() MOFClassifier {
	return &MOFClassifierImpl{}
}

// ClassifyModelPath is a convenience function for classifying a model
func ClassifyModelPath(modelPath string) (string, *ClassificationResult, error) {
	classifier := GetMOFClassifier()
	classStr, err := classifier.Classify(modelPath)
	if err != nil {
		return "", nil, err
	}

	// Re-classify to get the result details
	impl := &MOFClassifierImpl{}
	result, err := impl.classifyModelPath(modelPath)
	if err != nil {
		return classStr, nil, err
	}

	return classStr, result, nil
}
