# Model CLI

Your tour guide through the secure ML model deployment journey.

Model CLI makes it easy to package, sign, verify, and deploy ML models with a clean, guided TUI.

## Overview

Start with the full guided journey:

```bash
model-cli wizard
```

This walks you through packaging, signing, verifying, and deploying your model.

Individual commands are also available:

```bash
model-cli package    # Package model as OCI artifact
model-cli sign       # Sign with Sigstore or Notary v2
model-cli verify     # Verify artifact signature
model-cli deploy     # Deploy to Kubernetes
model-cli serve      # Serve with vLLM or KServe
```

## Features

### Pluggable Architecture

All tools are pluggable via a common provider interface:

- GitOps: Argo CD, Flux
- Registry: ORAS, ModelPack
- Signing: Sigstore (cosign), Notary v2 (notation)
- SBOM: Syft
- Runtime: vLLM, KServe

### Secure Supply Chain

- SBOM Generation via Syft
- Cryptographic Signing with Sigstore or Notary v2
- Signature Verification
- MOF Classification
- OCI Artifacts

### Clean TUI

Built with huh and lipgloss:
- Step-by-step guided workflows
- Clear success and warning indicators
- Minimal clutter, maximum clarity

### Configuration

Preferences saved to ~/.model-cli.yaml:

```yaml
gitops: argo
registry: oras
signer: sigstore
```

## Workflows

### Local Development

```bash
model-cli wizard --skip-deploy --skip-signing
```

### Production Deployment

```bash
model-cli package
model-cli sign
model-cli verify
model-cli deploy
```

### CI/CD Pipeline

```bash
model-cli package --model phi-4-mini --registry oras
model-cli sign --artifact my-model:v1 --signer sigstore --key $SIGNING_KEY
model-cli verify --artifact my-model:v1
model-cli deploy --gitops argo --repo $REPO_URL --path ./manifests
```

## Standards

- OCI Specification
- OSSF Model Signing Spec
- Model Openness Framework
- SBOM
- GitOps

## Architecture

```
User -> TUI -> Commands -> Workflows -> Providers
```

Providers:
- GitOps: Argo, Flux
- Registry: ORAS, ModelPack
- Signing: Sigstore, Notary v2
- SBOM: Syft
- Runtime: vLLM, KServe

## Documentation

See [docs](docs/) for details:

- [Installation](docs/installation.md)
- [Architecture](docs/architecture.md)
- [Resources](docs/resources.md)

## Contributing

Follow the provider pattern:

1. Add interface in internal/workflow/providers.go
2. Implement concrete provider
3. Register in factory function (Get*Provider)
4. Update CLI commands to use it

## License

Apache License 2.0
