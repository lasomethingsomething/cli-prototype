### Features

#### Clean TUI
Built with huh and lipgloss for a clean terminal experience:
- Color-coded steps and status
- Clear success/warning indicators
- Minimal clutter - only key information
- Intuitive navigation

#### Pluggable Architecture
All tools are pluggable:
- GitOps: Argo CD, Flux
- Registry: ORAS, ModelPack
- Signing: Sigstore (cosign), Notary v2 (notation)
- SBOM: Syft
- MOF: Built-in classifier

#### Secure Supply Chain
Following OpenSSF Model Signing Specification:
- SBOM Generation with Syft
- Cryptographic Signing with Sigstore or Notary v2
- Signature Verification
- MOF Classification
- OCI Artifacts

#### Configuration
Preferences saved to ~/.model-cli.yaml:
```yaml
gitops: argo
registry: oras
signer: sigstore
```
