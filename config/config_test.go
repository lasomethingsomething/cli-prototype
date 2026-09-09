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
		GitOps:          "argo",
		Registry:        "oras",
		Signer:          "sigstore",
		Runtime:         "vllm",
		ServingTopology: "vllm",
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
		GitOps:          "flux",
		Registry:        "modelpack",
		Signer:          "notary",
		Runtime:         "kserve",
		ServingTopology: "kserve-vllm",
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
	if loaded.ServingTopology != cfg.ServingTopology {
		t.Errorf("ServingTopology = %q, want %q", loaded.ServingTopology, cfg.ServingTopology)
	}
}

func TestSaveSkipsWriteWhenUnchanged(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ".model-cli.yaml")
	resetViper()
	viper.SetConfigFile(cfgPath)

	cfg := &Config{GitOps: "argo", Registry: "oras"}
	if err := Save(cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	before, err := os.Stat(cfgPath)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}

	// Make the file read-only: a second Save with identical content must not
	// try to write and therefore must not fail.
	if err := os.Chmod(cfgPath, 0444); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	t.Cleanup(func() { os.Chmod(cfgPath, 0644) })

	if err := Save(&Config{GitOps: "argo", Registry: "oras"}); err != nil {
		t.Errorf("Save() with unchanged config should be a no-op, got error: %v", err)
	}
	after, _ := os.Stat(cfgPath)
	if !after.ModTime().Equal(before.ModTime()) {
		t.Errorf("config file was rewritten although nothing changed")
	}
}
