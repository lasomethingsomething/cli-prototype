package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

// newPromptTestCmd builds a throwaway command with the flags the ask*
// helpers are exercised against, parsed from args.
func newPromptTestCmd(t *testing.T, args ...string) *cobra.Command {
	t.Helper()
	c := &cobra.Command{Use: "t", Run: func(*cobra.Command, []string) {}}
	c.Flags().String("name", "", "")
	c.Flags().String("tool", "", "")
	c.Flags().Bool("yes", false, "")
	c.SetArgs(args)
	if err := c.Execute(); err != nil {
		t.Fatal(err)
	}
	return c
}

// These tests only cover the flag paths: when a flag was given, no prompt
// may run (the tests have no TTY, so a prompt would fail loudly).

func TestAskStringUsesFlagWhenGiven(t *testing.T) {
	c := newPromptTestCmd(t, "--name", "phi-4")
	dest := "default"
	if err := askString(c, "name", &dest, "Name:", ""); err != nil {
		t.Fatalf("askString() error = %v", err)
	}
	if dest != "phi-4" {
		t.Errorf("dest = %q, want flag value phi-4", dest)
	}
}

func TestAskStringRespectsExplicitEmptyFlag(t *testing.T) {
	c := newPromptTestCmd(t, "--name", "")
	dest := "default"
	if err := askString(c, "name", &dest, "Name:", ""); err != nil {
		t.Fatalf("askString() error = %v", err)
	}
	if dest != "" {
		t.Errorf("dest = %q, want empty: an explicit empty flag must win and not prompt", dest)
	}
}

func TestAskSelectUsesFlagWhenGiven(t *testing.T) {
	c := newPromptTestCmd(t, "--tool", "notary")
	dest := ""
	if err := askSelect(c, "tool", &dest, "Tool:", "", []string{"sigstore", "notary"}); err != nil {
		t.Fatalf("askSelect() error = %v", err)
	}
	if dest != "notary" {
		t.Errorf("dest = %q, want notary", dest)
	}
}

func TestAskSelectIfEmptyKeepsExistingValue(t *testing.T) {
	c := newPromptTestCmd(t)
	dest := "sigstore" // e.g. loaded from config
	if err := askSelectIfEmpty(c, "tool", &dest, "Tool:", "", []string{"sigstore", "notary"}); err != nil {
		t.Fatalf("askSelectIfEmpty() error = %v", err)
	}
	if dest != "sigstore" {
		t.Errorf("dest = %q, want the existing value kept without prompting", dest)
	}
}

func TestAskSelectIfEmptyFlagOverridesExisting(t *testing.T) {
	c := newPromptTestCmd(t, "--tool", "notary")
	dest := "sigstore"
	if err := askSelectIfEmpty(c, "tool", &dest, "Tool:", "", []string{"sigstore", "notary"}); err != nil {
		t.Fatalf("askSelectIfEmpty() error = %v", err)
	}
	if dest != "notary" {
		t.Errorf("dest = %q, want the flag to override the existing value", dest)
	}
}

func TestAskConfirmUsesFlagWhenGiven(t *testing.T) {
	for _, arg := range []string{"--yes", "--yes=false"} {
		c := newPromptTestCmd(t, arg)
		dest := true
		if err := askConfirm(c, "yes", &dest, "Sure?", ""); err != nil {
			t.Fatalf("askConfirm(%s) error = %v", arg, err)
		}
		if want := arg == "--yes"; dest != want {
			t.Errorf("askConfirm(%s) dest = %v, want %v", arg, dest, want)
		}
	}
}
