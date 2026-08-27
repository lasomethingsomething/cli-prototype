package cmd

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// The "Next: ..." hints and help examples name other model-cli commands.
// Every one of them must resolve to a registered command, so a hint can
// never point at a command that does not exist (issue #104).
func TestHintsReferenceExistingCommands(t *testing.T) {
	hint := regexp.MustCompile(`model-cli ([a-z][a-z-]*)(?: ([a-z][a-z-]*))?`)

	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		src, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range hint.FindAllStringSubmatch(string(src), -1) {
			c := findSubcommand(rootCmd, m[1])
			if c == nil {
				t.Errorf("%s: hint %q names unknown command %q", file, m[0], m[1])
				continue
			}
			// Only treat the second word as a subcommand for commands that have
			// subcommands; otherwise it is prose ("run model-cli package first").
			if m[2] != "" && c.HasSubCommands() && findSubcommand(c, m[2]) == nil {
				t.Errorf("%s: hint %q names unknown subcommand %q of %q", file, m[0], m[2], m[1])
			}
		}
	}
}

func findSubcommand(parent *cobra.Command, name string) *cobra.Command {
	for _, c := range parent.Commands() {
		if c.Name() == name {
			return c
		}
	}
	return nil
}
