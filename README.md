# Model CLI

**Your tour guide through the secure ML model deployment journey.**

Model CLI makes it easy to package, sign, verify, and deploy ML models with a clean, guided TUI. It follows a simple principle: **orchestrate the workflow, don't duplicate the tools.**

## Quick Test Drive (5 minutes)

Try the CLI immediately with just a text file - no real model, registry, or GPU required:

```bash
# 1. Create a dummy model
mkdir -p ~/test-model
echo "test" > ~/test-model/model.txt

# 2. Build the CLI
cd cli-prototype
go build -o model-cli .

# 3. Package it
./model-cli package \
  --model test-model \
  --model-path ~/test-model \
  --artifact test:v1 \
  --registry oras \
  --registry-url ""
```

The CLI will ask you a few questions (runtime, accelerator, etc.). **These are just metadata - you don't need the actual hardware or software installed.**

After answering, you'll see:
- OCI manifest created with standardized annotations
- SBOM generation attempted (warning if syft not installed)
- MOF classification applied
- Provenance attestation generated

---

## Key Concepts

### OCI Artifacts

Model CLI packages models as **OCI artifacts** - the same standard format used by Docker, Kubernetes, and container registries:
- Store models in any OCI-compliant registry (Docker Hub, GHCR, AWS ECR, etc.)
- Use standard OCI tools to push, pull, and manage models
- Enable GitOps tools to discover and deploy models without special integration

### Annotations

Annotations are metadata attached to OCI manifests. Model CLI automatically injects:

- **Profile**: Version, artifact type
- **Security**: Signing framework, SBOM format, provenance type
- **MOF**: Openness class (I, II, or III), version, components
- **Runtime**: Serving runtime (vLLM, KServe), accelerator, memory
- **Infrastructure**: GPU type, vRAM, topology
- **Relationships**: Model→Skill→Pipeline dependency mapping

These annotations enable:
- Policy engines to validate models without downloading them
- GitOps tools to route models to appropriate clusters
- Registry tools to index and search models
- Deployment tools to match models to hardware

### Metadata Contract

The **Standardized Metadata Contract** ensures all models have required annotations for:
- **Security**: Signature verification, SBOM presence
- **Compliance**: MOF classification, license info
- **Deployment**: Runtime requirements, hardware needs
- **Discovery**: Model type, relationships, dependencies

---

## How It Works

Model CLI **orchestrates** your ML deployment workflow. It collects your preferences and model information, attaches standardized metadata (annotations) to OCI manifests, validates requirements, then hands off to the right external tool for each job.

### Tool Integration

| Task | Model CLI Role | External Tool |
|------|----------------|---------------|
| Package model | Collects model info, creates manifest, injects annotations | ORAS or ModelPack |
| Generate SBOM | Sets up SBOM config, attaches to manifest | Syft, Trivy, or cdxgen |
| Sign artifact | Sets up signing config | Cosign (Sigstore) or Notation (Notary v2) |
| Verify signature | Checks manifest | Cosign or Notation |
| Deploy to K8s | Collects GitOps preferences, validates | Argo CD or Flux |
| Validate nodes | Collects requirements, checks cluster | kubectl |

**Model CLI only requires Go.** All other tools are optional and checked at runtime with clear installation instructions. Model CLI aligns with: [OCI Specification](https://specs.opencontainers.org/image-spec/), [OCI Distribution Spec](https://github.com/opencontainers/distribution-spec), [OSSF Model Signing Spec](https://github.com/ossf/model-signing-spec), [Model Openness Framework](https://github.com/Adopt-MOF/MOF), [JSON Schema](https://json-schema.org/), and GitOps principles.

AI agent guidance: [skills/model-cli/SKILL.md](skills/model-cli/SKILL.md)

---

## Documentation

- [Quick Start](docs/quick-start.md) - Phase 1: Developer Laptop
- [Architecture](docs/architecture.md) - Understand how it works
- [Enterprise OCI Registry](docs/enterprise-oci-registry.md) - Phase 2: Registry integration
- [GitOps Admission & Policy Enforcement](docs/gitops-admission.md) - Phase 3: Kubernetes production cluster
- [TUI Guide](docs/tui.md) - Learn about the terminal interface
- [Resources](docs/resources.md) - Standards and tools
