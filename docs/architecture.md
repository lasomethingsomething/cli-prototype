# Architecture

```
User
  |
  v
TUI (huh + lipgloss + bubbletea)
  - Clean, color-coded interface
  - Step-by-step guidance
  - Context side panels
  - Tab-based navigation (Progress, Config, Model, SBOM, MOF, Logs, Help, Env)
  - Minimal clutter
  |
  v
Commands (cobra)
  - wizard, package, sign, verify, deploy
  - serve, push, admit, validate, schedule
  - harden (Step 2: Local Hardening & Compliance)
  - search (Story #61: Cross-Reference Assets)
  - map (Story #60: Map Complex Relationships)
  |
  v
Workflows (internal/workflow)
  - DeployWorkflow, PackageWorkflow
  - SignWorkflow, VerifyWorkflow
  - HardenWorkflow (SBOM + MOF classification)
  - AdmissionWebhook (Story #59: Enforce Metadata Contract)
  - MetadataContractValidation (Story #62: Validate Pushes)
  - SearchWorkflow (Story #61: Cross-Reference Assets)
  - RelationshipMapping (Story #60: Map Relationships)
  |
  v
Providers (pluggable tools; one recommended per category, see internal/workflow/options.go)
  - GitOps: Flux (recommended), Argo CD
  - Registry: ORAS (recommended; Harbor and any OCI registry), ModelPack (planned)
  - Signing: cosign (Sigstore, recommended), notation (Notary v2)
  - SBOM: Syft (recommended), Trivy, cdxgen
  - MOF: Classifier
  - Runtime: vLLM, KServe
```

#### TUI Architecture

The Model CLI uses a layered TUI architecture:

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

1. Define the interface in the matching `internal/workflow/*.go` file (`registry.go`, `signing.go`, `sbom.go`, `gitops.go`, `runtime.go`)
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

#### Enterprise OCI Registry Architecture

The Enterprise OCI Registry Integration (Phase 2) follows a **decoupled handoff pattern**:

```
┌─────────────────────────────────────────────────────────────────┐
│                        Model CLI (Orchestrator)                   │
├─────────────────────────────────────────────────────────────────┤
│  Commands: push, validate, enforce, search, map                    │
│  Workflows: Package, Sign, Verify, Deploy, Search, Validate        │
└─────────────────────┬───────────────────────────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Registry Provider Interface                    │
├─────────────────────────────────────────────────────────────────┤
│  - ORAS Provider (full implementation)                           │
│  - ModelPack Provider (stub implementation)                     │
│  - Methods: Push, Pull, Search, GetArtifactDigest, etc.          │
└─────────────────────┬───────────────────────────────────────────┘
                      │
        ┌─────────────┴─────────────┐
        ▼                           ▼
┌───────────────────┐    ┌─────────────────────┐
│      ORAS CLI     │    │    ModelPack CLI     │
│  - oras push      │    │  - modelpack push    │
│  - oras pull      │    │  - modelpack pull    │
│  - oras discover  │    │  - (stub) search     │
└───────────────────┘    └─────────────────────┘
        │                           │
        └─────────────┬─────────────┘
                      ▼
┌─────────────────────────────────────────────────────────────────┐
│                      OCI Registry (ghcr.io, etc.)                │
│  - Stores OCI artifacts with manifest annotations                 │
│  - Supports OCI Distribution Spec filtering                       │
│  - Enables admission webhooks for validation                     │
└─────────────────────────────────────────────────────────────────┘
```

**Key Design Principles:**
1. **CLI as Orchestrator**: Model CLI delegates to registry tools (ORAS, ModelPack)
2. **Manifest-Level Metadata**: All AI-specific metadata stored in OCI annotations
3. **Client-Side Filtering**: Complex queries filtered client-side when needed
4. **Decoupled Validation**: Validation happens at CLI level, enforcement at registry level
5. **Extensible Providers**: Easy to add new registry tools via the RegistryProvider interface

**Data Flow:**
1. User runs `model-cli push` with metadata
2. CLI creates UnifiedOCIManifest with annotations
3. CLI delegates to ORAS/ModelPack to push to registry
4. Registry stores manifest with annotations
5. User runs `model-cli search` to query
6. CLI delegates to ORAS/ModelPack to fetch manifests
7. CLI parses and filters results client-side
8. CLI displays structured results to user
