### Quick Start

Get started with Model CLI in just a few minutes.

## Prerequisites

- Go 1.20+ (for building from source)
- A terminal with reasonable width (100+ columns recommended)

## Installation

### Option 1: Install from source (recommended for development)

```bash
git clone https://github.com/lasomethingsomething/cli-prototype.git
cd cli-prototype
go build -o model-cli .
./model-cli --help
```

### Option 2: Install binary

```bash
# From GitHub releases (when available)
# Download the binary for your platform from:
# https://github.com/lasomethingsomething/cli-prototype/releases

# Make it executable
chmod +x model-cli

# Verify installation
./model-cli --help
```

## Try the Wizard

The easiest way to experience Model CLI is through the interactive wizard:

```bash
./model-cli wizard
```

This will guide you through the complete workflow step-by-step:

1. **Setup Preferences** - Choose your tools (registry, GitOps, signing)
2. **Model Details** - Enter your model information
3. **Kubernetes Setup** - Configure deployment options
4. **Package** - Create OCI artifact with SBOM and MOF classification
5. **Sign** - Sign your artifact (optional, can be skipped)
6. **Verify** - Verify the signature (optional, can be skipped)
7. **Deploy** - Deploy to Kubernetes (optional, can be skipped)

### Wizard Skip Flags

You can skip certain steps if you don't need them:

```bash
# Skip signing and verification (just package)
./model-cli wizard --skip-signing

# Skip deployment (just package and sign)
./model-cli wizard --skip-deploy

# Skip both signing and deployment (just package)
./model-cli wizard --skip-signing --skip-deploy
```

### What You'll Need

The wizard will prompt you for:
- **Model files** - Path to your model directory
- **Model name** - A name for your model (e.g., "phi-4-mini")
- **Artifact name** - OCI artifact reference (e.g., "my-org/my-model:v1.0.0")
- **Tool preferences** - Which registry, GitOps, and signing tools to use

### Tool Requirements

Model CLI checks if required tools are installed and provides installation instructions:

| Tool | Purpose | Install Command |
|------|---------|----------------|
| oras | OCI registry | `brew install oras` |
| argocd | GitOps (UI) | `brew install argoproj/tap/argocd` |
| flux | GitOps (agents) | `brew install fluxcd/tap/flux` |
| cosign | Signing (Sigstore) | `brew install sigstore/tap/cosign` |
| notation | Signing (Notary v2) | `brew install notation` |
| syft | SBOM generation | `brew install anchore/syft/syft` |

## Individual Commands

For automation or when you want to run specific steps:

```bash
# Package only
./model-cli package

# Sign only
./model-cli sign

# Verify only
./model-cli verify

# Deploy only
./model-cli deploy
```

## Next Steps

After trying the wizard:

1. **For local development**: Use `model-cli package --skip-deploy`
2. **For production**: Use the full `model-cli wizard` workflow
3. **For CI/CD**: Use non-interactive flags with all commands

See [TUI documentation](tui.md) for more details on interactive vs non-interactive modes.
