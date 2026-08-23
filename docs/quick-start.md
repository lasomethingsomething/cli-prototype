# Quick Start

**For when you're serious about using Model CLI.** This guide gets you properly set up and running.

## Installation

```bash
go install github.com/lasomethingsomething/cli-prototype@latest
```

This installs the `model-cli` binary to `$GOPATH/bin`. Make sure this is in your PATH.

### Verify Installation

```bash
model-cli --help
```

## First Run

Start with the wizard for the full guided experience:

```bash
model-cli wizard
```

The wizard will guide you through:
1. **Setup Preferences** - Choose your tools (registry, GitOps, signing)
2. **Model Details** - Enter your model information
3. **Kubernetes Setup** - Configure deployment options
4. **Package** - Create OCI artifact with standardized annotations
5. **Compliance Check** - Local validation (annotations, SBOM, MOF)
6. **Sign** - Sign your artifact (optional)
7. **Verify** - Verify the signature (optional)
8. **Deploy** - Deploy to Kubernetes (optional)

### Skip Steps

```bash
# Just package, no signing or deployment
model-cli wizard --skip-signing --skip-deploy

# Package and sign, but skip deployment
model-cli wizard --skip-deploy
```

## Quick Examples

### Full Guided Journey
```bash
model-cli wizard
```

### Individual Commands
```bash
# Package
model-cli package --model phi-4-mini --model-path ./models --registry oras

# Local compliance check
model-cli check --model-path ./models

# Sign
model-cli sign --artifact my-model:v1

# Validate metadata contract
model-cli validate --manifest my-manifest.json

# Validate for GitOps deployment
model-cli validate-gitops --artifact my-model:v1

# Evaluate artifact for admission
model-cli admit --artifact my-model:v1

# Validate node hardware requirements
model-cli validate-nodes --artifact my-model:v1

# Validate runtime availability
model-cli validate-runtime --artifact my-model:v1

# Package with node requirements
model-cli package --gpu-type nvidia-h100 --vram-min 80GiB --gpu-topology 8xH100

# Package with runtime requirements
model-cli package --runtime-type vllm --layer-dedup true

# Package agentic skills (agentskills.io format)
model-cli package --skill --skill-refs "skill1:sha256:abc,skill2:sha256:def"

# Map relationships between assets
model-cli map --model my-model --skill my-skill --pipeline my-pipeline
```

## Understanding the Warnings

You may see messages like:

```
Warning: SBOM generation failed: syft not installed. Install with: brew install anchore/syft/syft
```

**This is not a failure - it's a feature!** The CLI:

1. Checks if required tools are available
2. Tells you exactly which tool is missing
3. Gives you the exact command to install it
4. Continues with the workflow anyway

This is the "orchestrate and hand off" design. The CLI doesn't crash when tools are missing - it guides you to install them.

## Tool Requirements

Model CLI only requires Go. The tools it integrates with are checked at runtime:

| Tool | Purpose | Install | Required for |
|------|---------|---------|---------------|
| oras | OCI registry | `brew install oras` | Package/Push |
| argocd | GitOps (UI) | `brew install argoproj/tap/argocd` | Deploy |
| flux | GitOps (agents) | `brew install fluxcd/tap/flux` | Deploy |
| cosign | Signing (Sigstore) | `brew install sigstore/tap/cosign` | Sign/Verify |
| notation | Signing (Notary v2) | `brew install notation` | Sign/Verify |
| syft | SBOM generation | `brew install anchore/syft/syft` | Supply chain |

**Note:** The CLI will tell you exactly how to install any missing tool when you need it.

## Configuration

Model CLI saves your preferences to `~/.model-cli.yaml`. After running the wizard once, it remembers your choices.

Example config:
```yaml
gitops: flux
registry: oras
signer: sigstore
runtime: vllm
```

Override at any time with command-line flags.

## Troubleshooting

### Build errors
```bash
go mod tidy
go build -o model-cli .
```

### "command not found"
Ensure `$GOPATH/bin` is in your PATH, or use the full path to the binary.

### Terminal display issues
Use a terminal with at least 100 columns and 24-bit color support (iTerm2, VS Code, Alacritty, etc.).
