# Model CLI

Your tour guide through the secure ML model deployment journey.

Model CLI makes it easy to package, sign, verify, and deploy ML models with a clean, guided TUI (Terminal User Interface).

## Get Started

### If you already have the repository:

```bash
cd cli-prototype
go build -o model-cli .
./model-cli wizard
```

### If you're starting fresh:

```bash
git clone https://github.com/lasomethingsomething/cli-prototype.git
cd cli-prototype
go build -o model-cli .
./model-cli wizard
```

That's it! The wizard will guide you through every step.

## What is Model CLI?

Model CLI is a tour guide CLI that helps you:

- Package ML models as OCI artifacts
- Package agentic skills using the agentskills.io standard format
- Generate SBOMs for supply chain transparency
- Classify models with MOF (Model Openness Framework)
- Sign artifacts with Sigstore or Notary v2
- Verify signatures before deployment
- Deploy to Kubernetes with Argo or Flux
- Serve models with vLLM or KServe

All through a clean, colorful, step-by-step TUI that shows you exactly what's happening.

## Quick Examples

### Full Guided Journey (Recommended)

```bash
./model-cli wizard
```

Follow the prompts. The wizard will:
1. Ask about your model (name, path, etc.)
2. Ask about your tool preferences (registry, GitOps, etc.)
3. Package your model as an OCI artifact
4. Optionally sign and verify
5. Optionally deploy to Kubernetes

### Skip Steps You Don't Need

```bash
# Just package, skip signing and deployment
./model-cli wizard --skip-signing --skip-deploy

# Package and sign, but skip deployment
./model-cli wizard --skip-deploy
```

### Individual Commands

```bash
# Package only
./model-cli package --model phi-4-mini --model-path ./models

# Sign only
./model-cli sign --artifact my-model:v1 --signer sigstore

# Deploy only
./model-cli deploy --gitops argo --registry oras --model my-model
```

### Agentic Skills

Package agentic skills using the agentskills.io standard format:

```bash
# Package a skill directory
./model-cli package --model my-skill --model-path ./skills/my-skill --skill

# Or use the short form
./model-cli package --skill --model-path ./my-skill --artifact my-org/my-skill:v1
```

Agentic skills are packaged as OCI artifacts with the `org.cncf.ai.artifact.type=skill` annotation,
following the agentskills.io standard format. This enables:
- Skill discovery and reuse across projects
- Composable AI workflows
- Standardized skill packaging

## Features

### Tour Guide Experience
- Interactive TUI with huh and lipgloss
- Clean, color-coded output
- Step-by-step guidance through complex workflows
- Clear success and warning indicators

### Pluggable Architecture
All tools are pluggable via interfaces:
- GitOps: Argo CD, Flux
- Registry: ORAS, ModelPack
- Signing: Sigstore (cosign), Notary v2 (notation)
- SBOM: Syft
- Runtime: vLLM, KServe

### Secure Supply Chain
- SBOM Generation (Syft)
- Cryptographic Signing (Sigstore/Notary v2)
- Signature Verification
- MOF Classification
- OCI Artifact format

### Non-Interactive Mode
Every interactive command also supports non-interactive mode via flags for CI/CD and automation:

```bash
# Same as interactive, but via flags
./model-cli package \
  --model phi-4-mini \
  --model-path ./models \
  --artifact my-model:v1 \
  --registry oras
```

## Tool Requirements

Model CLI itself only needs Go. The tools it integrates with are optional and checked at runtime:

| Tool | Install | Purpose |
|------|---------|---------|
| oras | `brew install oras` | OCI registry |
| argocd | `brew install argoproj/tap/argocd` | GitOps (UI) |
| flux | `brew install fluxcd/tap/flux` | GitOps (agents) |
| cosign | `brew install sigstore/tap/cosign` | Signing |
| notation | `brew install notation` | Signing |

If a tool isn't installed, Model CLI will tell you exactly how to install it.

## Documentation

- [Installation](docs/installation.md) - Get Model CLI installed
- [Quick Start](docs/quick-start.md) - Try the wizard
- [Architecture](docs/architecture.md) - Understand how it works
- [TUI](docs/tui.md) - Learn about the terminal interface
- [Resources](docs/resources.md) - Standards and tools

## Skills

AI agent guidance: [skills/model-cli/SKILL.md](skills/model-cli/SKILL.md)

## Standards

Model CLI aligns with:
- [OCI Specification](https://specs.opencontainers.org/image-spec/) - Artifact format
- [OSSF Model Signing Spec](https://github.com/ossf/model-signing-spec) - Signing standards
- [Model Openness Framework](https://github.com/Adopt-MOF/MOF) - Classification
- GitOps principles - Deployment patterns

## Contributing

1. Add interface in internal/workflow/providers.go
2. Implement concrete provider
3. Register in factory function (Get*Provider)
4. Add CLI command
5. Support both interactive and non-interactive modes

## License

Apache License 2.0
