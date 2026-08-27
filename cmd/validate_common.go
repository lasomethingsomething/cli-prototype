package cmd

import (
	"fmt"
	"os"

	"github.com/lasomethingsomething/cli-prototype/config"
	"github.com/lasomethingsomething/cli-prototype/internal/workflow"
	"github.com/spf13/cobra"
)

// Shared plumbing for the `validate` subcommands: they all resolve an
// artifact reference and a registry tool the same way, fetch the same
// manifest annotations, and render the same ValidationReport.

// addArtifactFlags registers the flags every registry-backed validation takes.
func addArtifactFlags(c *cobra.Command) {
	c.Flags().String("artifact", "", "OCI artifact reference to validate (e.g., ghcr.io/my-org/my-model:latest)")
	c.Flags().String("registry", "", "Registry tool: "+workflow.RegistryOptions().Summary())
}

// addOutputFlags registers the output flags every validation takes.
func addOutputFlags(c *cobra.Command) {
	c.Flags().Bool("quiet", false, "Quiet mode: only print the pass/fail line")
	c.Flags().Bool("json-output", false, "Print the report as JSON for CI/CD integration")
}

type outputOptions struct {
	quiet, jsonOutput bool
}

func outputOptionsFrom(cmd *cobra.Command) outputOptions {
	quiet, _ := cmd.Flags().GetBool("quiet")
	jsonOutput, _ := cmd.Flags().GetBool("json-output")
	return outputOptions{quiet: quiet, jsonOutput: jsonOutput}
}

// silent reports whether progress lines should be suppressed.
func (o outputOptions) silent() bool { return o.quiet || o.jsonOutput }

// resolveArtifact returns the artifact reference (flag or prompt) and the
// installed registry provider (flag, saved config or prompt).
func resolveArtifact(cmd *cobra.Command) (string, workflow.RegistryProvider, error) {
	var artifact string
	if err := askString(cmd, "artifact", &artifact, "Artifact to validate:", "The OCI artifact reference (e.g., ghcr.io/my-org/my-model:latest)"); err != nil {
		return "", nil, err
	}
	if err := requireValues("artifact", artifact); err != nil {
		return "", nil, err
	}
	cfg := config.Load()
	if err := askSelectToolIfEmpty(cmd, "registry", &cfg.Registry, "Registry tool:", "Choose how to fetch the artifact manifest", workflow.RegistryOptions()); err != nil {
		return "", nil, err
	}
	warnIfSaveFails(config.Save(cfg))

	provider, err := workflow.GetRegistryProvider(cfg.Registry)
	if err != nil {
		return "", nil, err
	}
	if !provider.IsInstalled() {
		return "", nil, fmt.Errorf("%s not installed. Install with: %s", provider.Name(), provider.InstallInstructions())
	}
	return artifact, provider, nil
}

// fetchAnnotations resolves the artifact and fetches its manifest annotations.
func fetchAnnotations(cmd *cobra.Command, out outputOptions) (string, map[string]string, error) {
	artifact, provider, err := resolveArtifact(cmd)
	if err != nil {
		return "", nil, err
	}
	if !out.silent() {
		fmt.Printf("\n→ Fetching manifest for %s...\n", artifact)
	}
	annotations, err := provider.FetchManifestAnnotations(artifact)
	if err != nil {
		return artifact, nil, fmt.Errorf("failed to fetch manifest for %s: %v (hint: push the artifact first, or use 'validate manifest --manifest <file>' for a local manifest)", artifact, err)
	}
	if !out.silent() {
		fmt.Println("✓ Fetched artifact manifest from registry")
	}
	return artifact, annotations, nil
}

// reportHints are the closing lines printed under a text report.
type reportHints struct {
	pass, fail string
}

// printReport renders r as JSON, a single line (quiet) or a full text report,
// and returns an error when the report did not pass so the command exits 1.
func printReport(r *workflow.ValidationReport, out outputOptions, hints reportHints) error {
	switch {
	case out.jsonOutput:
		data, err := r.JSON()
		if err != nil {
			return err
		}
		fmt.Println(string(data))
	case out.quiet:
		fmt.Println(r.Summary())
	default:
		printTextReport(r, hints)
	}
	if !r.Passed {
		return fmt.Errorf("%s", r.Summary())
	}
	return nil
}

func printTextReport(r *workflow.ValidationReport, hints reportHints) {
	symbols := map[workflow.CheckStatus]string{
		workflow.CheckPass: "✓", workflow.CheckFail: "✗", workflow.CheckWarn: "⚠", workflow.CheckInfo: "ℹ",
	}
	section := ""
	for _, c := range r.Checks {
		if c.Section != section {
			section = c.Section
			fmt.Printf("\n=== %s ===\n", section)
		}
		if c.Detail != "" {
			fmt.Printf("  %s %s: %s\n", symbols[c.Status], c.Name, c.Detail)
		} else {
			fmt.Printf("  %s %s\n", symbols[c.Status], c.Name)
		}
	}

	fmt.Println("\n=== Result ===")
	if r.Passed {
		fmt.Printf("✓ %s\n", r.Summary())
		if hints.pass != "" {
			fmt.Println(hints.pass)
		}
	} else {
		fmt.Printf("✗ %s\n", r.Summary())
		if hints.fail != "" {
			fmt.Println(hints.fail)
		}
	}
	if len(r.Warnings) > 0 {
		fmt.Println("\nWarnings:")
		for _, w := range r.Warnings {
			fmt.Printf("  ⚠ %s\n", w)
		}
	}
}

// gitopsHints are shared by the gitops and admission targets.
var gitopsHints = reportHints{
	pass: "GitOps tools (Argo CD, Flux) can proceed. External policy engines (Sigstore Policy Controller,\nOPA/Gatekeeper, Kyverno) perform the actual admission control from these annotations.",
	fail: "To fix: re-package with 'model-cli package' (add --sign) so every required annotation is present.",
}

// deprecatedAlias returns a hidden top-level copy of a validate subcommand
// under its old name, so existing scripts keep working.
func deprecatedAlias(oldName string, mk func() *cobra.Command) *cobra.Command {
	alias := mk()
	newName := alias.Name()
	alias.Use = oldName
	alias.Hidden = true
	alias.Deprecated = fmt.Sprintf("use 'model-cli validate %s' instead", newName)
	alias.SetOut(os.Stderr) // keep the deprecation notice off stdout (JSON output)
	return alias
}
