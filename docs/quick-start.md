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
