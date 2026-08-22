### Installation

## Quick Install

### Option 1: Build from source in current directory

If you already have the repository cloned:

```bash
# Build the CLI
go build -o model-cli .

# Verify it works
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

## Prerequisites

### Required
- Go 1.20+ (for building from source)
- Git

### Recommended
- A terminal emulator with good Unicode support (iTerm2, VS Code, Alacritty, etc.)
- Terminal width of at least 100 columns for optimal TUI display

## Verify Installation

```bash
# Check the version and available commands
./model-cli --help

# Or if installed via go install
model-cli --help
```

You should see output listing all available commands:
- `wizard` - Full guided workflow
- `package` - Package model as OCI artifact
- `sign` - Sign with Sigstore or Notary v2
- `verify` - Verify artifact signature
- `deploy` - Deploy to Kubernetes
- `serve` - Serve with vLLM or KServe
- `push` - Push artifact to registry
- `admit` - Evaluate artifact for GitOps admission
- `validate` - Validate artifact manifest
- `schedule` - Schedule workload to node

## First Run

Start with the wizard to get the full experience:

```bash
./model-cli wizard
```

Or try a single command:

```bash
./model-cli package --help
```

## Tool Dependencies

Model CLI is lightweight and only requires Go + charm libraries. However, the tools it integrates with need to be installed separately.

The CLI will check if tools are available and provide clear installation instructions when they're missing.

### Optional Tools (Install as needed)

| Tool | Purpose | Install | Required |
|------|---------|---------|----------|
| [oras](https://oras.land/) | OCI registry operations | `brew install oras` | For push/pull |
| [argocd](https://argo-cd.readthedocs.io/) | GitOps with UI | `brew install argoproj/tap/argocd` | For deploy |
| [flux](https://fluxcd.io/) | GitOps with agents | `brew install fluxcd/tap/flux` | For deploy |
| [cosign](https://www.sigstore.dev/) | Signing (Sigstore) | `brew install sigstore/tap/cosign` | For sign/verify |
| [notation](https://notaryproject.dev/) | Signing (Notary v2) | `brew install notation` | For sign/verify |
| [syft](https://github.com/anchore/syft) | SBOM generation | `brew install anchore/syft/syft` | For supply chain |

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
