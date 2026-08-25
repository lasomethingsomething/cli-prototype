---
name: model-cli
description: Use Model CLI to package, sign, verify, and deploy ML models with a secure supply chain workflow. Aligns with OCI specs, OSSF Model Signing Spec, and Model Openness Framework.
---

# Model CLI

Model CLI is your **tour guide** through the secure ML model deployment journey. It provides a clean, wizard-style TUI with context panels that makes complex workflows simple, transparent, and secure.

Use this skill when:
- Packaging ML models as OCI artifacts
- Packaging agentic skills using the agentskills.io standard format
- Generating SBOMs for supply chain transparency
- Signing and verifying model artifacts (Sigstore, Notary v2)
- Deploying models to Kubernetes with GitOps (Argo, Flux)
- Classifying models with MOF (Model Openness Framework)
- Following OpenSSF Model Signing Specification best practices
- Mapping complex relationships between assets (model → skill → pipeline)
- Validating metadata contracts at manifest level
- Cross-referencing AI assets in registries

## Core Philosophy

**Simple, Clear, Fun.** The CLI is designed to be:
- **Easy to use** - Wizard guides you through every step
- **Uncluttered** - Clean TUI shows key info immediately
- **Secure by default** - SBOM, signing, verification built-in
- **Interoperable** - OCI artifacts, OSSF standards
- **Flexible** - Supports Argo/Flux, ORAS/ModelPack, Sigstore/Notary, vLLM/KServe
- **Orchestrates, doesn't duplicate** - Hands off to external tools, doesn't reimplement them

## Recommended Starting Point

```bash
model-cli wizard
```

This single command guides users through the **complete workflow** for Phase 1 (Developer Laptop) and Phase 2 (Enterprise OCI Registry):
1. Package model as OCI artifact (with SBOM + MOF)
2. Attach standardized annotations (CNCF AI Interoperability Profile)
3. Sign with Sigstore or Notary v2
4. Verify the signature
5. Optionally deploy to Kubernetes with Argo or Flux

## Command Overview

| Command | Purpose | When to Use |
|---------|---------|-------------|
| `wizard` | Full guided workflow | Start here for the complete experience |
| `package` | Package model as OCI artifact | When you only need packaging |
| `sign` | Sign artifact | After packaging, before deployment |
| `verify` | Verify signature | Before deploying to production |
| `deploy` | Deploy to Kubernetes | After signing/verification |
| `check` | Local compliance check | Before pushing to registry |
| `validate` | Validate metadata contract | Before pushing to registry |
| `enforce` | Enforce metadata contract | At registry/admission proxy level |
| `validate-gitops` | Pre-flight validation for GitOps | Before GitOps deployment |
| `validate-nodes` | Validate node hardware requirements | Before scheduling to cluster |
| `validate-runtime` | Validate runtime operator availability | Before deployment |
| `admit` | Evaluate artifact for admission | For GitOps admission control |
| `search` | Cross-reference assets | Discover assets in registry |
| `map` | Map relationships | Define model→skill→pipeline relationships |

## The Secure Supply Chain Workflow

### Phase 1: Developer Laptop (The Inner Loop)

#### 1. Package
```bash
model-cli package
```
- Creates OCI artifact from model files
- Generates SBOM (Software Bill of Materials) with Syft, Trivy, or cdxgen
- Applies MOF (Model Openness Framework) classification (Class I, II, or III)
- Attaches CNCF AI Interoperability Profile annotations
- Optionally includes RAG context
- Optionally packages as agentic skill (agentskills.io format)
- Optionally embeds relationship maps (model→skill→pipeline)
- Pushes to registry (ORAS or ModelPack)

**Standards:** OCI Image Spec, MOF, CNCF AI Interoperability Profile

#### 2. Local Compliance Check
```bash
model-cli check
```
- Validates local artifact against metadata contract
- Checks required annotations present
- Verifies SBOM presence
- Confirms MOF classification applied
- Blocks with clear message when required pieces are missing

#### 3. Sign
```bash
model-cli sign
```
- Signs OCI artifact with Sigstore (cosign) or Notary v2 (notation)
- Creates cryptographic provenance
- Aligns with OpenSSF Model Signing Specification

**Standards:** OSSF Model Signing Spec

#### 4. Verify
```bash
model-cli verify
```
- Validates artifact signature
- Confirms artifact hasn't been tampered with
- Ensures provenance chain is intact

**Standards:** OSSF Model Signing Spec

### Phase 2: Enterprise OCI Registry

#### 5. Push with Metadata Contract
```bash
# Push with standardized metadata
model-cli package --push --registry oras
```
- Pushes unified OCI manifests with standardized metadata to registries
- Attaches CNCF AI annotations to manifests
- Generates and freezes immutable provenance/attestation metadata
- Validates against metadata contract before push

#### 6. Enforce Metadata Contract at Manifest Level
```bash
model-cli enforce --manifest my-manifest.json
# Or as admission webhook
model-cli enforce --webhook --port 8443
```
- Validates required fields (model.framework, skill.pipeline_ref)
- Rejects pushes with invalid or missing metadata
- Can run as admission webhook (Kubernetes-style) or registry middleware

#### 7. Map Complex Relationships in Manifest
```bash
model-cli map --model my-model --skill my-skill --pipeline my-pipeline
```
- Embeds relationship maps (model → skill → pipeline) in OCI manifest
- Supports ai.relationships field in manifest annotations
- Enables registry to parse dependencies without unpacking artifacts

#### 8. Cross-Reference Assets in Registry
```bash
model-cli search --destination ghcr.io/my-org --type model
model-cli search --uses-model model:sha256:abc123
model-cli search --metadata ai.model.type=llm
```
- Queries registry to discover and cross-reference AI assets
- Supports filtering by metadata
- Returns structured results (list of pipelines + their skills/models)

#### 9. Validate Pushes Against Metadata Contract
```bash
model-cli validate --manifest my-manifest.json
model-cli validate --manifest my-manifest.json --json-schema --strict
```
- Validates required fields (model.type, skill.dependencies)
- Returns clear error messages for missing/invalid metadata
- Supports dry-run validation
- Uses JSON Schema for contract validation

### Phase 3: Kubernetes Production Cluster (The Outer Loop)

#### 10. Pass Trust Profile to GitOps
```bash
model-cli validate-gitops --artifact my-model:v1
```
- Attaches Trust Profile annotations to OCI manifests for GitOps admission
- Passes artifact reference with annotations to GitOps tools (Argo CD, Flux)
- Policy enforcement delegated to external tools (Sigstore Policy Controller, OPA/Gatekeeper, Kyverno)
- Validates Trust Profile annotations before deployment

#### 11. Pass Infrastructure Requirements to GitOps
```bash
model-cli validate-gitops --artifact my-model:v1
```
- Validates infrastructure requirement annotations
- Annotations: runtime, accelerator, accelerator.cuda.min, resource.memory.min
- Policy engines verify artifact requirements match destination environment
- Supports GPU/CPU requirements, CUDA version matching, memory requirements
- Allows deployment to air-gapped or hybrid-cloud environments with safety policies

#### 12. GitOps Pre-Sync Validation Hook
```bash
model-cli validate-gitops --artifact my-model:v1 --quiet --json-output
```
- Pre-flight validation for GitOps deployment
- Fetches artifact manifest from registry and validates annotations
- Validates Trust Profile annotations
- Validates Infrastructure Requirement annotations
- Returns pass/fail exit code for CI/CD integration
- Supports quiet mode and JSON output for automation

#### 13. Implement Real Admission Evaluation
```bash
model-cli admit --artifact my-model:v1 --registry ghcr.io
model-cli admit --artifact my-model:v1 --json-output
```
- Enhances admit command to fetch real manifests from registries
- Validates Trust Profile annotations
- Validates Infrastructure Requirement annotations
- Validates Environment Safety Policies
- Supports --registry, --json-output flags for CI/CD integration
- Provides comprehensive admission decision with pass/fail status

#### 14. Support Air-Gapped and Hybrid-Cloud Safety Policies
```bash
model-cli validate-gitops --artifact my-model:v1 --env air-gapped --region us-east-1
```
- Extends validate-gitops with environment-specific validation
- Air-gapped: validates packaging format and SBOM presence
- Hybrid-cloud: validates data residency and network access requirements
- Supports --env and --region flags for environment targeting
- Provides clear guidance for air-gapped dependency pre-loading

#### 15. Infrastructure & Resource Orchestration
```bash
./model-cli package --gpu-type nvidia-h100 --vram-min 80GiB --gpu-topology 8xH100
./model-cli validate-nodes --artifact my-model:v1 --namespace production
```
- Attaches node requirement annotations (GPU type, vRAM minimum, GPU topology) to OCI manifests during packaging
- Validates cluster nodes against artifact requirements
- Hands off to Kubernetes scheduler (does NOT implement scheduling)
- Annotations: `ai.node.gpu.type`, `ai.node.vram.min`, `ai.node.gpu.topology`

#### 16. Runtime Execution & Optimization
```bash
./model-cli package --runtime-type vllm --layer-dedup true --dlc-endpoint https://skills.example.com
./model-cli validate-runtime --artifact my-model:v1 --namespace production
```
- Attaches runtime-specific annotations (runtime type, layer deduplication) to OCI manifests during packaging
- Supports Reference Skill DLC with endpoint and skill reference annotations
- Validates runtime operator availability in cluster
- Hands off to serving runtimes (KServe, vLLM) - does NOT implement model serving
- Annotations: `ai.runtime.type`, `ai.runtime.optimization.layer-dedup`, `ai.skill.dlc-endpoint`, `ai.skill.references`

## Key Features

### Pluggable Architecture
All tools are pluggable via interfaces:
- **GitOps:** Argo, Flux
- **Registry:** ORAS, ModelPack
- **Signing:** Sigstore (cosign), Notary v2 (notation)
- **SBOM:** Syft, Trivy, cdxgen
- **Runtime:** vLLM, KServe

### Interactive TUI (huh + lipgloss + bubbletea)
- Clean, color-coded interface
- Step-by-step guidance with context side panels
- Clear success/warning indicators
- Minimal clutter - key info only
- Progress indicators
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
- **CNCF AI Interoperability Profile** - For standardized annotations
- **JSON Schema** - For metadata contract validation
- **GitOps principles** - For deployment patterns

## Usage Patterns

### For Users: The Wizard Experience
```bash
# Start the full guided journey
model-cli wizard

# Skip signing if you just want to package
model-cli wizard --skip-signing

# Skip deployment if you don't have K8s yet
model-cli wizard --skip-deploy

# Skip both
model-cli wizard --skip-signing --skip-deploy
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
# All commands support flags for automation
export MODEL_CLI_NO_INTERACTIVE=true
model-cli package --model phi-4-mini --registry oras --artifact my-model:v1
model-cli sign --artifact my-model:v1 --signer sigstore
model-cli verify --artifact my-model:v1
model-cli deploy --gitops argo --repo $REPO_URL --path ./manifests
```

## The "10-Minute Idea-to-Inference" Thread

This CLI enables the vision from your user journey matrix:

```
Phase 1: Author & Package (Inner Loop - Developer Laptop)
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

## Understanding the "Warnings"

Model CLI checks if required tools are installed and provides clear installation instructions:

```
Flux not installed or not in PATH. Install with: brew install fluxcd/tap/flux
```

This is **not a failure** - it's the "orchestrate and hand off" design:
1. CLI checks for required tools
2. Tells you exactly what's missing
3. Gives you the exact command to install it
4. Continues with the workflow anyway (when possible)

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
- Class I: Open weights, open training data, open code
- Class II: Open weights, closed training data or code
- Class III: Closed weights
- Classification stored in artifact metadata

### CNCF AI Interoperability Profile
- Standardized annotations for AI/ML workloads
- Enables policy engines to validate without downloading models
- Facilitates GitOps routing and registry indexing

## Architecture

### Directory Structure
```
cmd/                      # One cobra command per file (package, sign, verify, deploy,
                          # wizard, check, validate*, admit, enforce, harden, search, map, ...)
  root.go                 # CLI root + config loading

internal/workflow/        # Orchestration logic and provider interfaces
  package.go              # Package workflow
  deploy.go, harden.go, check.go, search.go, ...
  registry.go             # RegistryProvider: ORAS, ModelPack
  signing.go              # SigningProvider: Sigstore, Notary v2
  sbom.go                 # SBOMGenerator: Syft, Trivy, cdxgen
  gitops.go               # GitOpsProvider: Argo CD, Flux
  runtime.go              # RuntimeProvider: vLLM, KServe
  mof.go                  # MOF classifier
  annotations.go          # CNCF AI Interoperability Profile annotation keys
  manifest.go             # OCI manifest read/write
  metadata_contract.go    # JSON-Schema metadata contract

internal/tui/             # bubbletea / huh / lipgloss models and context panels
config/                   # ~/.model-cli.yaml handling
skills/model-cli/SKILL.md # This file - AI agent guidance
```

### Provider Pattern
All external tool integrations follow the same pattern:
1. Define the interface in the matching `internal/workflow/*.go` file (see Directory Structure)
2. Implement concrete provider
3. Register in factory function (`Get*Provider`)
4. Use via dependency injection in workflows

This makes it easy to add new tool integrations without changing existing code.

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
- Ensure all workflows align with relevant standards (OCI, OSSF, MOF, CNCF AI Interoperability Profile)

## Related Resources

- [OCI Spec](https://specs.opencontainers.org/image-spec/)
- [OCI Distribution Spec](https://github.com/opencontainers/distribution-spec)
- [OSSF Model Signing Spec](https://github.com/ossf/model-signing-spec)
- [Model Openness Framework](https://github.com/Adopt-MOF/MOF)
- [CNCF AI Interoperability Profile](https://github.com/ai-interop/ai-interop)
- [Sigstore](https://www.sigstore.dev/)
- [Notary v2](https://github.com/notaryproject/notaryproject)
- [ORAS](https://oras.land/)
- [ModelPack](https://modelpack.ai/)
- [Argo CD](https://argo-cd.readthedocs.io/)
- [Flux CD](https://fluxcd.io/)
- [vLLM](https://github.com/vllm-project/vllm)
- [KServe](https://kserve.github.io/website/)
