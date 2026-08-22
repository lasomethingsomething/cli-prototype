package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

var initConfigErr error

var rootCmd = &cobra.Command{
	Use:   "model-cli",
	Short: "A CLI to orchestrate model workflows",
	Long: `A lightweight CLI to guide users through model deployment, serving, and testing workflows.
It delegates to external tools like Argo, Flux, ORAS, or ModelPack.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return initConfigErr
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "Config file (default: $HOME/.model-cli.yaml)")
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			initConfigErr = err
			return
		}
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".model-cli")
	}
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			initConfigErr = err
		}
	}
}
