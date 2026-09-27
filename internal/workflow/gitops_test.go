package workflow

import (
	"path/filepath"
	"testing"
)

// TestIsInsideModelPath verifies the git status porcelain line classification
func TestIsInsideModelPath(t *testing.T) {
	tests := []struct {
		name      string
		line      string
		modelPath string
		want      bool
	}{
		// Untracked directory forms
		{
			name:      "untracked dir with ?? prefix",
			line:      "?? models/iskipped/",
			modelPath: "./models/iskipped",
			want:      true,
		},
		{
			name:      "untracked dir with ?? prefix and trailing slash in modelPath",
			line:      "?? models/iskipped/",
			modelPath: "./models/iskipped/",
			want:      true,
		},
		// Modified file forms
		{
			name:      "modified file inside model dir",
			line:      " M models/iskipped/model.joblib",
			modelPath: "./models/iskipped",
			want:      true,
		},
		{
			name:      "modified file with different status",
			line:      "A models/iskipped/model.joblib",
			modelPath: "./models/iskipped",
			want:      true,
		},
		// Outside model directory
		{
			name:      "modified README outside model dir",
			line:      " M README.md",
			modelPath: "./models/iskipped",
			want:      false,
		},
		{
			name:      "modified file in different dir",
			line:      " M docs/guide.md",
			modelPath: "./models/iskipped",
			want:      false,
		},
		// Edge cases
		{
			name:      "empty modelPath",
			line:      "?? models/iskipped/",
			modelPath: "",
			want:      false,
		},
		{
			name:      "short line",
			line:      "??",
			modelPath: "./models/iskipped",
			want:      false,
		},
		{
			name:      "exact match at root",
			line:      "?? models/iskipped",
			modelPath: "./models/iskipped",
			want:      true,
		},
		{
			name:      "nested path",
			line:      "?? models/iskipped/subdir/file.txt",
			modelPath: "./models/iskipped",
			want:      true,
		},
		{
			name:      "absolute modelPath",
			line:      "?? models/iskipped/model.joblib",
			modelPath: "/full/path/models/iskipped",
			want:      false, // Different roots, won't match
		},
		{
			name:      "modelPath with ./ prefix",
			line:      "?? models/iskipped/model.joblib",
			modelPath: "./models/iskipped",
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isInsideModelPath(tt.line, tt.modelPath)
			if got != tt.want {
				t.Errorf("isInsideModelPath(%q, %q) = %v, want %v", tt.line, tt.modelPath, got, tt.want)
			}
		})
	}
}

// TestIsInsideModelPathNormalization verifies path normalization edge cases
func TestIsInsideModelPathNormalization(t *testing.T) {
	tests := []struct {
		line      string
		modelPath string
		want      bool
	}{
		// Trailing slash variants
		{line: "?? models/test/", modelPath: "./models/test", want: true},
		{line: "?? models/test/", modelPath: "./models/test/", want: true},
		{line: " M models/test/file.txt", modelPath: "models/test", want: true},
		{line: " M models/test/file.txt", modelPath: "./models/test", want: true},
		{line: " M ./models/test/file.txt", modelPath: "./models/test", want: true},
		{line: "?? ./models/test/", modelPath: "./models/test", want: true},
	}

	for _, tt := range tests {
		name := filepath.Base(tt.modelPath)
		t.Run(name, func(t *testing.T) {
			got := isInsideModelPath(tt.line, tt.modelPath)
			if got != tt.want {
				t.Errorf("isInsideModelPath(%q, %q) = %v, want %v", tt.line, tt.modelPath, got, tt.want)
			}
		})
	}
}
