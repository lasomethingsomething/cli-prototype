package cmd

import (
	"testing"
)

func TestHardenCmd(t *testing.T) {
	// This is a basic test to ensure the harden command can be initialized
	// Actual execution requires model files which are tested in workflow tests

	// The command should be registered
	if hardenCmd == nil {
		t.Fatal("hardenCmd is nil")
	}

	if hardenCmd.Use != "harden" {
		t.Errorf("Expected Use 'harden', got '%s'", hardenCmd.Use)
	}

	if hardenCmd.Short != "Apply local hardening and compliance to model artifact" {
		t.Errorf("Unexpected Short description: %s", hardenCmd.Short)
	}

	// Verify license flag exists and has correct default
	licenseFlag := hardenCmd.Flags().Lookup("license")
	if licenseFlag == nil {
		t.Fatal("license flag not found")
	}

	if licenseFlag.DefValue != "CC-BY-4.0" {
		t.Errorf("Expected default license 'CC-BY-4.0', got '%s'", licenseFlag.DefValue)
	}

	// Verify required flags
	requiredFlags := []string{"model", "model-path", "artifact"}
	for _, flagName := range requiredFlags {
		flag := hardenCmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("Required flag '%s' not found", flagName)
		}
	}

	// SBOM and MOF are harden's job (issue #87): the tool choice and the
	// declared MOF class live here, not on package.
	for flagName, wantDefault := range map[string]string{"sbom-tool": "syft", "mof-class": "", "mof-components": ""} {
		flag := hardenCmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("flag '%s' not found on harden", flagName)
			continue
		}
		if flag.DefValue != wantDefault {
			t.Errorf("flag '%s' default = %q, want %q", flagName, flag.DefValue, wantDefault)
		}
	}
	for _, flagName := range []string{"mof-class", "mof-components"} {
		if packageCmd.Flags().Lookup(flagName) != nil {
			t.Errorf("flag '%s' is still on package; MOF classification belongs to harden", flagName)
		}
	}
}
