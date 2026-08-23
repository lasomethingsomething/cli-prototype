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
- Run local compliance checks before push
- Sign artifacts with Sigstore or Notary v2
- Verify signatures before deployment
- Validate pushes against standardized metadata contract
- Enforce metadata contract at manifest level
- Cross-reference AI assets in registries
- Map complex relationships between assets (model → skill → pipeline)
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

# Local compliance check
./model-cli check --model-path ./models --artifact-path ./output

# Sign only
./model-cli sign --artifact my-model:v1 --signer sigstore

# Validate metadata contract
./model-cli validate --manifest my-manifest.json
./model-cli validate --manifest my-manifest.json --json-schema --strict

# Enforce metadata contract at registry level
./model-cli enforce --manifest my-manifest.json
./model-cli enforce --webhook --port 8443

# Validate for GitOps deployment (Story #65, #66)
./model-cli validate-gitops --artifact my-model:v1
./model-cli validate-gitops --artifact my-model:v1 --env air-gapped
./model-cli validate-gitops --artifact my-model:v1 --env hybrid-cloud --region us-east-1

# Admission evaluation (Story #67)
./model-cli admit --artifact my-model:v1
./model-cli admit --artifact my-model:v1 --env production
./model-cli admit --artifact my-model:v1 --json-output

# Validate node hardware requirements (Story #68)
./model-cli validate-nodes --artifact my-model:v1
./model-cli validate-nodes --artifact my-model:v1 --namespace production
./model-cli validate-nodes --artifact my-model:v1 --json-output

# Package with node requirements (Story #68)
./model-cli package --gpu-type nvidia-h100 --vram-min 80GiB --gpu-topology 8xH100

# Validate runtime availability (Story #69)
./model-cli validate-runtime --artifact my-model:v1
./model-cli validate-runtime --artifact my-model:v1 --namespace production
./model-cli validate-runtime --artifact my-model:v1 --json-output

# Package with runtime requirements (Story #69)
./model-cli package --runtime-type vllm --layer-dedup true --dlc-endpoint https://dlc.example.com --skill-refs skill:sha256:abc

# Cross-reference assets in registry
./model-cli search --destination ghcr.io/my-org --type model
./model-cli search --uses-model model:sha256:abc123

# Map relationships between assets
./model-cli map --model my-model --skill my-skill --pipeline my-pipeline

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
- Local Compliance Checks (annotations, SBOM, MOF)
- MOF metadata generation with CC-BY-4.0 license by default

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
| syft | `brew install anchore/syft/syft` | SBOM generation (SPDX, CycloneDX) |
| trivy | `brew install aquasecurity/trivy/trivy` | SBOM generation (SPDX, CycloneDX) |
| cdxgen | `npm install -g @cyclonedx/cdxgen` | SBOM generation (CycloneDX) |

If a tool isn't installed, Model CLI will tell you exactly how to install it.

## MOF Metadata License

When generating MOF/MOT-compliant metadata config files (`mof.json`), the default license is **CC-BY-4.0** as recommended by the MOF specification for metadata and documentation. You can override this via:

- `--license` flag on the `model-cli harden` command
- Configuration file setting

Example:
```bash
# Use default CC-BY-4.0
./model-cli harden

# Override with custom license
./model-cli harden --license MIT
```

## Documentation

- [Installation](docs/installation.md) - Get Model CLI installed
- [Quick Start](docs/quick-start.md) - Try the wizard
- [Architecture](docs/architecture.md) - Understand how it works
- [Enterprise OCI Registry](docs/enterprise-oci-registry.md) - Phase 2: Registry integration
- [GitOps Admission & Policy Enforcement](docs/gitops-admission.md) - Phase 3: Kubernetes production cluster
- [TUI](docs/tui.md) - Learn about the terminal interface
- [Resources](docs/resources.md) - Standards and tools

## Skills

AI agent guidance: [skills/model-cli/SKILL.md](skills/model-cli/SKILL.md)

## Standards

Model CLI aligns with:
- [OCI Specification](https://specs.opencontainers.org/image-spec/) - Artifact format
- [OCI Distribution Spec](https://github.com/opencontainers/distribution-spec) - Registry operations
- [OSSF Model Signing Spec](https://github.com/ossf/model-signing-spec) - Signing standards
- [Model Openness Framework](https://github.com/Adopt-MOF/MOF) - Classification
- [JSON Schema](https://json-schema.org/) - Metadata contract validation
- GitOps principles - Deployment patterns

## Contributing

1. Add interface in internal/workflow/providers.go
2. Implement concrete provider
3. Register in factory function (Get*Provider)
4. Add CLI command
5. Support both interactive and non-interactive modes

## License

Apache License 2.0
