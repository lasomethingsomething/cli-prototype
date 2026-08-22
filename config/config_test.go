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
	// Point viper at the explicit path so the first-run fallback writes here
	viper.SetConfigFile(cfgPath)

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
