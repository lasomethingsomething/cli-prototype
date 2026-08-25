# Model CLI

**Your tour guide through the secure ML model deployment journey.**

Model CLI packages, signs, verifies, and deploys ML models as OCI artifacts through a guided TUI. It follows one principle: **orchestrate the workflow, don't duplicate the tools.** It collects your intent, attaches standardized metadata to OCI manifests, validates it, and hands the actual work to ORAS, Cosign, Syft, Argo CD, and friends.

> **Status: prototype.** The workflow, annotation model, and validation commands are real. Some tool integrations are still placeholders — see [Known limitations](#known-limitations) before relying on any single step.

## Quick Test Drive (5 minutes)

Try the CLI with a plain text file. No real model, registry, or GPU required.

```bash
# 1. Create a dummy model
mkdir -p ~/test-model
echo "test" > ~/test-model/model.txt

# 2. Build the CLI (requires Go 1.23+)
git clone https://github.com/lasomethingsomething/cli-prototype.git
cd cli-prototype
go build -o model-cli .

# 3. Install ORAS - `package` checks for it up front
brew install oras   # or see https://oras.land

# 4. Package it (empty --registry-url = keep it local, push nothing)
./model-cli package \
  --model test-model \
  --model-path ~/test-model \
  --artifact test:v1 \
  --registry oras \
  --registry-url ""
```

The CLI asks a few questions (runtime, accelerator, MOF class, ...). **These are metadata only** - you don't need the hardware or software installed. Press Enter to accept defaults.

You'll end up with:

- `~/test-model/manifest.json` - an OCI manifest carrying CNCF AI Interoperability Profile annotations
- `~/test-model/attestation.json` - a SLSA provenance attestation
- an SBOM at `~/test-model/sbom.spdx-json` if `syft` is installed (otherwise a warning, not a failure)

A detailed, annotated walkthrough of this run is in [docs/trial-run-report.md](docs/trial-run-report.md).

Any value you pass as a flag is not prompted for again (`--runtime vllm --accelerator cpu --mof-class II ...`). Empty values and the RAG confirmation still prompt, so `package` cannot yet run without a terminal - see [Known limitations](#known-limitations).

## Commands

Start with `model-cli wizard` for the guided end-to-end journey. Each step is also its own command:

| Phase | Command | What it does |
|-------|---------|--------------|
| Package | `package` | Package a model or agentic skill as an OCI artifact, inject annotations, generate SBOM + provenance |
| Package | `harden` | Apply local hardening (SBOM, MOF classification) to an existing artifact |
| Package | `check` | Local compliance check against the metadata contract before pushing |
| Sign | `sign` / `verify` | Sign and verify with Sigstore (cosign) or Notary v2 (notation) |
| Registry | `push` | Push an artifact to an OCI registry |
| Registry | `validate` / `enforce` | Validate a manifest against the metadata contract (JSON Schema); `enforce` can run as an admission webhook |
| Registry | `map` / `search` | Embed and query model → skill → pipeline relationships |
| Deploy | `validate-gitops` | Pre-sync validation of trust-profile and infrastructure annotations for Argo CD / Flux |
| Deploy | `admit` | Evaluate an artifact for GitOps admission (fetches the real manifest) |
| Deploy | `validate-nodes` / `validate-runtime` / `schedule` | Check cluster nodes and runtime operators against artifact requirements |
| Deploy | `deploy` / `serve` | Hand off to Argo CD / Flux and vLLM / KServe |

Every command takes flags and falls back to prompts for anything you leave out. Tool choices are remembered in `~/.model-cli.yaml`.

## Key Concepts

**OCI artifacts.** Models are packaged in the same format Docker, Kubernetes, and every container registry already understand, so standard tooling can push, pull, sign, and route them.

**Annotations.** Model CLI attaches metadata to the OCI manifest under the CNCF AI Interoperability Profile keys:

- Profile - `org.cncf.ai.interop.profile.version`, `org.cncf.ai.artifact.type`
- Security - signing framework, SBOM format, provenance type, packaging format
- MOF - openness class (I / II / III), version, components
- Runtime - serving runtime, accelerator, minimum CUDA and memory
- Infrastructure - `ai.node.gpu.type`, `ai.node.vram.min`, `ai.node.gpu.topology`
- Execution - `ai.runtime.type`, layer dedup, skill DLC endpoint, skill references

Policy engines, GitOps tools, and registries can act on these without downloading the model.

**Metadata contract.** `check`, `validate`, `enforce`, and `admit` all evaluate the same JSON-Schema-backed contract, so a model that passes locally passes at the registry and at admission.

## Tool Integration

| Task | Model CLI role | External tool |
|------|----------------|---------------|
| Package / push | Build manifest, inject annotations | ORAS (ModelPack planned) |
| SBOM | Configure format, attach result | Syft |
| Sign / verify | Configure signer, run it | Cosign (Sigstore) or Notation (Notary v2) |
| Provenance | Generate SLSA v1.0 attestation | built in |
| Deploy | Validate, then hand off | Argo CD or Flux |
| Cluster checks | Compare annotations with nodes | kubectl |

**Only Go is required to build.** Every other tool is optional, detected at runtime, and reported with an install command when missing.

Standards: [OCI Image Spec](https://specs.opencontainers.org/image-spec/), [OCI Distribution Spec](https://github.com/opencontainers/distribution-spec), [OSSF Model Signing Spec](https://github.com/ossf/model-signing-spec), [Model Openness Framework](https://github.com/Adopt-MOF/MOF), [SPDX](https://spdx.dev/), [SLSA](https://slsa.dev/), [JSON Schema](https://json-schema.org/).

## Known limitations

Honest list of what is scaffolding today:

- **ModelPack provider is not implemented** - tool detection works, but every registry operation returns a clear `not implemented` error. Use `--registry oras`.
- **The written `manifest.json` is minimal** - it has annotations and a config media type but no layers or digests, so it is not yet a manifest a registry would accept as-is. On push, ORAS builds the real manifest and the annotations are passed to it.
- **MOF auto-classification is informational** - the class you answer in the prompt (or `--mof-class`) is what lands in the manifest; the classifier's result is printed but not applied.
- **`package` always needs a TTY** - flags suppress their prompt only when non-empty, and the RAG confirm always asks, so it fails in CI with `huh: could not open a new TTY`. The `MODEL_CLI_NO_INTERACTIVE` variable mentioned in older docs is not implemented.
- **No published binaries yet**; build from source or tag a release.

## Development

```bash
go build ./...
go test ./...
gofmt -l .          # should print nothing
```

CI runs build and tests on every PR. Tagging `v*` builds binaries for Linux, macOS, and Windows and attaches them to a GitHub release (see `.github/workflows/release.yml`).

See [CONTRIBUTING.md](CONTRIBUTING.md) for the provider pattern used to add new tool integrations. AI agent guidance lives in [skills/model-cli/SKILL.md](skills/model-cli/SKILL.md).

## Documentation

- [Quick Start](docs/quick-start.md)
- [Trial Run Report](docs/trial-run-report.md)
- [Architecture](docs/architecture.md)
- [Enterprise OCI Registry](docs/enterprise-oci-registry.md)
- [GitOps Admission & Policy Enforcement](docs/gitops-admission.md)
- [TUI Guide](docs/tui.md)
- [Resources](docs/resources.md)
