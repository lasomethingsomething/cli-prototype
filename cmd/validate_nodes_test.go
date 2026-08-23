package cmd

import (
	"testing"
)

func TestValidateNodesCommandExists(t *testing.T) {
	// This test verifies that the validate-nodes command is registered
	// and can be found in the root command
	if validateNodesCmd == nil {
		t.Fatal("validateNodesCmd is nil - command not properly initialized")
	}

	if validateNodesCmd.Use != "validate-nodes" {
		t.Errorf("validateNodesCmd.Use = %q, want %q", validateNodesCmd.Use, "validate-nodes")
	}

	if validateNodesCmd.Short != "Validate cluster nodes match artifact hardware requirements" {
		t.Errorf("validateNodesCmd.Short = %q, want %q", validateNodesCmd.Short, "Validate cluster nodes match artifact hardware requirements")
	}
}

func TestValidateNodesFlags(t *testing.T) {
	// This test verifies that the validate-nodes command has the expected flags
	if validateNodesCmd == nil {
		t.Fatal("validateNodesCmd is nil")
	}

	// Check that flags are registered
	flagNames := []string{"artifact", "registry", "namespace", "quiet", "json-output"}
	for _, flagName := range flagNames {
		if validateNodesCmd.Flags().Lookup(flagName) == nil {
			t.Errorf("flag %q not found in validate-nodes command", flagName)
		}
	}
}

// Note: Full integration testing of the validate-nodes command would require:
// 1. A mock registry with artifacts containing node requirement annotations
// 2. A running Kubernetes cluster with GPU nodes
// 3. kubectl configured and accessible
// These are best tested as integration tests rather than unit tests.
