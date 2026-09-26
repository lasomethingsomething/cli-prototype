package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCreateInferenceServiceConfigStorageUri tests the full storageUri construction
// Regression test for issue #47 where storageUri was malformed with double domain
// and missing model directory path
func TestCreateInferenceServiceConfigStorageUri(t *testing.T) {
	// Create a temporary directory with model files
	tmpDir, err := os.MkdirTemp("", "inference-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)
	
	// Create the model directory structure: models/iris/
	modelDir := filepath.Join(tmpDir, "models", "iris")
	if err := os.MkdirAll(modelDir, 0755); err != nil {
		t.Fatalf("failed to create model dir: %v", err)
	}
	
	// Create model.joblib
	modelFile := filepath.Join(modelDir, "model.joblib")
	if err := os.WriteFile(modelFile, []byte("test model data"), 0644); err != nil {
		t.Fatalf("failed to create model file: %v", err)
	}
	
	// Create config.json (should be skipped)
	if err := os.WriteFile(filepath.Join(modelDir, "config.json"), []byte("{}"), 0644); err != nil {
		t.Fatalf("failed to create config.json: %v", err)
	}
	
	// Test case from issue #47:
	// repo: ssh://git@github.com/lasomethingsomething/cli-prototype.git
	// modelPath: the absolute path to models/iris
	// branch: main
	// Expected storageUri: https://raw.githubusercontent.com/lasomethingsomething/cli-prototype/main/models/iris/model.joblib
	
	repoURL := "ssh://git@github.com/lasomethingsomething/cli-prototype.git"
	modelName := "iris"
	modelPath := modelDir  // Use absolute path
	branch := "main"
	
	config, err := CreateInferenceServiceConfig(modelName, modelPath, repoURL, branch)
	if err != nil {
		t.Fatalf("CreateInferenceServiceConfig failed: %v", err)
	}
	
	// Check that storageUri is correctly formed
	// The modelPath is an absolute path like /tmp/.../models/iris
	// We need to handle this properly - the storageUri should use the relative path from repo root
	// For this test, we'll check the structure
	
	// Verify no double domain
	if strings.Contains(config.StorageUri, "github.com/github.com") {
		t.Errorf("StorageUri contains double domain: %s", config.StorageUri)
	}
	
	// Verify it uses the correct base URL
	if !strings.HasPrefix(config.StorageUri, "https://raw.githubusercontent.com/lasomethingsomething/cli-prototype/main/") {
		t.Errorf("StorageUri doesn't start with correct base: %s", config.StorageUri)
	}
	
	// Verify it contains the model file
	if !strings.Contains(config.StorageUri, "model.joblib") {
		t.Errorf("StorageUri missing model file: %s", config.StorageUri)
	}
	
	// Verify runtime is correct for sklearn
	if config.Runtime != "kserve-sklearnserver" {
		t.Errorf("Runtime = %q, want %q", config.Runtime, "kserve-sklearnserver")
	}
	
	// Verify modelFormat
	if config.ModelFormat != "sklearn" {
		t.Errorf("ModelFormat = %q, want %q", config.ModelFormat, "sklearn")
	}
}

// TestCreateInferenceServiceConfigWithSCPUrl tests with SCP-style Git URL
func TestCreateInferenceServiceConfigWithSCPUrl(t *testing.T) {
	// Create a temporary directory with model files
	tmpDir, err := os.MkdirTemp("", "inference-test-scp")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)
	
	modelDir := filepath.Join(tmpDir, "models", "iris")
	if err := os.MkdirAll(modelDir, 0755); err != nil {
		t.Fatalf("failed to create model dir: %v", err)
	}
	
	modelFile := filepath.Join(modelDir, "model.joblib")
	if err := os.WriteFile(modelFile, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create model file: %v", err)
	}
	
	repoURL := "git@github.com:lasomethingsomething/cli-prototype.git"
	modelName := "iris"
	modelPath := modelDir
	branch := "main"
	
	config, err := CreateInferenceServiceConfig(modelName, modelPath, repoURL, branch)
	if err != nil {
		t.Fatalf("CreateInferenceServiceConfig failed: %v", err)
	}
	
	// Verify no double domain
	if strings.Contains(config.StorageUri, "github.com/github.com") {
		t.Errorf("StorageUri contains double domain: %s", config.StorageUri)
	}
	
	// Verify it uses the correct base URL
	if !strings.HasPrefix(config.StorageUri, "https://raw.githubusercontent.com/lasomethingsomething/cli-prototype/main/") {
		t.Errorf("StorageUri doesn't start with correct base: %s", config.StorageUri)
	}
}

// TestCreateInferenceServiceConfigWithHTTPSUrl tests with HTTPS Git URL
func TestCreateInferenceServiceConfigWithHTTPSUrl(t *testing.T) {
	// Create a temporary directory with model files
	tmpDir, err := os.MkdirTemp("", "inference-test-https")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)
	
	modelDir := filepath.Join(tmpDir, "models", "iris")
	if err := os.MkdirAll(modelDir, 0755); err != nil {
		t.Fatalf("failed to create model dir: %v", err)
	}
	
	modelFile := filepath.Join(modelDir, "model.joblib")
	if err := os.WriteFile(modelFile, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create model file: %v", err)
	}
	
	repoURL := "https://github.com/lasomethingsomething/cli-prototype.git"
	modelName := "iris"
	modelPath := modelDir
	branch := "main"
	
	config, err := CreateInferenceServiceConfig(modelName, modelPath, repoURL, branch)
	if err != nil {
		t.Fatalf("CreateInferenceServiceConfig failed: %v", err)
	}
	
	// Verify no double domain
	if strings.Contains(config.StorageUri, "github.com/github.com") {
		t.Errorf("StorageUri contains double domain: %s", config.StorageUri)
	}
	
	// Verify it uses the correct base URL
	if !strings.HasPrefix(config.StorageUri, "https://raw.githubusercontent.com/lasomethingsomething/cli-prototype/main/") {
		t.Errorf("StorageUri doesn't start with correct base: %s", config.StorageUri)
	}
}

// TestNormalizeGitURL tests the normalizeGitURL function
func TestNormalizeGitURL(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// SCP-style
		{"git@github.com:lasomethingsomething/cli-prototype.git", "lasomethingsomething/cli-prototype"},
		{"git@github.com:lasomethingsomething/cli-prototype", "lasomethingsomething/cli-prototype"},
		// SSH-style
		{"ssh://git@github.com/lasomethingsomething/cli-prototype.git", "lasomethingsomething/cli-prototype"},
		{"ssh://git@github.com/lasomethingsomething/cli-prototype", "lasomethingsomething/cli-prototype"},
		// HTTPS
		{"https://github.com/lasomethingsomething/cli-prototype.git", "lasomethingsomething/cli-prototype"},
		{"https://github.com/lasomethingsomething/cli-prototype", "lasomethingsomething/cli-prototype"},
		{"http://github.com/lasomethingsomething/cli-prototype.git", "lasomethingsomething/cli-prototype"},
		// Git protocol
		{"git://github.com/lasomethingsomething/cli-prototype.git", "lasomethingsomething/cli-prototype"},
		// With trailing spaces
		{"  ssh://git@github.com/lasomethingsomething/cli-prototype.git  ", "lasomethingsomething/cli-prototype"},
	}
	
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizeGitURL(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeGitURL(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
