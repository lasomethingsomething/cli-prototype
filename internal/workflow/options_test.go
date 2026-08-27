package workflow

import "testing"

// TestToolOptionsMatchFactories asserts that every option offered to the user
// is accepted by the corresponding factory, that the provider reports the same
// name (so "X not installed" names the tool the user picked), and that each
// category has exactly one recommended tool.
func TestToolOptionsMatchFactories(t *testing.T) {
	type named interface{ Name() string }
	categories := []struct {
		category string
		options  ToolOptions
		factory  func(string) (named, error)
	}{
		{"registry", RegistryOptions(), func(n string) (named, error) { return GetRegistryProvider(n) }},
		{"signer", SignerOptions(), func(n string) (named, error) { return GetSigningProvider(n) }},
		{"gitops", GitOpsOptions(), func(n string) (named, error) { return GetGitOpsProvider(n) }},
		{"sbom", SBOMToolOptions(), func(n string) (named, error) { return GetSBOMGenerator(n) }},
	}

	for _, c := range categories {
		t.Run(c.category, func(t *testing.T) {
			if len(c.options) == 0 {
				t.Fatal("no options")
			}
			seen := map[string]bool{}
			recommended := 0
			for _, opt := range c.options {
				if seen[opt.Name] {
					t.Errorf("option %q listed twice", opt.Name)
				}
				seen[opt.Name] = true
				if opt.Recommended {
					recommended++
				}
				p, err := c.factory(opt.Name)
				if err != nil {
					t.Errorf("factory rejects option %q: %v", opt.Name, err)
					continue
				}
				if p.Name() != opt.Name {
					t.Errorf("provider for option %q reports Name() = %q", opt.Name, p.Name())
				}
			}
			if recommended != 1 {
				t.Errorf("%d recommended options, want exactly 1", recommended)
			}
			if c.options.Recommended() != c.options[0].Name {
				t.Errorf("Recommended() = %q, want the first option %q", c.options.Recommended(), c.options[0].Name)
			}
		})
	}
}

func TestToolOptionsFormatting(t *testing.T) {
	opts := ToolOptions{
		{Name: "a", Description: "first", Recommended: true},
		{Name: "b"},
	}
	if got := opts.Summary(); got != "a (recommended), b" {
		t.Errorf("Summary() = %q", got)
	}
	if got := opts.Bullets(); got != "  • a (recommended) - first\n  • b" {
		t.Errorf("Bullets() = %q", got)
	}
	if got := opts.Names(); len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Errorf("Names() = %v", got)
	}
	if got := (ToolOptions{{Name: "x"}}).Recommended(); got != "" {
		t.Errorf("Recommended() without a recommended option = %q, want empty", got)
	}
}
