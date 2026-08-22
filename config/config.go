package config

import (
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
	return viper.WriteConfig()
}
