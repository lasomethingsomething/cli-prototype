# Quick Start

Get started with Model CLI in just a few minutes.

## Prerequisites

- Go 1.20+ (for building from source)
- Git
- A terminal with reasonable width (100+ columns recommended for optimal TUI display)
- A terminal emulator with good Unicode support (iTerm2, VS Code, Alacritty, etc.)

## Installation

### Option 1: Build from source (recommended for development)

If you already have the repository cloned:

```bash
cd cli-prototype
go build -o model-cli .
./model-cli --help
```

### Option 2: Clone and build (if starting fresh)

```bash
# Clone the repository
git clone https://github.com/lasomethingsomething/cli-prototype.git
cd cli-prototype

# Build the CLI
go build -o model-cli .

# Verify it works
./model-cli --help
```

### Option 3: Using Go Install

```bash
go install github.com/lasomethingsomething/cli-prototype@latest
```

This will install the binary to `$GOPATH/bin`. Make sure this is in your PATH.

### Verify Installation

```bash
# Check the version and available commands
./model-cli --help

# Or if installed via go install
model-cli --help
```

You should see output listing all available commands including:
- `wizard` - Full guided workflow
- `package` - Package model as OCI artifact
- `sign` - Sign with Sigstore or Notary v2
- `verify` - Verify artifact signature
- `deploy` - Deploy to Kubernetes
- `check` - Local compliance check
- `validate` - Validate artifact manifest
- `validate-gitops` - Pre-flight validation for GitOps
- `validate-nodes` - Validate node hardware requirements
- `validate-runtime` - Validate runtime operator availability
- `admit` - Evaluate artifact for admission
- `map` - Map relationships between assets

## Try the Wizard

The easiest way to experience Model CLI is through the interactive wizard:

```bash
./model-cli wizard
```

This will guide you through the complete workflow step-by-step:

1. **Setup Preferences** - Choose your tools (registry, GitOps, signing)
2. **Model Details** - Enter your model information
3. **Kubernetes Setup** - Configure deployment options
4. **Package** - Create OCI artifact with annotations
5. **Compliance Check** - Local compliance check (annotations, SBOM, MOF)
6. **Sign** - Sign your artifact (optional, can be skipped)
7. **Verify** - Verify the signature (optional, can be skipped)
8. **Deploy** - Deploy to Kubernetes (optional, can be skipped)

The **Step 2: Local Hardening & Compliance** feature includes:
- **SBOM Generation**: Creates a Software Bill of Materials attached to artifact layers
- **MOF Classification**: Applies Model Openness Framework classification (Class I, II, or III)
- **Security Annotations**: Applies signing framework and provenance annotations

The **Compliance Check** validates:
- Required annotations present (org.cncf.ai.artifact.type, runtime, accelerator)
- SBOM present in artifact layers
- MOF classification applied
- Blocks with clear message when required pieces are missing

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

Model CLI checks if required tools are installed and provides installation instructions. All tools are optional - the CLI will tell you exactly how to install any missing dependencies.

| Tool | Purpose | Install Command | Required for |
|------|---------|----------------|---------------|
| [oras](https://oras.land/) | OCI registry operations | `brew install oras` | Package/Push |
| [argocd](https://argo-cd.readthedocs.io/) | GitOps with UI | `brew install argoproj/tap/argocd` | Deploy |
| [flux](https://fluxcd.io/) | GitOps with agents | `brew install fluxcd/tap/flux` | Deploy |
| [cosign](https://www.sigstore.dev/) | Signing (Sigstore) | `brew install sigstore/tap/cosign` | Sign/Verify |
| [notation](https://notaryproject.dev/) | Signing (Notary v2) | `brew install notation` | Sign/Verify |
| [syft](https://github.com/anchore/syft) | SBOM generation | `brew install anchore/syft/syft` | Supply chain |
| [trivy](https://github.com/aquasecurity/trivy) | SBOM generation | `brew install aquasecurity/trivy/trivy` | Supply chain |
| [cdxgen](https://github.com/CycloneDX/cdxgen) | SBOM generation | `npm install -g @cyclonedx/cdxgen` | Supply chain |

**Note:** The `check` command validates local artifacts and requires SBOM (from Syft/Trivy/cdxgen) and MOF classification to be present.

**SBOM Tools:** Multiple SBOM generators are supported:
- **Syft** (default): Generates SPDX and CycloneDX formats
- **Trivy**: Generates SPDX and CycloneDX formats with vulnerability scanning
- **cdxgen**: Generates CycloneDX format

Use `--sbom-tool` and `--sbom-format` flags to select your preferred generator and format.

## Individual Commands

For automation or when you want to run specific steps:

```bash
# Package only
./model-cli package --model phi-4-mini --model-path ./models --registry oras

# Local compliance check
./model-cli check --model-path ./models

# Sign only
./model-cli sign --artifact my-model:v1

# Verify only
./model-cli verify --artifact my-model:v1

# Validate metadata contract
./model-cli validate --manifest my-manifest.json

# Enforce metadata contract
./model-cli enforce --manifest my-manifest.json

# Validate for GitOps deployment
./model-cli validate-gitops --artifact my-model:v1

# Evaluate artifact for admission
./model-cli admit --artifact my-model:v1

# Validate node hardware requirements
./model-cli validate-nodes --artifact my-model:v1

# Validate runtime availability
./model-cli validate-runtime --artifact my-model:v1

# Deploy only
./model-cli deploy

# Search registry
./model-cli search --destination ghcr.io/my-org --type model

# Map relationships
./model-cli map --model my-model --skill my-skill
```

## Next Steps

After trying the wizard:

1. **For local development**: Use `model-cli package --skip-deploy`
2. **For production**: Use the full `model-cli wizard` workflow
3. **For CI/CD**: Use non-interactive flags with all commands

See [TUI documentation](tui.md) for more details on interactive vs non-interactive modes.

## Configuration

Model CLI saves your preferences to `~/.model-cli.yaml`. After running the wizard once, it will remember your tool choices.

Example config:
```yaml
gitops: flux
registry: oras
signer: sigstore
runtime: vllm
```

You can override these at any time with command-line flags.

## Troubleshooting

### Build errors

If you get build errors, try:
```bash
# Update Go modules
go mod tidy

# Rebuild
go build -o model-cli .
```

### "command not found"

Make sure the binary is in your PATH or use `./model-cli` when running from the repo directory.

### Terminal display issues

For best results:
- Use a terminal with at least 100 columns
- Ensure your terminal supports 24-bit color
- Use a modern terminal emulator (iTerm2, VS Code, Alacritty, etc.)
