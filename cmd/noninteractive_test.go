package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

func forceNonInteractive(t *testing.T) {
	t.Helper()
	prev := nonInteractive
	nonInteractive.active, nonInteractive.reason = true, "test"
	t.Cleanup(func() { nonInteractive = prev })
}

func TestNonInteractiveAskStringKeepsDefault(t *testing.T) {
	forceNonInteractive(t)
	c := newPromptTestCmd(t)
	dest := "24GiB"
	if err := askString(c, "name", &dest, "Memory:", ""); err != nil {
		t.Fatalf("askString() error = %v", err)
	}
	if dest != "24GiB" {
		t.Errorf("dest = %q, want the default kept without prompting", dest)
	}
}

func TestNonInteractiveAskSelect(t *testing.T) {
	forceNonInteractive(t)
	c := newPromptTestCmd(t)

	dest := "sigstore"
	if err := askSelect(c, "tool", &dest, "Tool:", "", []string{"sigstore", "notary"}); err != nil {
		t.Errorf("askSelect() with a valid current value should not error, got %v", err)
	}

	dest = ""
	err := askSelect(c, "tool", &dest, "Tool:", "", []string{"sigstore", "notary"})
	if err == nil || !strings.Contains(err.Error(), "--tool") || !strings.Contains(err.Error(), `"sigstore"`) {
		t.Errorf("askSelect() with no value should name the flag and options, got %v", err)
	}
}

func TestNonInteractiveAskConfirmKeepsDefault(t *testing.T) {
	forceNonInteractive(t)
	c := newPromptTestCmd(t)
	dest := false
	if err := askConfirm(c, "yes", &dest, "Sure?", ""); err != nil {
		t.Fatalf("askConfirm() error = %v", err)
	}
	if dest {
		t.Error("askConfirm() changed the default without a flag")
	}
}

func TestRequireValues(t *testing.T) {
	if err := requireValues("model", "phi", "artifact", "a:v1"); err != nil {
		t.Errorf("requireValues() with all values = %v, want nil", err)
	}
	err := requireValues("model", "", "artifact", " ")
	if err == nil || !strings.Contains(err.Error(), "--model") || !strings.Contains(err.Error(), "--artifact") {
		t.Errorf("requireValues() = %v, want both flags named", err)
	}
}

// TestPackageNonInteractiveMissingValues runs the real package command with
// prompts disabled via the environment and no model given: it must fail with
// a message naming the flags instead of trying to open a TTY.
func TestPackageNonInteractiveMissingValues(t *testing.T) {
	t.Setenv("MODEL_CLI_NO_INTERACTIVE", "1")
	t.Setenv("HOME", t.TempDir())
	viper.Reset()

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"package", "--registry", "oras"})
	t.Cleanup(func() { rootCmd.SetOut(nil); rootCmd.SetErr(nil); rootCmd.SetArgs(nil) })

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("package without --model should fail in non-interactive mode")
	}
	for _, want := range []string{"--model", "--model-path", "--artifact", "MODEL_CLI_NO_INTERACTIVE"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error = %q, want it to mention %s", err.Error(), want)
		}
	}
	if strings.Contains(err.Error(), "TTY") {
		t.Errorf("error = %q: a TTY was attempted", err.Error())
	}
}

func TestWizardRefusesWithoutTerminal(t *testing.T) {
	t.Setenv("MODEL_CLI_NO_INTERACTIVE", "1")
	t.Setenv("HOME", t.TempDir())
	viper.Reset()

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"wizard"})
	t.Cleanup(func() { rootCmd.SetOut(nil); rootCmd.SetErr(nil); rootCmd.SetArgs(nil) })

	err := rootCmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "needs a terminal") {
		t.Errorf("wizard without a terminal: error = %v, want a clear refusal", err)
	}
}
