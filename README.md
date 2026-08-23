# Model CLI

**Your tour guide through the secure ML model deployment journey.**

Model CLI makes it easy to package, sign, verify, and deploy ML models with a clean, guided TUI. It follows a simple principle: **orchestrate the workflow, don't duplicate the tools.**

## Quick Test Drive (5 minutes)

You can test the entire CLI with just a text file - no real model, registry, or GPU required:

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

**See [Trial Run Guide](docs/trial-run.md) for a complete walkthrough with explanations.**

**New users:** See [Quick Start](docs/quick-start.md) for detailed getting started instructions.

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

**Model CLI only requires Go.** All other tools are optional and checked at runtime with clear installation instructions.

---

## Quick Examples

### Full Guided Journey
```bash
./model-cli wizard
```

### Skip Steps
```bash
# Just package, skip signing and deployment
./model-cli wizard --skip-signing --skip-deploy

# Package and sign, but skip deployment
./model-cli wizard --skip-deploy
```

### Individual Commands
```bash
# Package
./model-cli package --model phi-4-mini --model-path ./models --registry oras

# Local compliance check
./model-cli check --model-path ./models

# Sign
./model-cli sign --artifact my-model:v1

# Validate metadata contract
./model-cli validate --manifest my-manifest.json

# Validate for GitOps deployment
./model-cli validate-gitops --artifact my-model:v1

# Evaluate artifact for admission
./model-cli admit --artifact my-model:v1

# Validate node hardware requirements
./model-cli validate-nodes --artifact my-model:v1

# Validate runtime availability
./model-cli validate-runtime --artifact my-model:v1

# Package with node requirements
./model-cli package --gpu-type nvidia-h100 --vram-min 80GiB --gpu-topology 8xH100

# Package with runtime requirements
./model-cli package --runtime-type vllm --layer-dedup true

# Package agentic skills (agentskills.io format)
./model-cli package --skill --skill-refs "skill1:sha256:abc,skill2:sha256:def"

# Map relationships between assets
./model-cli map --model my-model --skill my-skill --pipeline my-pipeline
```

---

## Understanding the "Warnings"

You may see messages like:

```
Warning: SBOM generation failed: syft not installed. Install with: brew install anchore/syft/syft
```

**This is not a failure - it's a feature!** The CLI:

1. Checks if required tools are available
2. Tells you exactly which tool is missing
3. Gives you the exact command to install it
4. Continues with the workflow anyway

This is the "orchestrate and hand off" design. The CLI doesn't crash when tools are missing - it guides you to install them.

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

## Documentation

- [Trial Run Guide](docs/trial-run.md) - 5-minute test drive with explanations
- [Quick Start](docs/quick-start.md) - Detailed getting started instructions
- [Architecture](docs/architecture.md) - Understand how it works
- [Enterprise OCI Registry](docs/enterprise-oci-registry.md) - Phase 2: Registry integration
- [GitOps Admission & Policy Enforcement](docs/gitops-admission.md) - Phase 3: Kubernetes production cluster
- [TUI Guide](docs/tui.md) - Learn about the terminal interface
- [Resources](docs/resources.md) - Standards and tools

## Skills

AI agent guidance: [skills/model-cli/SKILL.md](skills/model-cli/SKILL.md)

## Standards

Model CLI aligns with:
- [OCI Specification](https://specs.opencontainers.org/image-spec/) - Artifact format
- [OCI Distribution Spec](https://github.com/opencontainers/distribution-spec) - Registry operations
- [OSSF Model Signing Spec](https://github.com/ossf/model-signing-spec) - Signing standards
- [Model Openness Framework](https://github.com/Adopt-MOF/MOF) - Classification
- [JSON Schema](https://json-schema.org/) - Metadata contract validation
- GitOps principles - Deployment patterns

## Contributing

1. Add interface in `internal/workflow/providers.go`
2. Implement concrete provider
3. Register in factory function (`Get*Provider`)
4. Add CLI command
5. Support both interactive and non-interactive modes

## License

Apache License 2.0
