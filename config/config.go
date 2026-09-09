package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

type Config struct {
	GitOps          string `mapstructure:"gitops"`
	Registry        string `mapstructure:"registry"`
	Runtime         string `mapstructure:"runtime"`
	ServingTopology string `mapstructure:"serving-topology"`
	Signer          string `mapstructure:"signer"`
}

func Load() *Config {
	var cfg Config
	viper.Unmarshal(&cfg)
	return &cfg
}

// Save persists cfg to the config file. It is a no-op when cfg matches what
// is already loaded, so commands can call it unconditionally without
// rewriting ~/.model-cli.yaml on every run.
func Save(cfg *Config) error {
	if *cfg == *Load() && viper.ConfigFileUsed() != "" {
		return nil
	}
	viper.Set("gitops", cfg.GitOps)
	viper.Set("registry", cfg.Registry)
	viper.Set("runtime", cfg.Runtime)
	viper.Set("serving-topology", cfg.ServingTopology)
	viper.Set("signer", cfg.Signer)

	if err := viper.WriteConfig(); err != nil {
		_, isNotFound := err.(viper.ConfigFileNotFoundError)
		if isNotFound || os.IsNotExist(err) {
			// First run: no config file exists yet. Write to the path viper knows about
			// (set via viper.SetConfigFile), or fall back to ~/.model-cli.yaml.
			// Tests must call viper.SetConfigFile(path) before Save() to avoid
			// writing to the real home directory.
			cfgFile := viper.ConfigFileUsed()
			if cfgFile == "" {
				home, herr := os.UserHomeDir()
				if herr != nil {
					return fmt.Errorf("could not determine home directory: %w", herr)
				}
				cfgFile = filepath.Join(home, ".model-cli.yaml")
			}
			if err := viper.WriteConfigAs(cfgFile); err != nil {
				return err
			}
			viper.SetConfigFile(cfgFile)
			return nil
		}
		return err
	}
	return nil
}
