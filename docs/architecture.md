# Architecture

```
User
  |
  v
TUI (huh + lipgloss + bubbletea)
  - Clean, color-coded interface
  - Step-by-step guidance
  - Context side panels (Shopware CLI style)
  - Tab-based navigation (Progress, Config, Model, SBOM, MOF, Logs, Help, Env)
  - Minimal clutter
  |
  v
Commands (cobra)
  - wizard, package, sign, verify, deploy
  - serve, push, admit, validate, schedule
  - harden (Step 2: Local Hardening & Compliance)
  |
  v
Workflows (internal/workflow)
  - DeployWorkflow, PackageWorkflow
  - SignWorkflow, VerifyWorkflow
  - HardenWorkflow (SBOM + MOF classification)
  |
  v
Providers (pluggable tools)
  - GitOps: Argo, Flux
  - Registry: ORAS, ModelPack
  - Signing: Sigstore (cosign), Notary v2 (notation)
  - SBOM: Syft
  - MOF: Classifier
  - Runtime: vLLM, KServe
```

#### TUI Architecture

The Model CLI uses a layered TUI architecture inspired by Shopware CLI:

1. **Main Workflow Panels**: Each major step (Package, Harden, Sign, etc.) has its own TUI model
2. **Context Panel**: Right-side panel with tabs showing:
   - Progress: Current step and completion percentage
   - Config: Tool configuration (registry, gitops, signer, runtime)
   - Model: Model information (name, path, artifact)
   - SBOM: SBOM generation status and results
   - MOF: MOF classification status and results
   - Logs: Recent operation logs
   - Help: Keyboard shortcuts
   - Env: Environment information

3. **Two-Column Layout**: Main content on the left, context panel on the right

#### Provider Pattern

All external tool integrations follow the same pattern:

1. Define interface in providers.go
2. Implement concrete provider
3. Register in factory function (Get*Provider)
4. Use via dependency injection in workflows

Example:
```go
type NewToolProvider interface {
    Name() string
    IsInstalled() bool
    DoThing() error
}

type CoolTool struct{}
func (c *CoolTool) Name() string { return "cooltool" }

func GetNewToolProvider(name string) (NewToolProvider, error) {
    switch name {
    case "cooltool": return &CoolTool{}, nil
    default: return nil, fmt.Errorf("unknown tool")
    }
}
```
