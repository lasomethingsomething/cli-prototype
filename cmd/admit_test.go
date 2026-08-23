package cmd

import (
	"testing"
)

func TestAdmitCommandExists(t *testing.T) {
	// This test verifies that the admit command is registered
	// and can be found in the root command
	if admitCmd == nil {
		t.Fatal("admitCmd is nil - command not properly initialized")
	}

	if admitCmd.Use != "admit" {
		t.Errorf("admitCmd.Use = %q, want %q", admitCmd.Use, "admit")
	}

	if admitCmd.Short != "Evaluate artifact for GitOps admission" {
		t.Errorf("admitCmd.Short = %q, want %q", admitCmd.Short, "Evaluate artifact for GitOps admission")
	}
}

func TestAdmitFlags(t *testing.T) {
	// This test verifies that the admit command has the expected flags
	if admitCmd == nil {
		t.Fatal("admitCmd is nil")
	}

	// Check that flags are registered
	flagNames := []string{"artifact", "registry", "env", "strict", "json-output", "region"}
	for _, flagName := range flagNames {
		if admitCmd.Flags().Lookup(flagName) == nil {
			t.Errorf("flag %q not found in admit command", flagName)
		}
	}
}
