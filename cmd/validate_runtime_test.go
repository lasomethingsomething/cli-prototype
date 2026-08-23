package cmd

import (
	"testing"
)

func TestValidateRuntimeCommandExists(t *testing.T) {
	// This test verifies that the validate-runtime command is registered
	// and can be found in the root command
	if validateRuntimeCmd == nil {
		t.Fatal("validateRuntimeCmd is nil - command not properly initialized")
	}

	if validateRuntimeCmd.Use != "validate-runtime" {
		t.Errorf("validateRuntimeCmd.Use = %q, want %q", validateRuntimeCmd.Use, "validate-runtime")
	}

	if validateRuntimeCmd.Short != "Validate runtime availability for artifact deployment" {
		t.Errorf("validateRuntimeCmd.Short = %q, want %q", validateRuntimeCmd.Short, "Validate runtime availability for artifact deployment")
	}
}

func TestValidateRuntimeFlags(t *testing.T) {
	// This test verifies that the validate-runtime command has the expected flags
	if validateRuntimeCmd == nil {
		t.Fatal("validateRuntimeCmd is nil")
	}

	// Check that flags are registered
	flagNames := []string{"artifact", "registry", "namespace", "quiet", "json-output"}
	for _, flagName := range flagNames {
		if validateRuntimeCmd.Flags().Lookup(flagName) == nil {
			t.Errorf("flag %q not found in validate-runtime command", flagName)
		}
	}
}

// Note: Full integration testing of the validate-runtime command would require:
// 1. A mock registry with artifacts containing runtime requirement annotations
// 2. A running Kubernetes cluster with runtime operators (KServe, vLLM) installed
// 3. kubectl configured and accessible
// 4. Network access to Reference Skill DLC endpoints
// These are best tested as integration tests rather than unit tests.
