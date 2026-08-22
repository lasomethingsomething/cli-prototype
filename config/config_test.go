package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
)

func resetViper() {
	viper.Reset()
}

func TestSaveFirstRun(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ".model-cli.yaml")
	resetViper()
	// Override HOME so the first-run fallback writes to the temp dir, not ~/.model-cli.yaml.
	t.Setenv("HOME", dir)
	// Use SetConfigName/AddConfigPath (not SetConfigFile) so viper.ConfigFileUsed()
	// returns "" on first Save(), exercising the true first-run fallback path.
	viper.SetConfigName(".model-cli")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(dir)

	cfg := &Config{
		GitOps:   "argo",
		Registry: "oras",
		Signer:   "sigstore",
		Runtime:  "vllm",
	}

	if err := Save(cfg); err != nil {
		t.Fatalf("Save() on first run error = %v", err)
	}

	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		t.Errorf("config file not created at %s", cfgPath)
	}

	// Second Save() should succeed via WriteConfig() now that viper knows the file path.
	cfg.GitOps = "flux"
	if err := Save(cfg); err != nil {
		t.Fatalf("Save() on second run error = %v", err)
	}
}

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ".model-cli.yaml")
	resetViper()
	viper.SetConfigFile(cfgPath)

	cfg := &Config{
		GitOps:   "flux",
		Registry: "modelpack",
		Signer:   "notary",
		Runtime:  "kserve",
	}

	if err := Save(cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	resetViper()
	viper.SetConfigFile(cfgPath)
	if err := viper.ReadInConfig(); err != nil {
		t.Fatalf("ReadInConfig() error = %v", err)
	}

	loaded := Load()
	if loaded.GitOps != cfg.GitOps {
		t.Errorf("GitOps = %q, want %q", loaded.GitOps, cfg.GitOps)
	}
	if loaded.Registry != cfg.Registry {
		t.Errorf("Registry = %q, want %q", loaded.Registry, cfg.Registry)
	}
	if loaded.Signer != cfg.Signer {
		t.Errorf("Signer = %q, want %q", loaded.Signer, cfg.Signer)
	}
	if loaded.Runtime != cfg.Runtime {
		t.Errorf("Runtime = %q, want %q", loaded.Runtime, cfg.Runtime)
	}
}
