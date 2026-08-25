package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/term"
)

var cfgFile string

var initConfigErr error

// nonInteractiveFlag is the --non-interactive persistent flag.
var nonInteractiveFlag bool

// nonInteractive records whether prompting is disabled for this run and why.
// It is decided once, after flags are parsed, in detectInteractivity.
var nonInteractive struct {
	active bool
	reason string
}

// detectInteractivity disables prompts when asked to (--non-interactive or
// MODEL_CLI_NO_INTERACTIVE) or when stdin is not a terminal (CI, pipes).
func detectInteractivity() {
	switch {
	case nonInteractiveFlag:
		nonInteractive.active, nonInteractive.reason = true, "--non-interactive was given"
	case envTruthy("MODEL_CLI_NO_INTERACTIVE"):
		nonInteractive.active, nonInteractive.reason = true, "MODEL_CLI_NO_INTERACTIVE is set"
	case !term.IsTerminal(int(os.Stdin.Fd())):
		nonInteractive.active, nonInteractive.reason = true, "stdin is not a terminal"
	default:
		nonInteractive.active, nonInteractive.reason = false, ""
	}
}

func envTruthy(name string) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(name))) {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}

// interactive reports whether commands may prompt the user.
func interactive() bool { return !nonInteractive.active }

// nonInteractiveReason explains why prompting is disabled, for error messages.
func nonInteractiveReason() string { return nonInteractive.reason }

// requireValues fails when any of the given flag/value pairs is empty, naming
// the flags to pass. Used after prompting so that non-interactive runs get a
// clear message instead of a downstream failure.
func requireValues(pairs ...string) error {
	var missing []string
	for i := 0; i+1 < len(pairs); i += 2 {
		if strings.TrimSpace(pairs[i+1]) == "" {
			missing = append(missing, "--"+pairs[i])
		}
	}
	if len(missing) == 0 {
		return nil
	}
	msg := "missing required value(s): " + strings.Join(missing, ", ")
	if !interactive() {
		msg += " (prompts are disabled: " + nonInteractiveReason() + ")"
	}
	return fmt.Errorf("%s", msg)
}

var rootCmd = &cobra.Command{
	Use:   "model-cli",
	Short: "A CLI to orchestrate model workflows",
	Long: `A lightweight CLI to guide users through model deployment, serving, and testing workflows.
It delegates to external tools like Argo, Flux, ORAS, or ModelPack.`,
	// Errors are reported once by main; usage is only useful for flag errors,
	// which cobra still prints before returning.
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if initConfigErr != nil {
			return initConfigErr
		}
		detectInteractivity()
		return nil
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "Config file (default: $HOME/.model-cli.yaml)")
	rootCmd.PersistentFlags().BoolVar(&nonInteractiveFlag, "non-interactive", false, "Never prompt: use flags, saved config and defaults, and fail if a required value is missing (also enabled by MODEL_CLI_NO_INTERACTIVE=1 or when stdin is not a terminal)")
}

func initConfig() {
	initConfigErr = nil
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

// warnIfSaveFails reports a failed preference save without aborting the
// command: the user's answers are still valid for this run.
func warnIfSaveFails(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not save preferences: %v\n", err)
	}
}
