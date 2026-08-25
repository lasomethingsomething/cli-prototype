package cmd

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func findCommand(parent *cobra.Command, name string) *cobra.Command {
	for _, c := range parent.Commands() {
		if c.Name() == name {
			return c
		}
	}
	return nil
}

func TestValidateSubcommandsAndFlags(t *testing.T) {
	want := map[string][]string{
		"manifest":  {"artifact", "registry", "manifest", "json-schema", "strict", "artifact-type", "check-relationships"},
		"local":     {"model", "model-path", "artifact-path", "artifact", "strict"},
		"gitops":    {"artifact", "registry", "quiet", "json-output", "env", "region"},
		"admission": {"artifact", "registry", "quiet", "json-output", "env", "region", "strict"},
		"nodes":     {"artifact", "registry", "quiet", "json-output", "namespace"},
		"runtime":   {"artifact", "registry", "quiet", "json-output", "namespace"},
	}
	for name, flags := range want {
		sub := findCommand(validateCmd, name)
		if sub == nil {
			t.Errorf("validate %s: subcommand not registered", name)
			continue
		}
		for _, f := range flags {
			if sub.Flags().Lookup(f) == nil {
				t.Errorf("validate %s: flag --%s missing", name, f)
			}
		}
	}
	// The bare `validate` keeps the manifest flags for backwards compatibility.
	for _, f := range want["manifest"] {
		if validateCmd.Flags().Lookup(f) == nil {
			t.Errorf("validate: flag --%s missing on the parent", f)
		}
	}
}

func TestDeprecatedValidateAliases(t *testing.T) {
	aliases := map[string]string{
		"check":            "local",
		"validate-gitops":  "gitops",
		"admit":            "admission",
		"validate-nodes":   "nodes",
		"validate-runtime": "runtime",
	}
	for old, sub := range aliases {
		c := findCommand(rootCmd, old)
		if c == nil {
			t.Errorf("deprecated alias %q not registered", old)
			continue
		}
		if !c.Hidden || !strings.Contains(c.Deprecated, "validate "+sub) {
			t.Errorf("%q should be hidden and deprecated in favour of 'validate %s'; hidden=%v deprecated=%q", old, sub, c.Hidden, c.Deprecated)
		}
		// Same flags as the new subcommand.
		fresh := findCommand(validateCmd, sub)
		fresh.Flags().VisitAll(func(f *pflag.Flag) {
			if c.Flags().Lookup(f.Name) == nil {
				t.Errorf("%q lacks flag --%s that 'validate %s' has", old, f.Name, sub)
			}
		})
	}
}

func TestValidateNonInteractiveRequiresArtifact(t *testing.T) {
	forceNonInteractive(t)
	sub := newValidateGitOpsCmd()
	if err := sub.ParseFlags([]string{"--registry", "oras"}); err != nil {
		t.Fatal(err)
	}
	err := sub.RunE(sub, nil)
	if err == nil || !strings.Contains(err.Error(), "--artifact") {
		t.Errorf("validate gitops without --artifact: error = %v, want it to name --artifact", err)
	}
}
