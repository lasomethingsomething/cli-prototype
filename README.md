# Model CLI 🚀

**Your tour guide through the secure ML model deployment journey.**

Model CLI makes it **easy, fun, and secure** to package, sign, verify, and deploy ML models. It's designed as a clean, uncluttered TUI that guides you through each step of the workflow.

[![Go Report Card](https://goreportcard.com/badge/github.com/lasomethingsomething/cli-prototype)](https://goreportcard.com/report/github.com/lasomethingsomething/cli-prototype)
[![skills.sh](https://skills.sh/b/lasomethingsomething/cli-prototype)](https://skills.sh/lasomethingsomething/cli-prototype)

## ✨ The Magic

Turn a complex ML deployment into a simple, guided journey:

```bash
model-cli wizard
```

This single command walks you through:
1. ✅ **Package** your model as an OCI artifact (with SBOM + MOF classification)
2. ✅ **Sign** with Sigstore or Notary v2 for provenance
3. ✅ **Verify** the signature to ensure trust
4. ✅ **Deploy** to Kubernetes with Argo or Flux

All with a beautiful, clean TUI that shows you exactly what's happening.

## 🎯 Why Model CLI?

| Problem | Solution |
|---------|----------|
| ML deployment is complex | Guided wizard simplifies every step |
| Supply chain security is hard | SBOM, signing, verification built-in |
| Too many tools to learn | Pluggable architecture, one interface |
| Standards are confusing | Aligns with OCI, OSSF, MOF automatically |
| Hard to know what to do | Clear, actionable guidance at each step |

## 📦 Installation

```bash
# From source
go install github.com/lasomethingsomething/cli-prototype@latest

# Or build locally
git clone https://github.com/lasomethingsomething/cli-prototype.git
cd cli-prototype
go build -o model-cli .
./model-cli --help
```

## 🚀 Quick Start

### The Full Journey (Recommended)

```bash
model-cli wizard
```

Follow the interactive prompts. That's it! 🎉

### Individual Steps (For Automation)

```bash
# Package your model
model-cli package

# Sign the artifact
model-cli sign

# Verify before deployment
model-cli verify

# Deploy to Kubernetes
model-cli deploy
```

## 🎨 Features

### 🎭 Clean, Uncluttered TUI

Built with **[charmbracelet/huh](https://github.com/charmbracelet/huh)** and **[lipgloss](https://github.com/charmbracelet/lipgloss)** for a beautiful terminal experience:

- Color-coded steps and status
- Clear success/warning indicators
- Minimal clutter - only key information
- Intuitive navigation

### 🔧 Pluggable Architecture

All tools are **pluggable** - swap them without changing your workflow:

| Category | Supported Tools |
|----------|----------------|
| **GitOps** | Argo CD, Flux |
| **Registry** | ORAS, ModelPack |
| **Signing** | Sigstore (cosign), Notary v2 (notation) |
| **SBOM** | Syft |
| **MOF** | Built-in classifier |

### 🔒 Secure Supply Chain

Following **OpenSSF Model Signing Specification** best practices:

- **SBOM Generation** - Full transparency of model contents
- **Cryptographic Signing** - Sigstore or Notary v2 for provenance
- **Signature Verification** - Validate before deployment
- **MOF Classification** - Model Openness Framework compliance
- **OCI Artifacts** - Standard, interoperable format

### 💾 Configuration

Preferences are saved to `~/.model-cli.yaml`:

```yaml
# Example config
gitops: argo
registry: oras
signer: sigstore
```

## 📚 Commands

| Command | Description |
|---------|-------------|
| `wizard` | **Recommended** - Full guided workflow |
| `package` | Package model as OCI artifact |
| `sign` | Sign artifact with Sigstore/Notary |
| `verify` | Verify artifact signature |
| `deploy` | Deploy to Kubernetes |

### Wizard Options

```bash
# Skip signing (package and deploy only)
model-cli wizard --skip-signing

# Skip deployment (package and sign only)
model-cli wizard --skip-deploy

# Both
model-cli wizard --skip-signing --skip-deploy
```

### Package Options

```bash
model-cli package
# Interactively prompts for:
# - Model name
# - Model path
# - Artifact name
# - Registry URL
# - Include RAG context?
```

### Sign Options

```bash
model-cli sign
# Interactively prompts for:
# - Artifact to sign
# - Signing tool (sigstore/notary)
# - Key to use
```

### Verify Options

```bash
model-cli verify
# Interactively prompts for:
# - Artifact to verify
# - Signing tool used
```

### Deploy Options

```bash
model-cli deploy
# Interactively prompts for:
# - GitOps tool (argo/flux)
# - Registry tool (oras/modelpack)
# - Model name
# - Has Kubernetes?
# - Git repository URL
# - Manifest path
```

## 🌟 Example Workflows

### Local Development

```bash
# Package model without deploying
model-cli wizard --skip-deploy --skip-signing

# Later, when ready for production:
model-cli sign --artifact my-model:v1
model-cli verify --artifact my-model:v1
model-cli deploy
```

### CI/CD Pipeline

```bash
#!/bin/bash
set -e

# Build and package
model-cli package --model phi-4-mini --registry oras --output my-model:v1

# Sign with key from secrets
model-cli sign --artifact my-model:v1 --signer sigstore --key $SIGNING_KEY

# Verify
model-cli verify --artifact my-model:v1

# Deploy to staging
export KUBECONFIG=staging-config
model-cli deploy --gitops argo --repo $REPO_URL --path ./manifests

# Deploy to production
export KUBECONFIG=prod-config
model-cli verify --artifact my-model:v1
model-cli deploy --gitops argo --repo $REPO_URL --path ./manifests
```

### Team Onboarding

```bash
# First time user - start with wizard
model-cli wizard

# Team lead - show available commands
model-cli --help

# Check what's installed
model-cli package --help
model-cli sign --help
```

## 🏗️ Architecture

```
┌─────────────────────────────────────────────────────────┐
│                        User                             │
└─────────────────────┬───────────────────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────────────────┐
│                    TUI (huh + lipgloss)                    │
│  ✓ Clean, color-coded interface                            │
│  ✓ Step-by-step guidance                                  │
│  ✓ Minimal clutter                                        │
└─────────────────────┬───────────────────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────────────────┐
│                  Commands (cobra)                          │
│  wizard, package, sign, verify, deploy                      │
└─────────────────────┬───────────────────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────────────────┐
│                 Workflows (internal/workflow)               │
│  DeployWorkflow, PackageWorkflow                          │
└─────────────────────┬───────────────────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────────────────┐
│                 Providers (pluggable tools)                  │
│  GitOps: Argo, Flux                                           │
│  Registry: ORAS, ModelPack                                  │
│  Signing: Sigstore, Notary v2                               │
│  SBOM: Syft                                                 │
│  MOF: Classifier                                           │
└───────────────────────────────────────────────────────────┘
```

## 🔐 Security & Standards

### Standards Compliance

| Standard | Compliance | Purpose |
|----------|------------|---------|
| **OCI Spec** | ✅ Full | Artifact format & registry interop |
| **OSSF Model Signing** | ✅ Full | Signing & verification |
| **MOF** | ✅ Full | Model classification |
| **SBOM** | ✅ Full | Supply chain transparency |
| **GitOps** | ✅ Full | Deployment best practices |

### Security Features

- **SBOM Generation** - Know exactly what's in your model artifact
- **Cryptographic Signing** - Prove where your model came from
- **Signature Verification** - Ensure models haven't been tampered with
- **MOF Classification** - Understand model openness level
- **Secure by Default** - All security steps enabled automatically

## 📖 Documentation

- **[SKILL.md](skills/model-cli/SKILL.md)** - AI agent guidance (skills.sh compatible)
- This README - Human-friendly guide
- `--help` on all commands - Built-in documentation

## 🎓 Learning Resources

Learn more about the standards and tools:

- **[OCI Specification](https://specs.opencontainers.org/image-spec/)** - Open Container Initiative
- **[OSSF Model Signing Spec](https://github.com/ossf/model-signing-spec)** - OpenSSF
- **[Model Openness Framework](https://github.com/Adopt-MOF/MOF)** - MOF
- **[Sigstore](https://www.sigstore.dev/)** - Software Signing
- **[Notary v2](https://github.com/notaryproject/notaryproject)** - Cloud-native signing
- **[ORAS](https://oras.land/)** - OCI Registry As Storage
- **[ModelPack](https://modelpack.ai/)** - Model packaging
- **[Argo CD](https://argo-cd.readthedocs.io/)** - GitOps
- **[Flux CD](https://fluxcd.io/)** - GitOps with agents

## 🤝 Contributing

We welcome contributions! Here's how to help:

### Adding a New Tool

Follow the **provider pattern**:

1. Add interface in `internal/workflow/providers.go`
2. Implement concrete provider
3. Register in factory function (`Get*Provider`)
4. Update CLI commands to use it

Example:

```go
// Step 1: Define interface
type NewToolProvider interface {
    Name() string
    IsInstalled() bool
    DoThing() error
}

// Step 2: Implement
type CoolTool struct{}
func (c *CoolTool) Name() string { return "cooltool" }
// ... implement methods

// Step 3: Register
func GetNewToolProvider(name string) (NewToolProvider, error) {
    switch name {
    case "cooltool": return &CoolTool{}, nil
    default: return nil, fmt.Errorf("unknown tool")
    }
}
```

### Development Setup

```bash
git clone https://github.com/lasomethingsomething/cli-prototype.git
cd cli-prototype
go build -o model-cli .
./model-cli --help
```

### Running Tests

```bash
go test ./...
```

## 📜 License

[Apache License 2.0](LICENSE)

## 🙏 Acknowledgments

- **[charmbracelet](https://charm.sh/)** - For amazing TUI libraries (huh, lipgloss, bubbles)
- **[spf13/cobra](https://github.com/spf13/cobra)** - For CLI framework
- **[spf13/viper](https://github.com/spf13/viper)** - For configuration
- **[OpenSSF](https://openssf.org/)** - For security standards
- **[OCI](https://opencontainers.org/)** - For container standards
- **[Shopware CLI](https://github.com/shopware/shopware-cli)** - For TUI inspiration

---

**Made with ❤️ for the ML community**

*Model CLI - Your guide from model to production*