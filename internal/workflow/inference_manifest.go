package workflow

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
)

// InferenceServiceConfig holds the configuration for generating an InferenceService manifest
type InferenceServiceConfig struct {
	// ModelName is the name of the model
	ModelName string
	// Namespace is the Kubernetes namespace
	Namespace string
	// ModelPath is the local path to the model
	ModelPath string
	// RepoURL is the git repository URL
	RepoURL string
	// Branch is the git branch
	Branch string
	// Runtime is the serving runtime (e.g., "kserve-sklearnserver")
	Runtime string
	// StorageUri is the URL to the model file
	StorageUri string
	// ModelFormat is the model format (e.g., "sklearn")
	ModelFormat string
}

// InferenceServiceTemplate is the template for generating an InferenceService manifest
const InferenceServiceTemplate = `apiVersion: serving.kserve.io/v1beta1
kind: InferenceService
metadata:
  name: {{.ModelName}}
  namespace: {{.Namespace}}
  annotations:
    serving.kserve.io/deploymentMode: RawDeployment
spec:
  predictor:
    model:
      modelFormat:
        name: {{.ModelFormat}}
      runtime: {{.Runtime}}
      storageUri: "{{.StorageUri}}"
      resources:
        requests:
          cpu: 100m
          memory: 256Mi
        limits:
          cpu: 500m
          memory: 512Mi
`

// GenerateInferenceServiceManifest creates an InferenceService YAML manifest from the config
func GenerateInferenceServiceManifest(config *InferenceServiceConfig) (string, error) {
	tmpl, err := template.New("inference").Parse(InferenceServiceTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var builder strings.Builder
	if err := tmpl.Execute(&builder, config); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return builder.String(), nil
}

// WriteInferenceServiceManifest generates and writes an InferenceService manifest to a file
func WriteInferenceServiceManifest(config *InferenceServiceConfig, outputPath string) error {
	manifest, err := GenerateInferenceServiceManifest(config)
	if err != nil {
		return err
	}

	// Ensure output directory exists
	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	if err := os.WriteFile(outputPath, []byte(manifest), 0644); err != nil {
		return fmt.Errorf("failed to write manifest: %w", err)
	}

	return nil
}

// CreateInferenceServiceConfig creates a config for generating an InferenceService manifest
// It derives the runtime from the model format and constructs the storage URI
func CreateInferenceServiceConfig(
	modelName string,
	modelPath string,
	repoURL string,
	branch string,
) (*InferenceServiceConfig, error) {
	// Normalize the repo URL
	normalizedRepo := normalizeGitURL(repoURL)
	
	// If modelPath is empty, we need to derive it from the repo structure
	// For now, we'll use the provided modelPath or derive from modelName
	if modelPath == "" {
		modelPath = fmt.Sprintf("models/%s", modelName)
	}
	
	// Derive runtime from model path
	// Function signature: DeriveRuntimeFromModelPath(modelPath, modelName, repoURL, currentBranch)
	runtimeInfo, err := DeriveRuntimeFromModelPath(modelPath, modelName, repoURL, branch)
	if err != nil {
		return nil, fmt.Errorf("failed to derive runtime: %w", err)
	}

	rawBaseURL := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s", normalizedRepo, branch)

	// Get the model file from the model directory
	modelFile := FindModelFile(modelPath)
	if modelFile == "" {
		return nil, fmt.Errorf("no model file found in %s", modelPath)
	}

	// Build storage URI - include the model directory path
	// Strip leading "./" from modelPath for the URL
	modelDir := strings.TrimPrefix(modelPath, "./")
	storageUri := fmt.Sprintf("%s/%s/%s", rawBaseURL, modelDir, modelFile)

	// Determine namespace from config or use default
	namespace := "test-model"

	// Map model format to KServe model format name
	modelFormat := strings.ToLower(string(runtimeInfo.ModelFormat))
	// Validate and set model format for KServe
	validFormats := map[string]string{
		"sklearn":    "sklearn",
		"pytorch":    "pytorch",
		"tensorflow": "tensorflow",
		"onnx":       "onnx",
		"huggingface": "huggingface",
	}
	if validFormat, ok := validFormats[modelFormat]; ok {
		modelFormat = validFormat
	} else {
		modelFormat = "sklearn" // default to sklearn if unknown
	}

	return &InferenceServiceConfig{
		ModelName:   modelName,
		Namespace:  namespace,
		ModelPath:  modelPath,
		RepoURL:    repoURL,
		Branch:    branch,
		Runtime:   runtimeInfo.Runtime,
		StorageUri: storageUri,
		ModelFormat: modelFormat,
	}, nil
}

// GetCurrentGitBranch returns the current git branch
func GetCurrentGitBranch() (string, error) {
	// Try to get the current branch
	branch, err := runCommandOutput("git", "branch", "--show-current")
	if err == nil {
		return strings.TrimSpace(string(branch)), nil
	}

	// Fallback: try symbolic-ref
	branch, err = runCommandOutput("git", "symbolic-ref", "--short", "HEAD")
	if err == nil {
		return strings.TrimSpace(string(branch)), nil
	}

	return "main", nil
}

// runCommandOutput is a helper to run a command and get its output
func runCommandOutput(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	return cmd.Output()
}
