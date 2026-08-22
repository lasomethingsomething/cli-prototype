### Architecture

```
User
  |
  v
TUI (huh + lipgloss)
  - Clean, color-coded interface
  - Step-by-step guidance
  - Minimal clutter
  |
  v
Commands (cobra)
  - wizard, package, sign, verify, deploy
  |
  v
Workflows (internal/workflow)
  - DeployWorkflow, PackageWorkflow
  |
  v
Providers (pluggable tools)
  - GitOps: Argo, Flux
  - Registry: ORAS, ModelPack
  - Signing: Sigstore, Notary v2
  - SBOM: Syft
  - MOF: Classifier
```

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
