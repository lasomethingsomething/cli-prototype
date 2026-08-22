package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
)

func pinConfigFile(t *testing.T) string {
	t.Helper()
	viper.Reset()
	f := filepath.Join(t.TempDir(), ".model-cli.yaml")
	viper.SetConfigFile(f)
	return f
}

func TestSaveAndLoad(t *testing.T) {
	f := pinConfigFile(t)

	cfg := &Config{
		GitOps:   "argo",
		Registry: "oras",
		Runtime:  "vllm",
		Signer:   "sigstore",
	}

	if err := Save(cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	if _, err := os.Stat(f); os.IsNotExist(err) {
		t.Error("Save() did not create config file")
	}

	// Re-init viper pointing at the same file so Load() reads it back
	viper.Reset()
	viper.SetConfigFile(f)
	if err := viper.ReadInConfig(); err != nil {
		t.Fatalf("ReadInConfig() error = %v", err)
	}
	got := Load()

	if got.GitOps != cfg.GitOps {
		t.Errorf("GitOps = %q, want %q", got.GitOps, cfg.GitOps)
	}
	if got.Registry != cfg.Registry {
		t.Errorf("Registry = %q, want %q", got.Registry, cfg.Registry)
	}
	if got.Runtime != cfg.Runtime {
		t.Errorf("Runtime = %q, want %q", got.Runtime, cfg.Runtime)
	}
	if got.Signer != cfg.Signer {
		t.Errorf("Signer = %q, want %q", got.Signer, cfg.Signer)
	}
}

func TestLoadNoFile(t *testing.T) {
	viper.Reset()
	viper.AddConfigPath(t.TempDir())
	viper.SetConfigType("yaml")
	viper.SetConfigName(".model-cli")

	got := Load()

	if got.GitOps != "" || got.Registry != "" || got.Runtime != "" || got.Signer != "" {
		t.Errorf("Load() with no file = %+v, want zero-value Config", got)
	}
}
