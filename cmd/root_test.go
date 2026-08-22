package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInitConfigResetsBetweenCalls(t *testing.T) {
	original := cfgFile
	defer func() { cfgFile = original; initConfigErr = nil }()

	cfgFile = ""
	initConfig()
	// initConfigErr may be nil (no config file is fine) — just verify it's not the artificial value
	initConfigErr = os.ErrInvalid

	cfgFile = ""
	initConfig()
	if initConfigErr == os.ErrInvalid {
		t.Error("initConfig() did not reset initConfigErr at start of call")
	}
}

func TestInitConfigBadConfigFile(t *testing.T) {
	dir := t.TempDir()
	badFile := filepath.Join(dir, "bad.yaml")
	if err := os.WriteFile(badFile, []byte(":\ninvalid: [yaml"), 0644); err != nil {
		t.Fatal(err)
	}

	original := cfgFile
	defer func() { cfgFile = original; initConfigErr = nil }()

	cfgFile = badFile
	initConfig()

	if initConfigErr == nil {
		t.Error("initConfig() with invalid YAML should set initConfigErr, got nil")
	}
}
