package workflow

import "strings"

// ToolOption is one selectable tool in a category (registry, signer, ...).
// Name is the value the CLI flag and the corresponding Get* factory accept,
// and the value the provider reports from Name().
type ToolOption struct {
	Name        string
	Description string
	Recommended bool
}

// Label is the option as shown in prompts: "name (recommended) - description".
func (o ToolOption) Label() string {
	label := o.Name
	if o.Recommended {
		label += " (recommended)"
	}
	if o.Description != "" {
		label += " - " + o.Description
	}
	return label
}

// ToolOptions is the list of choices for one category. The tools in a
// category are mutually exclusive: exactly one is used per artifact.
type ToolOptions []ToolOption

// Names returns the option names in display order.
func (opts ToolOptions) Names() []string {
	names := make([]string, 0, len(opts))
	for _, o := range opts {
		names = append(names, o.Name)
	}
	return names
}

// Recommended returns the name of the recommended option, the default when
// nothing was chosen.
func (opts ToolOptions) Recommended() string {
	for _, o := range opts {
		if o.Recommended {
			return o.Name
		}
	}
	return ""
}

// Summary lists the names for flag help: "oras (recommended), modelpack".
func (opts ToolOptions) Summary() string {
	parts := make([]string, 0, len(opts))
	for _, o := range opts {
		if o.Recommended {
			parts = append(parts, o.Name+" (recommended)")
		} else {
			parts = append(parts, o.Name)
		}
	}
	return strings.Join(parts, ", ")
}

// Bullets renders the options as an indented bullet list for long help text.
func (opts ToolOptions) Bullets() string {
	var sb strings.Builder
	for _, o := range opts {
		sb.WriteString("  • " + o.Label() + "\n")
	}
	return strings.TrimRight(sb.String(), "\n")
}

// RegistryOptions lists the registry tools GetRegistryProvider accepts.
// Harbor, GHCR, zot and every other OCI registry are reached through ORAS;
// the registry host is part of the artifact reference or --registry-url.
func RegistryOptions() ToolOptions {
	return ToolOptions{
		{Name: "oras", Description: "any OCI registry (Harbor, GHCR, zot, ...)", Recommended: true},
		{Name: "modelpack", Description: "CNCF ModelPack Model Spec artifacts via modctl (needs oras too)"},
	}
}

// SignerOptions lists the signing tools GetSigningProvider accepts.
func SignerOptions() ToolOptions {
	return ToolOptions{
		{Name: "cosign", Description: "Sigstore", Recommended: true},
		{Name: "notary", Description: "Notary v2 (notation)"},
	}
}

// GitOpsOptions lists the GitOps tools GetGitOpsProvider accepts.
func GitOpsOptions() ToolOptions {
	return ToolOptions{
		{Name: "flux", Description: "agent-based GitOps", Recommended: true},
		{Name: "argocd", Description: "UI-based GitOps"},
	}
}

// RuntimeOptions lists the serving runtimes the CLI can hand artifacts to.
func RuntimeOptions() ToolOptions {
	return ToolOptions{
		{Name: "vllm", Description: "high-throughput local inference", Recommended: true},
		{Name: "kserve", Description: "Kubernetes InferenceService"},
	}
}

// SBOMToolOptions lists the SBOM generators GetSBOMGenerator accepts.
func SBOMToolOptions() ToolOptions {
	return ToolOptions{
		{Name: "syft", Description: "SPDX and CycloneDX", Recommended: true},
		{Name: "trivy", Description: "scanner with SBOM output"},
		{Name: "cdxgen", Description: "CycloneDX generator"},
	}
}
