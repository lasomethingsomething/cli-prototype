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
	// ModelFormatName is the model format name (e.g., "sklearn", "huggingface")
	ModelFormatName string
	// ModelFormatVersion is the model format version (e.g., "1")
	ModelFormatVersion string
	// ContainerArgs are additional arguments for the serving container
	ContainerArgs []string
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
        name: {{.ModelFormatName}}
        version: "{{.ModelFormatVersion}}"
      runtime: {{.Runtime}}
      storageUri: "{{.StorageUri}}"
{{if .ContainerArgs}}
      container:
        args:{{range .ContainerArgs}}
        - {{.}}{{end}}
{{end}}
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
// If runtime is provided, it overrides the derived runtime (for explicit serving topology selection)
func CreateInferenceServiceConfig(
	modelName string,
	modelPath string,
	repoURL string,
	branch string,
	runtime string, // Optional: if provided, overrides the derived runtime
) (*InferenceServiceConfig, error) {
	// Normalize the repo URL
	normalizedRepo := normalizeGitURL(repoURL)

	// If modelPath is empty, we need to derive it from the repo structure
	// For now, we'll use the provided modelPath or derive from modelName
	if modelPath == "" {
		modelPath = fmt.Sprintf("models/%s", modelName)
	}

	// Derive runtime from model path if not explicitly provided
	var runtimeInfo *RuntimeInfo
	var err error
	if runtime == "" {
		// Function signature: DeriveRuntimeFromModelPath(modelPath, modelName, repoURL, currentBranch)
		runtimeInfo, err = DeriveRuntimeFromModelPath(modelPath, modelName, repoURL, branch)
		if err != nil {
			return nil, fmt.Errorf("failed to derive runtime: %w", err)
		}
		// Use the derived runtime
		if runtime == "" {
			runtime = runtimeInfo.Runtime
		}
	} else {
		// Use the explicitly provided runtime
		// Still need runtimeInfo for model format detection
		runtimeInfo, err = DeriveRuntimeFromModelPath(modelPath, modelName, repoURL, branch)
		if err != nil {
			// If we can't derive, create a minimal runtimeInfo
			runtimeInfo = &RuntimeInfo{
				ModelFormat: DetectModelFormatFromPath(modelPath),
				Runtime:     runtime,
			}
		}
	}

	// Build storage URI
	var storageUri string
	if runtime == "kserve-huggingfaceserver" {
		// For Hugging Face runtime, use hf:// prefix with model name
		// This allows models to be pulled from Hugging Face Hub
		storageUri = fmt.Sprintf("hf://%s", modelName)
	} else {
		rawBaseURL := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s", normalizedRepo, branch)

		// Get the model file from the model directory
		modelFile := FindModelFile(modelPath)
		if modelFile == "" {
			return nil, fmt.Errorf("no model file found in %s", modelPath)
		}

		// Build storage URI - include the model directory path
		// Strip leading "./" from modelPath for the URL
		modelDir := strings.TrimPrefix(modelPath, "./")
		storageUri = fmt.Sprintf("%s/%s/%s", rawBaseURL, modelDir, modelFile)
	}

	// Determine namespace from config or use default
	// All InferenceServices use the 'models' namespace for consistency
	namespace := "models"

	// Map model format to KServe model format name and version
	modelFormatName := strings.ToLower(string(runtimeInfo.ModelFormat))
	modelFormatVersion := "1" // Default version
	
	// Validate and set model format for KServe
	validFormats := map[string]string{
		"sklearn":     "sklearn",
		"pytorch":     "pytorch",
		"tensorflow":  "tensorflow",
		"onnx":        "onnx",
		"huggingface": "huggingface",
	}
	if validFormat, ok := validFormats[modelFormatName]; ok {
		modelFormatName = validFormat
	} else {
		modelFormatName = "sklearn" // default to sklearn if unknown
	}
	
	// For huggingface runtime, explicitly set format to huggingface
	if runtime == "kserve-huggingfaceserver" {
		modelFormatName = "huggingface"
		modelFormatVersion = "1"
	}

	// Set container args based on runtime
	var containerArgs []string
	switch runtime {
	case "kserve-sklearnserver", "kserve-mlserver":
		containerArgs = []string{
			fmt.Sprintf("--model_name=%s", modelName),
			fmt.Sprintf("--model_dir=/mnt/models"),
			"--http_port=8080",
		}
	case "kserve-huggingfaceserver":
		containerArgs = []string{
			fmt.Sprintf("--model_id=%s", modelName),
			"--backend=vllm",
		}
	default:
		// For other runtimes (pytorch, tensorflow, etc.), use standard args
		containerArgs = []string{
			fmt.Sprintf("--model_name=%s", modelName),
			fmt.Sprintf("--model_dir=/mnt/models"),
			"--http_port=8080",
		}
	}

	return &InferenceServiceConfig{
		ModelName:          modelName,
		Namespace:          namespace,
		ModelPath:          modelPath,
		RepoURL:            repoURL,
		Branch:             branch,
		Runtime:            runtime,
		StorageUri:         storageUri,
		ModelFormatName:    modelFormatName,
		ModelFormatVersion: modelFormatVersion,
		ContainerArgs:      containerArgs,
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
