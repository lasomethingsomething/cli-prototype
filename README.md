# Model CLI

**Your tour guide through the secure ML model deployment journey.**

Model CLI makes it easy to package, sign, verify, and deploy ML models with a clean, guided TUI (Terminal User Interface). It follows a simple principle: **orchestrate the workflow, don't duplicate the tools.**

## Quick Test Drive (5 minutes)

You can test the entire CLI with just a text file - no real model, registry, or GPU required:

```bash
# 1. Create a dummy model
mkdir -p ~/test-model
echo "test" > ~/test-model/model.txt

# 2. Build the CLI
cd cli-prototype
go build -o model-cli .

# 3. Package it
./model-cli package \\
  --model test-model \\
  --model-path ~/test-model \\
  --artifact test:v1 \\
  --registry oras \\
  --registry-url ""
```

The CLI will ask you a few questions (runtime, accelerator, etc.). **These are just metadata - you don't need the actual hardware or software installed.**

After answering, you'll see:
- OCI manifest created with annotations
- SBOM generation attempted (warning if syft not installed)
- MOF classification applied
- Provenance attestation generated

**See [Trial Run Guide](docs/trial-run.md) for a complete walkthrough with explanations.**

---

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

---

## What Makes Model CLI Different

### Orchestrate, Don't Duplicate

Model CLI **doesn't implement** the functionality of other tools. Instead, it:

1. **Collects** your preferences and model information
2. **Attaches** standardized metadata (annotations) to OCI manifests
3. **Validates** that required metadata is present
4. **Hands off** to the right tool for each job
5. **Provides** clear error messages when tools are missing

### The Tools It Integrates With

| What You Want To Do | What Model CLI Does | What External Tool Does |
|---------------------|--------------------|------------------------|
| Package a model | Collects model info, creates manifest, injects annotations | ORAS/ModelPack creates OCI artifact |
| Generate SBOM | Sets up SBOM config, attaches to manifest | Syft generates the SBOM |
| Sign artifact | Sets up signing config | Cosign/Notation creates signature |
| Verify signature | Checks manifest | Cosign/Notation verifies |
| Deploy to K8s | Collects GitOps preferences, validates | Argo CD/Flux deploys |
| Validate nodes | Collects requirements, checks cluster | kubectl queries nodes |

### You Only Need Go

The CLI itself requires only Go. All other tools are **optional** and checked at runtime:

```bash
# Build the CLI (only Go required)
go build -o model-cli .

# Run any command - CLI tells you if other tools are needed
./model-cli package --model my-model --model-path ./my-model
# If ORAS isn't installed: "oras not installed. Install with: brew install oras"
```

---

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

**All metadata is attached automatically. No manual JSON editing.**

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

# Validate for GitOps deployment
./model-cli validate-gitops --artifact my-model:v1

# Evaluate artifact for admission
./model-cli admit --artifact my-model:v1

# Validate node hardware requirements
./model-cli validate-nodes --artifact my-model:v1

# Validate runtime availability
./model-cli validate-runtime --artifact my-model:v1

# Package with node requirements
./model-cli package --gpu-type nvidia-h100 --vram-min 80GiB --gpu-topology 8xH100

# Package with runtime requirements
./model-cli package --runtime-type vllm --layer-dedup true
```

---

## What is Model CLI?

Model CLI is a tour guide CLI that helps you:

- Package ML models as OCI artifacts with standardized annotations
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
- Validate cluster nodes match hardware requirements
- Validate runtime operators are available
- Deploy to Kubernetes with Argo or Flux
- Serve models with vLLM or KServe

All through a clean, colorful, step-by-step TUI that shows you exactly what's happening at each step.

---

## Features

### Tour Guide Experience
- Interactive TUI with huh and lipgloss
- Clean, color-coded output
- Step-by-step guidance through complex workflows
- Clear success, warning, and error indicators
- Context panel with progress, config, model info, and logs

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
./model-cli package \\
  --model phi-4-mini \\
  --model-path ./models \\
  --artifact my-model:v1 \\
  --registry oras
```

---

## Tool Requirements

**Model CLI itself only needs Go.** The tools it integrates with are optional and checked at runtime with clear installation instructions:

| Tool | Install | Purpose |
|------|---------|---------|
| oras | `brew install oras` | OCI registry operations |
| argocd | `brew install argoproj/tap/argocd` | GitOps (UI-based) |
| flux | `brew install fluxcd/tap/flux` | GitOps (agent-based) |
| cosign | `brew install sigstore/tap/cosign` | Signing (Sigstore) |
| notation | `brew install notation` | Signing (Notary v2) |
| syft | `brew install anchore/syft/syft` | SBOM generation |

**If a tool isn't installed, Model CLI will tell you exactly how to install it.**

Example error message:
```
oras not installed. Install with: brew install oras
```

---

## Understanding the "Errors"

You may see warning messages like:

```
Warning: SBOM generation failed: syft not installed. Install with: brew install anchore/syft/syft
```

**This is not a failure - it's a feature!** The CLI is:

1. Checking if required tools are available
2. Telling you exactly which tool is missing
3. Giving you the exact command to install it
4. Continuing with the workflow anyway

This is the "orchestrate and hand off" design. The CLI doesn't crash when tools are missing - it guides you to install them.

---

## Key Concepts

### OCI Artifacts

Model CLI packages models as **OCI artifacts** - the same standard format used by Docker, Kubernetes, and container registries. This means:

- Your models can be stored in any OCI-compliant registry (Docker Hub, GHCR, AWS ECR, etc.)
- You can use standard OCI tools to push, pull, and manage models
- GitOps tools can discover and deploy models without special integration

### Annotations

Annotations are **metadata attached to OCI manifests**. Model CLI automatically injects:

- **Profile annotations**: Version, artifact type
- **Security annotations**: Signing framework, SBOM format, provenance type
- **MOF annotations**: Openness class, version, components
- **Runtime annotations**: Serving runtime, accelerator, memory
- **Infrastructure annotations**: GPU type, vRAM, topology

These annotations enable:
- Policy engines to validate models without downloading them
- GitOps tools to route models to appropriate clusters
- Registry tools to index and search models
- Deployment tools to match models to hardware

### Metadata Contract

The **Standardized Metadata Contract** ensures all models have the required annotations for:

- **Security**: Signature verification, SBOM presence
- **Compliance**: MOF classification, license info
- **Deployment**: Runtime requirements, hardware needs
- **Discovery**: Model type, relationships, dependencies

---

## Documentation

- [Trial Run Guide](docs/trial-run.md) - 5-minute test drive with explanations
- [Installation](docs/installation.md) - Get Model CLI installed
- [Quick Start](docs/quick-start.md) - Try the wizard
- [Architecture](docs/architecture.md) - Understand how it works
- [Enterprise OCI Registry](docs/enterprise-oci-registry.md) - Phase 2: Registry integration
- [GitOps Admission & Policy Enforcement](docs/gitops-admission.md) - Phase 3: Kubernetes production cluster
- [TUI Guide](docs/tui.md) - Learn about the terminal interface
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
