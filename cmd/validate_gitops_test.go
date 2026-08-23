package cmd

import (
	"testing"
)

func TestValidateGitOpsCommandExists(t *testing.T) {
	// This test verifies that the validate-gitops command is registered
	// and can be found in the root command
	if validateGitOpsCmd == nil {
		t.Fatal("validateGitOpsCmd is nil - command not properly initialized")
	}

	if validateGitOpsCmd.Use != "validate-gitops" {
		t.Errorf("validateGitOpsCmd.Use = %q, want %q", validateGitOpsCmd.Use, "validate-gitops")
	}

	if validateGitOpsCmd.Short != "Validate artifact annotations for GitOps deployment" {
		t.Errorf("validateGitOpsCmd.Short = %q, want %q", validateGitOpsCmd.Short, "Validate artifact annotations for GitOps deployment")
	}
}

func TestValidateGitOpsFlags(t *testing.T) {
	// This test verifies that the validate-gitops command has the expected flags
	if validateGitOpsCmd == nil {
		t.Fatal("validateGitOpsCmd is nil")
	}

	// Check that flags are registered
	flagNames := []string{"artifact", "registry", "quiet", "json-output"}
	for _, flagName := range flagNames {
		if validateGitOpsCmd.Flags().Lookup(flagName) == nil {
			t.Errorf("flag %q not found in validate-gitops command", flagName)
		}
	}
}

// Note: Full integration testing of the validate-gitops command would require:
// 1. A mock registry with artifacts
// 2. A running registry server
// 3. ORAS or ModelPack CLI installed
// These are covered by the workflow-level tests in internal/workflow/registry_test.go
// and are best tested as integration tests rather than unit tests.
