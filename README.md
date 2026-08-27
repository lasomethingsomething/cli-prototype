# Model CLI

**Your tour guide through the secure ML model deployment journey.**

## What is this, in plain words?

A trained ML model is a folder of files. Getting that folder from your laptop onto a production server safely means a series of steps: bundle it up, label it, list what is inside, record where it came from, sign it so nobody can tamper with it, upload it, check it against your organization's rules, and hand it to the system that runs it. 

Specialized tools already exist for each step in the [CNCF ecosystem](https://www.cncf.io/) and beyond. Model CLI is the guide that walks you through the steps in order, asks the questions each step needs, and calls the right tool for each one.

The workflow in one sentence per phase:

1. **Package** - bundle the model folder into a standard container-style artifact and tag it with metadata.
2. **Harden** - generate an ingredients list (SBOM) and classify how open the model is (MOF), and record both in the packaged artifact.
3. **Sign** - stamp it cryptographically so tampering can be detected later, and record a proof of origin (provenance) for the finished artifact.
4. **Push** - upload it to a registry.
5. **Validate** - check the metadata against your rules locally, at the registry, and at the cluster door, using one shared engine so passing in one place means passing everywhere.
6. **Deploy** - hand it to Argo CD / Flux and a serving runtime such as vLLM or KServe.

If terms like *OCI*, *SBOM*, or *admission* are new to you, see the [Glossary](#glossary) below.

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

The CLI asks a few questions (runtime, accelerator, ...). **These are metadata only** - you don't need the hardware or software installed. Press Enter to accept defaults.

You'll end up with:

- `~/test-model/manifest.json` - an OCI manifest carrying CNCF AI Interoperability Profile annotations

```bash
# 5. Harden it: SBOM + MOF classification, recorded in the manifest from step 4
brew install syft   # or pick trivy / cdxgen with --sbom-tool
./model-cli harden --model test-model --model-path ~/test-model --artifact test:v1
```

This asks which SBOM tool to use (`syft` is recommended) and which MOF class to declare (default: detected from the files). It adds:

- `~/test-model/sbom.spdx-json` - the SBOM
- `~/test-model/mof.json` - the MOF metadata
- the SBOM format and MOF class/components as annotations in `~/test-model/manifest.json`

`harden` refuses to run before `package` (no `manifest.json` yet), so the order is always package, then harden.

Signing and the SLSA provenance attestation are a separate step that runs afterwards: `model-cli sign --artifact test:v1`.

A detailed, annotated walkthrough of this run is in [docs/trial-run-report.md](docs/trial-run-report.md).

Anything you pass as a flag is not prompted for again. In CI or scripts, prompts are disabled automatically when stdin is not a terminal (or with `--non-interactive` / `MODEL_CLI_NO_INTERACTIVE=1`): saved config and defaults fill the gaps, and a missing required value fails with a message naming the flag to pass:

```bash
./model-cli package --model test-model --model-path ~/test-model --artifact test:v1 \
  --registry oras --registry-url "" --accelerator cpu --non-interactive
```

## Commands

Start with `model-cli wizard` - the guided, end-to-end tour. Everything it does is also a standalone command.

**Build the artifact**

- `package` - package a model or agentic skill as an OCI artifact: manifest, annotations, push
- `harden` - the step after `package`: generate an SBOM (syft recommended, or trivy / cdxgen), classify the model (MOF), and record both in the packaged manifest
- `sign` - the separate step after packaging: sign with cosign (Sigstore, recommended) or Notary v2 (notation) and record a SLSA provenance attestation
- `verify` - check the signature and the provenance attestation
- `push` - push to an OCI registry, with the provenance attestation attached as a referrer

**Validate** - one command family, one report format (text, `--quiet`, or `--json-output` for CI)

- `validate local` - the model directory, before pushing
- `validate manifest` - a manifest against the metadata contract (JSON Schema)
- `validate gitops` - trust-profile and infrastructure annotations, as an Argo CD / Flux pre-sync hook
- `validate admission` - admission to a target environment (air-gapped, hybrid-cloud, ...)
- `validate nodes`, `validate runtime` - cluster nodes and runtime operators against the artifact's requirements

**Deploy and serve**

- `deploy` - hand off to Flux (recommended) or Argo CD
- `serve` - hand off to vLLM or KServe
- `schedule` - pick a node that satisfies the artifact's hardware requirements
- `enforce` - run the metadata contract as an admission webhook
- `map` - embed model → skill → pipeline relationships in a manifest
- `search` - list the AI artifacts in a registry, filtered by `--type`, `--metadata key=value` (exact annotation match) or relationship (`--uses-model`); `--output json` prints only the result document, for scripts

Every command takes flags and prompts for anything you leave out - never without a terminal, see the non-interactive note above. Tool choices are remembered in `~/.model-cli.yaml`. The wizard is the one command that is prompt-only.

No specific tool is required: each category (registry, signing, SBOM, GitOps) offers several tools, one of them marked *recommended* and used as the default. The tools within a category are mutually exclusive - one per artifact. The lists live in `internal/workflow/options.go` and drive every prompt and `--help` text.

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

**Metadata contract.** Every `validate` target and `enforce` evaluate the same annotations with one shared engine and report format (text, `--quiet`, or `--json-output`), so a model that passes locally passes at the registry and at admission. The old top-level spellings (`check`, `validate-gitops`, `admit`, `validate-nodes`, `validate-runtime`) still work as hidden, deprecated aliases.

## Glossary

The words this README and the docs use, in plain language.

| Term | What it means |
|------|---------------|
| **[OCI artifact](https://specs.opencontainers.org/image-spec/)** | The packaging format Docker containers use: a standardized "box" that every container registry already knows how to store, sign, and move around. Model CLI puts the *model* in that same kind of box so it gets all that existing infrastructure for free. |
| **[Registry](https://github.com/opencontainers/distribution-spec)** | A server that stores those boxes (Docker Hub, GitHub Packages, AWS ECR, zot, ...). A warehouse for containers and models. |
| **[Manifest](https://github.com/opencontainers/image-spec/blob/main/manifest.md)** | The box's table of contents: a JSON document listing the files (layers) inside, their checksums, and the annotations. |
| **[Annotations](https://github.com/opencontainers/image-spec/blob/main/annotations.md)** | Key-value tags on the manifest, e.g. "needs a GPU", "needs 16 GB VRAM", "runs on vLLM". Tools can read them *without downloading the model* and decide where it is allowed to run. |
| **[SBOM](https://spdx.dev/)** (Software Bill of Materials) | An ingredients list: which files and packages are in the artifact. Used for security audits. Generated by [Syft](https://github.com/anchore/syft) (recommended), [Trivy](https://trivy.dev/) or [cdxgen](https://github.com/CycloneDX/cdxgen). |
| **[Provenance / SLSA attestation](https://slsa.dev/spec/v1.0/provenance)** | A signed statement of who built the artifact, from what, and when. Proof of origin. |
| **[Sign / verify](https://github.com/ossf/model-signing-spec)** | Cryptographically stamp the artifact so anyone can later check it was not modified. Done by [Cosign](https://www.sigstore.dev/) (Sigstore, recommended) or [Notation](https://github.com/notaryproject/notaryproject) (Notary v2). |
| **[MOF](https://isitopen.ai/)** (Model Openness Framework) | A classification of how open a model is. Class I is fully open (model, code, data, documentation), Class II is partially open, Class III is the least open. |
| **[ORAS](https://oras.land/)** | The tool that pushes and pulls OCI artifacts that are not container images. Model CLI uses it for packaging and pushing to any OCI registry, including [Harbor](https://goharbor.io/): `--registry oras --registry-url <harbor-host>/<project>`. |
| **[ModelPack](https://github.com/modelpack/model-spec)** | An alternative packaging format for models. Model CLI drives it through the [modctl](https://github.com/modelpack/modctl) CLI with `--registry modelpack`. |
| **Serving runtime** ([vLLM](https://docs.vllm.ai/), [KServe](https://kserve.github.io/website/)) | The software that loads the model and answers requests once it is deployed. |
| **Layer deduplication** | An optimization for large models: identical layers are stored once instead of being copied per artifact. |
| **Skill / Reference Skill DLC endpoint** | An *agentic skill* is a packaged capability an AI agent can load. The DLC endpoint is the URL from which such skills are loaded dynamically at serve time. |
| **[GitOps](https://opengitops.dev/)** ([Argo CD](https://argo-cd.readthedocs.io/), [Flux](https://fluxcd.io/)) | A deployment style where you describe what should run in a Git repo and a controller makes the cluster match it. Model CLI validates the artifact and then hands it to one of these (Flux recommended). |
| **[Admission](https://kubernetes.io/docs/reference/access-authn-authz/admission-controllers/)** | The bouncer at the door of a Kubernetes cluster: is this artifact signed, does it meet policy, may it come in? `enforce` can run as this bouncer (an admission webhook). |
| **Air-gapped** | A network with no internet access, common in high-security environments. `validate admission` knows about this as a target environment. |
| **TUI** (Text User Interface) | Interactive menus and forms in the terminal, as opposed to plain flags. |

## Tool Integration

| Task | Model CLI role | Options (recommended first) |
|------|----------------|-----------------------------|
| Package / push | Build manifest, inject annotations | `oras` - any OCI registry: Harbor, GHCR, zot, ... ; or `modelpack` via `modctl` (needs `oras` too) |
| SBOM | Configure format, attach result | `syft`, `trivy`, `cdxgen` |
| Sign / verify | Configure signer, run it | `cosign` (Sigstore), `notary` (Notation / Notary v2) |
| Provenance | Generate SLSA v1.0 attestation | built in |
| Deploy | Validate, then hand off | `flux`, `argocd` |
| Serve | Hand off | `vllm`, `kserve` |
| Cluster checks | Compare annotations with nodes | kubectl |

**Only Go is required to build.** Every other tool is optional, detected at runtime, and reported with an install command when missing.

Standards: [OCI Image Spec](https://specs.opencontainers.org/image-spec/), [OCI Distribution Spec](https://github.com/opencontainers/distribution-spec), [OSSF Model Signing Spec](https://github.com/ossf/model-signing-spec), [Model Openness Framework](https://isitopen.ai/), [SPDX](https://spdx.dev/), [SLSA](https://slsa.dev/), [JSON Schema](https://json-schema.org/).

## Known limitations

Honest list of what is scaffolding today:

- **ModelPack needs two tools** - `--registry modelpack` builds and pushes with [modctl](https://github.com/modelpack/modctl) (`go install github.com/modelpack/modctl@latest`). modctl cannot set manifest annotations or read referrers, so the CNCF annotations are merged into the pushed manifest with ORAS, and digest checks, manifest reads and provenance referrers go through ORAS too. Both `modctl` and `oras` must be installed.
- **The AI config is still also inlined in the local `manifest.json`** - `package` writes the AI config to `config.json` next to the manifest and pushes it as the manifest's config blob, so the registry's config descriptor (digest, size, media type) is the one in the local manifest. The same config is still inlined as `aiConfig` for readers of the local file (`enforce`, `validate`, admission checks); those do not yet read the blob back from the registry, which is pending (#84).
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
