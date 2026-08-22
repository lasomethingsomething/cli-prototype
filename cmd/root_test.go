package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
)

func TestInitConfigNoFile(t *testing.T) {
	viper.Reset()
	origCfgFile := cfgFile
	cfgFile = ""
	defer func() { cfgFile = origCfgFile }()

	initConfig()
	if initConfigErr != nil {
		t.Errorf("initConfig() with no file set initConfigErr = %v, want nil (missing file is not an error)", initConfigErr)
	}
}

func TestInitConfigBadFile(t *testing.T) {
	dir := t.TempDir()
	badFile := filepath.Join(dir, "bad.yaml")
	os.WriteFile(badFile, []byte("key: [invalid yaml"), 0644)

	origCfgFile := cfgFile
	cfgFile = badFile
	defer func() { cfgFile = origCfgFile }()

	viper.Reset()
	initConfig()
	if initConfigErr == nil {
		t.Error("initConfig() with invalid YAML file should set initConfigErr, got nil")
	}
}

func TestInitConfigErrResetBetweenCalls(t *testing.T) {
	// Verify that a second call to initConfig() clears a prior error.
	dir := t.TempDir()
	badFile := filepath.Join(dir, "bad.yaml")
	os.WriteFile(badFile, []byte("key: [invalid yaml"), 0644)

	origCfgFile := cfgFile
	cfgFile = badFile
	defer func() { cfgFile = origCfgFile }()

	viper.Reset()
	initConfig()
	if initConfigErr == nil {
		t.Fatal("expected error from bad file, got nil")
	}

	// Now reset to no file — error should clear.
	cfgFile = ""
	viper.Reset()
	initConfig()
	if initConfigErr != nil {
		t.Errorf("initConfig() after reset: initConfigErr = %v, want nil", initConfigErr)
	}
}
