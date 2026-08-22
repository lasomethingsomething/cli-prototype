package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

type Config struct {
	GitOps   string `mapstructure:"gitops"`
	Registry string `mapstructure:"registry"`
	Runtime  string `mapstructure:"runtime"`
	Signer   string `mapstructure:"signer"`
}

func Load() *Config {
	var cfg Config
	viper.Unmarshal(&cfg)
	return &cfg
}

func Save(cfg *Config) error {
	viper.Set("gitops", cfg.GitOps)
	viper.Set("registry", cfg.Registry)
	viper.Set("runtime", cfg.Runtime)
	viper.Set("signer", cfg.Signer)

	if err := viper.WriteConfig(); err != nil {
		_, isNotFound := err.(viper.ConfigFileNotFoundError)
		if isNotFound || os.IsNotExist(err) {
			cfgFile := viper.ConfigFileUsed()
			if cfgFile == "" {
				home, herr := os.UserHomeDir()
				if herr != nil {
					return fmt.Errorf("could not determine home directory: %w", herr)
				}
				cfgFile = filepath.Join(home, ".model-cli.yaml")
			}
			return viper.WriteConfigAs(cfgFile)
		}
		return err
	}
	return nil
}
