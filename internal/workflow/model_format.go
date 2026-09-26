package workflow

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ModelFormat represents the format/type of a model
type ModelFormat string

const (
	ModelFormatSklearn    ModelFormat = "sklearn"
	ModelFormatPyTorch    ModelFormat = "pytorch"
	ModelFormatTensorFlow ModelFormat = "tensorflow"
	ModelFormatONNX       ModelFormat = "onnx"
	ModelFormatHuggingFace ModelFormat = "huggingface"
	ModelFormatUnknown    ModelFormat = "unknown"
)

// ModelFileExtensions is the canonical list of model file extensions
// Used consistently across all model file detection logic
// Excludes .json to avoid matching config.json, mof.json, etc.
var ModelFileExtensions = []string{
	".joblib", ".pkl", ".pickle",
	".pt", ".pth",
	".pb", ".h5", ".hdf5",
	".onnx",
	".bin",
	".safetensors",
	".tflite",
}

// RuntimeInfo contains runtime information for deploying a model
type RuntimeInfo struct {
	// ModelFormat is the detected format of the model
	ModelFormat ModelFormat
	// Runtime is the serving runtime to use (e.g., "sklearnserver", "kserve-sklearnserver")
	Runtime string
	// StorageUri points to the actual model file
	StorageUri string
}

// DetectModelFormatFromPath detects the model format based on file extensions
func DetectModelFormatFromPath(modelPath string) ModelFormat {
	// Check for sklearn models
	if hasFileWithExtensions(modelPath, ".joblib", ".pkl", ".pickle") {
		return ModelFormatSklearn
	}

	// Check for PyTorch models
	if hasFileWithExtensions(modelPath, ".pt", ".pth") {
		return ModelFormatPyTorch
	}

	// Check for TensorFlow models
	if hasFileWithExtensions(modelPath, ".pb", ".h5", ".hdf5", ".tflite") {
		return ModelFormatTensorFlow
	}

	// Check for ONNX models
	if hasFileWithExtensions(modelPath, ".onnx") {
		return ModelFormatONNX
	}

	// Check for HuggingFace directory structure
	if hasHuggingFaceStructure(modelPath) {
		return ModelFormatHuggingFace
	}

	return ModelFormatUnknown
}

// hasFileWithExtensions checks if the directory contains files with any of the given extensions
func hasFileWithExtensions(dir string, extensions ...string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		lowerName := strings.ToLower(entry.Name())
		for _, ext := range extensions {
			if strings.HasSuffix(lowerName, ext) {
				return true
			}
		}
	}
	return false
}

// hasHuggingFaceStructure checks for common HuggingFace model directory structure
func hasHuggingFaceStructure(dir string) bool {
	// HuggingFace models typically have: config.json, pytorch_model.bin, tf_model.h5, etc.
	hasConfig := false
	hasModelFile := false

	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		lowerName := strings.ToLower(entry.Name())
		if lowerName == "config.json" {
			hasConfig = true
		}
		if strings.HasSuffix(lowerName, ".bin") || strings.HasSuffix(lowerName, ".json") ||
			strings.HasSuffix(lowerName, ".h5") || strings.HasSuffix(lowerName, ".pt") ||
			strings.HasSuffix(lowerName, ".pth") || strings.HasSuffix(lowerName, ".safetensors") {
			hasModelFile = true
		}
	}

	return hasConfig && hasModelFile
}

// DeriveRuntimeFromModelPath detects the model format and returns appropriate runtime info
// This ensures sklearn models use sklearnserver, not vllm
func DeriveRuntimeFromModelPath(modelPath string, modelName string, repoURL string, currentBranch string) (*RuntimeInfo, error) {
	modelFormat := DetectModelFormatFromPath(modelPath)

	// Get the actual model file path
	modelFile := FindModelFile(modelPath)
	if modelFile == "" {
		return nil, fmt.Errorf("no model file found in %s", modelPath)
	}

	// Build storage URI - use raw GitHub URL in branch form
	storageUri := ""
	if repoURL != "" {
		// Normalize the repo URL
		normalizedRepo := normalizeGitURL(repoURL)
		// Get branch - if not provided, use "main"
		branch := currentBranch
		if branch == "" {
			branch = "main"
		}
		storageUri = fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s", normalizedRepo, branch, modelFile)
	}

	// Derive runtime based on model format
	runtime := ""
	switch modelFormat {
	case ModelFormatSklearn:
		// sklearn models should use sklearnserver or kserve-sklearnserver, NOT vllm
		runtime = "kserve-sklearnserver"
	case ModelFormatPyTorch:
		runtime = "pytorch"
	case ModelFormatTensorFlow:
		runtime = "tensorflow"
	case ModelFormatONNX:
		runtime = "onnx"
	case ModelFormatHuggingFace:
		// For HuggingFace, we could use vllm if it's a text generation model
		// but for now use the generic huggingface runtime
		runtime = "huggingface"
	default:
		// Default to vllm only if we can't determine the format
		// But for sklearn, we should NEVER default to vllm
		if strings.Contains(strings.ToLower(modelName), "sklearn") ||
			strings.Contains(strings.ToLower(modelPath), "sklearn") {
			runtime = "kserve-sklearnserver"
		} else {
			runtime = "vllm"
		}
	}

	return &RuntimeInfo{
		ModelFormat: modelFormat,
		Runtime:    runtime,
		StorageUri:  storageUri,
	}, nil
}

// FindModelFile finds the primary model file in a directory
func FindModelFile(dir string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		// Don't silently discard error - log it for debugging
		// In production, consider returning an error from the caller
		return ""
	}

	// First, check for common known model files using the canonical extension list
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		lowerName := strings.ToLower(entry.Name())
		// Skip files that model-cli itself writes (manifest, SBOM, MOF, etc.)
		if lowerName == "manifest.json" || lowerName == "config.json" ||
			lowerName == "attestation.json" || lowerName == "mof.json" ||
			strings.HasPrefix(lowerName, "sbom.") {
			continue
		}
		for _, ext := range ModelFileExtensions {
			if strings.HasSuffix(lowerName, ext) {
				// Return the entry name directly - it's already relative to dir
				return filepath.ToSlash(entry.Name())
			}
		}
	}

	return ""
}


