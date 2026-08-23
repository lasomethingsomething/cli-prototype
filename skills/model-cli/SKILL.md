---
name: model-cli
description: Use Model CLI to package, sign, verify, and deploy ML models with a secure supply chain workflow. Aligns with OCI specs, OSSF Model Signing Spec, and Model Openness Framework.
---

# Model CLI

Model CLI is your **tour guide** through the secure ML model deployment journey. It provides a clean, wizard-style TUI with **Shopware CLI-style context panels** that makes complex workflows simple, transparent, and secure.

Use this skill when:
- Packaging ML models as OCI artifacts
- Generating SBOMs for supply chain transparency
- Signing and verifying model artifacts (Sigstore, Notary v2)
- Deploying models to Kubernetes with GitOps (Argo, Flux)
- Classifying models with MOF (Model Openness Framework)
- Following OpenSSF Model Signing Specification best practices

## Core Philosophy

**Simple, Clear, Fun.** The CLI is designed to be:
- **Easy to use** - Wizard guides you through every step
- **Uncluttered** - Clean TUI shows key info immediately
- **Secure by default** - SBOM, signing, verification built-in
- **Interoperable** - OCI artifacts, OSSF standards
- **Flexible** - Supports Argo/Flux, ORAS/ModelPack, Sigstore/Notary

## Recommended Starting Point

```bash
model-cli wizard
```

This single command guides users through the **complete workflow**:
1. Package model as OCI artifact (with SBOM + MOF)
2. Sign with Sigstore or Notary v2
3. Verify the signature
4. Deploy to Kubernetes with Argo or Flux

## Command Overview

| Command | Purpose | When to Use |
|---------|---------|-------------|
| `wizard` | Full guided workflow | Start here for the complete experience |
| `package` | Package model as OCI artifact | When you only need packaging |
| `sign` | Sign artifact | After packaging, before deployment |
| `verify` | Verify signature | Before deploying to production |
| `deploy` | Deploy to Kubernetes | After signing/verification |
| `validate` | Validate metadata contract | Before pushing to registry |
| `validate-gitops` | Pre-flight validation for GitOps | Before GitOps deployment |
| `enforce` | Enforce metadata contract | At registry/admission proxy level |
| `search` | Cross-reference assets | Discover assets in registry |
| `map` | Map relationships | Define model→skill→pipeline relationships |

## The Secure Supply Chain Workflow

### 1. Package
```bash
model-cli package
```
- Creates OCI artifact from model files
- Generates SBOM (Software Bill of Materials) with Syft
- Applies MOF (Model Openness Framework) classification
- Optionally includes RAG context
- Pushes to registry (ORAS or ModelPack)

**Standards:** OCI Image Spec, MOF

### 2. Sign
```bash
model-cli sign
```
- Signs OCI artifact with Sigstore (cosign) or Notary v2 (notation)
- Creates cryptographic provenance
- Aligns with OpenSSF Model Signing Specification

**Standards:** OSSF Model Signing Spec

### 3. Verify
```bash
model-cli verify
```
- Validates artifact signature
- Confirms artifact hasn't been tampered with
- Ensures provenance chain is intact

**Standards:** OSSF Model Signing Spec

### 4. Deploy
```bash
model-cli deploy
```
- Deploys to Kubernetes with Argo or Flux
- Validates GitOps configuration
- Supports both UI-based (Argo) and agent-based (Flux) approaches

**Standards:** GitOps principles

## Key Features

### Pluggable Architecture
All tools are pluggable via interfaces:
- **GitOps:** Argo, Flux
- **Registry:** ORAS, ModelPack
- **Signing:** Sigstore (cosign), Notary v2 (notation)
- **SBOM:** Syft

### Interactive TUI (huh + lipgloss + bubbletea)
- Clean, color-coded interface
- Step-by-step guidance with **context side panels** (Shopware CLI style)
- Clear success/warning indicators
- Minimal clutter - key info only
- Progress bars with visual indicators
- Organized information in panel sections

### Configuration Management
- Saves preferences to `~/.model-cli.yaml`
- Remembers tool choices (registry, gitops, signer)
- Can be overridden via flags

### Standards Compliance
- **OCI Spec** - For artifact format
- **OSSF Model Signing Spec** - For signing/verification
- **MOF (Model Openness Framework)** - For classification
- **SBOM** - For supply chain transparency

## Usage Patterns

### For Users: The Wizard Experience
```bash
# Start the full guided journey
model-cli wizard

# Skip signing if you just want to package
model-cli wizard --skip-signing

# Skip deployment if you don't have K8s yet
model-cli wizard --skip-deploy
```

### For Automation: Individual Commands
```bash
# Package only
model-cli package --model phi-4-mini --registry oras

# Sign only
model-cli sign --artifact my-model:latest --signer sigstore

# Verify only
model-cli verify --artifact my-model:latest

# Deploy only
model-cli deploy --gitops argo --registry oras
```

### For CI/CD: Non-Interactive Mode
```bash
# All commands support --help for flags
export MODEL_CLI_NO_INTERACTIVE=true
model-cli package --model phi-4-mini --registry oras --output my-model:v1
model-cli sign --artifact my-model:v1 --signer sigstore --key $SIGNING_KEY
model-cli verify --artifact my-model:v1
model-cli deploy --gitops argo --repo $REPO_URL --path ./manifests
```

## The "10-Minute Idea-to-Inference" Thread

This CLI enables the vision from your user journey matrix:

```
Phase 1: Author & Package
  ↓
Phase 2: Enterprise OCI Registry Integration (SBOM + MOF + Signing)
  ↓
Phase 3: Kubernetes Production Cluster (The Outer Loop)
  ↓
Phase 4: GitOps Promotion (Argo/Flux)
  ↓
Phase 5: Runtime Execution (Kubernetes)
```

All in under 10 minutes with `model-cli wizard`.

## Best Practices

### Always Package Before Deploying
```bash
model-cli package
model-cli deploy
```

### Always Sign Before Production
```bash
model-cli package
model-cli sign
model-cli verify
model-cli deploy
```

### Use the Wizard for First-Time Users
```bash
model-cli wizard
```

### Use Individual Commands for Automation
```bash
# In CI/CD pipelines
model-cli package --model $MODEL --registry oras
model-cli sign --artifact $ARTIFACT --signer sigstore
```

## Troubleshooting

### Tool Not Installed
The CLI checks if required tools are installed and provides installation instructions:
```
Flux not installed or not in PATH. Install with: brew install fluxcd/tap/flux
```

### Configuration Issues
Reset config by deleting `~/.model-cli.yaml` or use flags to override.

### Signature Verification Failed
This means the artifact was tampered with or signed with an untrusted key. Do not deploy.

## Architecture

### Directory Structure
```
cmd/
  deploy.go    # Deploy command
  package.go   # Package command
  sign.go      # Sign command
  verify.go    # Verify command
  wizard.go    # Unified TUI workflow
  root.go      # CLI root

internal/workflow/
  deploy.go    # Deploy workflow
  package.go   # Package workflow
  providers.go # Pluggable provider interfaces + implementations

config/
  config.go    # Configuration management

skills/model-cli/
  SKILL.md     # This file - AI agent guidance
```

### Provider Pattern
All external tool integrations follow the same pattern:
1. Define interface in `providers.go`
2. Implement concrete provider
3. Register in factory function (`Get*Provider`)
4. Use via dependency injection in workflows

This makes it easy to add new tool integrations without changing existing code.

## When NOT to Use Model CLI

- **Direct tool access needed** - Use `argocd`, `flux`, `oras`, `cosign` directly
- **Custom workflows** - If your workflow doesn't fit the standard pattern
- **Non-OCI formats** - If you're not using OCI artifacts
- **Non-Kubernetes targets** - Currently focused on K8s deployment

## Integration with Other Standards

### OpenSSF Model Signing Spec
- Full support for Sigstore (cosign) signing
- Full support for Notary v2 (notation) signing
- SBOM generation for transparency
- Provenance tracking

### OCI Spec
- OCI artifacts for model packaging
- Standard manifest format
- Registry interoperability

### Model Openness Framework (MOF)

### Enterprise OCI Registry Integration
Model CLI implements a complete Enterprise OCI Registry Integration workflow:

#### Phase 2: Enterprise OCI Registry (Stories #58-62)

**Story #58: Push Unified OCI Manifest to Registry**
- Push unified OCI manifests with standardized metadata to registries
- Attach CNCF AI annotations to manifests
- Generate and freeze immutable provenance/attestation metadata

**Story #59: Enforce Standardized Metadata Contract at Manifest Level**
- Validate required fields (model.framework, skill.pipeline_ref)
- Reject pushes with invalid or missing metadata
- Use admission webhooks (Kubernetes-style) or registry middleware

**Story #60: Map Complex Relationships in Manifest**
- Embed relationship maps (model → skill → pipeline) in OCI manifest
- Support ai.relationships field in manifest annotations
- Enable registry to parse dependencies without unpacking artifacts

**Story #61: Cross-Reference Assets in Registry**
- Query registry to discover and cross-reference AI assets
- Support filtering by metadata (e.g., ?filter=ai.model.type=llm)
- CLI search commands (e.g., model-cli search --model my-llm --type pipeline)
- Return structured results (list of pipelines + their skills/models)

**Story #62: Validate Pushes Against Metadata Contract**
- Validate required fields (model.type, skill.dependencies)
- Return clear error messages for missing/invalid metadata
- Support dry-run validation (model-cli validate --manifest my-manifest.json)
- Use JSON Schema for contract validation

#### Phase 3: Kubernetes Production Cluster (The Outer Loop)

**Story #63: Pass Trust Profile to GitOps**
- Attach Trust Profile annotations to OCI manifests for GitOps admission
- Pass artifact reference with annotations to GitOps tools (Argo CD, Flux)
- Policy enforcement delegated to external tools (Sigstore Policy Controller, OPA/Gatekeeper, Kyverno)
- Annotations: security.signing.framework, security.sbom.format, security.provenance.type, interop.profile.version, artifact.type

**Story #64: Pass Infrastructure Requirements to GitOps**
- Add infrastructure requirement annotations to OCI manifests during push
- Annotations: runtime, accelerator, accelerator.cuda.min, resource.memory.min
- Policy engines verify artifact requirements match destination environment
- Supports GPU/CPU requirements, CUDA version matching, memory requirements
- Allows deployment to air-gapped or hybrid-cloud environments with safety policies

**Story #65: GitOps Pre-Sync Validation Hook**
- Provide `validate-gitops` command for pre-flight validation
- Fetches artifact manifest from registry and validates annotations
- Validates Trust Profile annotations (Story #63)
- Validates Infrastructure Requirement annotations (Story #64)
- Returns pass/fail exit code for CI/CD integration
- Supports quiet mode and JSON output for automation

**Story #66: Support Air-Gapped and Hybrid-Cloud Safety Policies**
- Extend `validate-gitops` with environment-specific validation
- Air-gapped: validate packaging format and SBOM presence
- Hybrid-cloud: validate data residency and network access requirements
- Support `--env` and `--region` flags for environment targeting
- Provide clear guidance for air-gapped dependency pre-loading

**Story #67: Implement Real Admission Evaluation**
- Enhance `admit` command to fetch real manifests from registries
- Validate Trust Profile annotations (Story #63)
- Validate Infrastructure Requirement annotations (Story #64)
- Validate Environment Safety Policies (Story #66)
- Support `--registry`, `--json-output` flags for CI/CD integration
- Provide comprehensive admission decision with pass/fail status


#### Metadata Contract Validation
```bash
# Dry-run validation before push
model-cli validate --manifest my-manifest.json

# Strict JSON Schema validation
model-cli validate --manifest my-manifest.json --json-schema --strict

# Enforce at registry level (admission webhook)
model-cli enforce --manifest my-manifest.json
model-cli enforce --webhook --port 8443
```

#### Registry Search and Discovery
```bash
# Search for all models in a registry
model-cli search --destination ghcr.io/my-org --type model

# Find all pipelines using a specific model
model-cli search --uses-model model:sha256:abc123

# Filter by metadata
model-cli search --metadata ai.model.type=llm
```

#### Relationship Mapping
```bash
# Create relationship graph in manifest
model-cli map --model my-model --skill my-skill --pipeline my-pipeline

# Validate relationships
model-cli validate --manifest my-manifest.json --check-relationships
```
- Class I: Open weights, open training data, open code
- Class II: Open weights, closed training data or code
- Class III: Closed weights
- Classification stored in artifact metadata

## Example: Full Production Workflow

```bash
# Developer laptop - Phase 1 & 2
model-cli wizard
# → Packages model with SBOM + MOF
# → Signs with Sigstore
# → Verifies signature
# → User selects "No K8s" (local packaging only)

# Push to registry manually
oras push my-registry/my-model:v1 .

# CI/CD pipeline - Phase 3
# (ORAS push happens automatically in production)

# Production cluster - Phase 4 & 5
model-cli deploy --gitops argo \
  --repo https://github.com/org/model-manifests \
  --path ./production

# Verification in deployment pipeline
model-cli verify --artifact my-registry/my-model:v1
model-cli deploy --gitops argo ...
```

## Success Metrics

Users should be able to:
- [x] Package a model in under 2 minutes
- [x] Sign and verify in under 1 minute
- [x] Deploy to K8s in under 5 minutes
- [x] Understand exactly what's happening at each step
- [x] Feel confident about supply chain security
- [x] Have fun using the CLI

## Contributing

When working on this repository:
- Follow the pluggable provider pattern for new tools
- Keep TUI clean and uncluttered
- Add clear error messages with installation instructions
- Document new features in this SKILL.md
- Ensure all workflows align with relevant standards (OCI, OSSF, MOF)

## Related Resources

- [OCI Spec](https://specs.opencontainers.org/image-spec/)
- [OSSF Model Signing Spec](https://github.com/ossf/model-signing-spec)
- [Model Openness Framework](https://github.com/Adopt-MOF/MOF)
- [Sigstore](https://www.sigstore.dev/)
- [Notary v2](https://github.com/notaryproject/notaryproject)
- [ORAS](https://oras.land/)
- [ModelPack](https://modelpack.ai/)
- [Argo CD](https://argo-cd.readthedocs.io/)
- [Flux CD](https://fluxcd.io/)
