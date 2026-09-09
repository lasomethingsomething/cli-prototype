# Model CLI

**Your tour guide through the secure ML model deployment journey.**

## What is this, in plain words?

A trained ML model is a folder of files. Moving it safely from your laptop to production requires packaging, security checks, publishing, and deployment. Specialized tools in the [CNCF ecosystem](https://www.cncf.io/) already handle each task.

Model CLI guides you through that journey, collects the needed details, and lets you choose an appropriate tool for each task.

The workflow, in seven action-led steps:

1. **Package** - bundle the model folder into a standard container-style artifact and tag it with metadata.
2. **Harden** - generate an ingredients list (SBOM) and classify how open the model is (MOF), and record both in the packaged artifact.
3. **Sign** - stamp it cryptographically so tampering can be detected later, and record a proof of origin (provenance) for the finished artifact.
4. **Push** - upload it to a registry.
5. **Validate** - check the metadata against your rules locally, at the registry, and at the cluster door, using one shared engine so passing in one place means passing everywhere.
6. **Orchestrate** - match the artifact's runtime and hardware requirements to infrastructure that can run it.
7. **Deploy** - hand the approved artifact to Argo CD / Flux and a serving runtime such as vLLM or KServe.

If terms like *SBOM*, *MOF*, or *admission* are new to you, see the [Glossary](#glossary) below.

> **Status: prototype.** Model CLI guides ML-model packaging and delivery as OCI artifacts. It collects intent, writes standardized metadata, and delegates supported operations to selected tools such as ORAS, Syft, Cosign, Flux, and Argo CD. Some wizard stages and integrations remain simulated or incomplete; see [Known limitations](#known-limitations) before relying on a step in production.

## System Requirements

For the macOS Quick Test Drive, install these prerequisites before starting the
wizard:

> This list intentionally includes the recommended tools needed to demonstrate
> the complete flow. In normal use, Model CLI lets you choose a tool per phase
> and checks for it only when that phase runs.

| Requirement | Why it is needed | Install or check |
|-------------|------------------|------------------|
| Go 1.23+ | Build Model CLI | `go version` |
| Homebrew | Install the external tools | [brew.sh](https://brew.sh/) |
| Current Xcode Command Line Tools | Required when Homebrew builds a dependency | System Settings > General > Software Update; if no update is offered, run `xcode-select --install` |
| ORAS (recommended) | Create OCI artifacts; ModelPack is the alternative | `brew install oras` |
| Syft (recommended) | Generate the SBOM; Trivy and cdxgen are alternatives | `brew install anchore/syft/syft` |
| Cosign (recommended) | Show the Sigstore signing and verification branch; Notation is the alternative | `brew install sigstore/tap/cosign` |
| Podman (recommended for local publishing) | Run a local OCI registry without an account or cloud service | `brew install podman` |
| Flux (recommended) | Execute the GitOps path against a Kubernetes cluster; Argo CD is the alternative | `brew install fluxcd/tap/flux` |
| kubectl and a Kubernetes cluster | Execute real GitOps admission, node checks, and runtime checks | `brew install kubectl` |
| vLLM (recommended) and/or KServe | Execute serving instead of the wizard's runtime simulation; KServe can manage a Kubernetes deployment that uses vLLM | vLLM: `pip install vllm`; KServe: install its Kubernetes operator |

The publish demonstration uses Podman's Linux VM and the open-source OCI
Distribution Registry at `localhost:5000`. It does not require Docker Desktop.
Flux, `kubectl`, a Kubernetes cluster, and a runtime are not needed for the
macOS test drive: without a cluster, the wizard demonstrates phases 5-7 as
guided simulations. Argo CD and KServe remain supported alternatives to Flux
and vLLM, respectively.

## Quick Test Drive (5 minutes)

Try the complete local package, harden, compliance, and publish flow with a
plain text file. No real model, registry account, or GPU is required.

### Interactive Wizard

#### Before Step 1: Build And Prepare

Choose the build path that matches where you are starting:

```bash
# First-time user (requires Go 1.23+)
git clone https://github.com/lasomethingsomething/cli-prototype.git
cd cli-prototype
go build -o model-cli .
```

```bash
# Contributor: run this from your existing cli-prototype checkout
go build -o model-cli .
```

From the `cli-prototype` directory, create the harmless dummy model and install
the local package and hardening tools:

```bash
mkdir -p ~/test-model
echo "test" > ~/test-model/model.txt
brew install oras
brew install anchore/syft/syft
brew install sigstore/tap/cosign
```

To include the optional publish demonstration, prepare the local registry before
starting the wizard. Run `podman machine init` only the first time.

```bash
brew install podman
podman machine init
podman machine start
podman run -d --rm --name model-cli-registry -p 5000:5000 registry:2
curl http://localhost:5000/v2/
```

`{}` from `curl` means the local registry is ready. Start the wizard. Without a
Kubernetes cluster, it simulates phases 5-7 so you can see the complete
journey:

```bash
./model-cli wizard
```

#### Step 1: Develop & Package

| Prompt | Answer |
|--------|--------|
| Model name | `test-model` |
| Model path | `~/test-model` |
| Artifact name | `test:v1` |
| Include RAG context? | `No` to skip |

Press Enter at the context panel to package the model locally. It writes `manifest.json` and `config.json` to `~/test-model` without contacting a registry; that choice comes at the publish prompt in Step 3.

#### Step 2: Local Hardening & Compliance

| Prompt | Answer |
|--------|--------|
| SBOM tool | `syft (recommended)` |
| Model Openness Framework class | `auto (recommended)` |

Press Enter to run the local compliance check after hardening completes. This generates the SBOM and MOF metadata, then checks both before the artifact can continue.

#### Step 3: Supply Chain Check

| Prompt | Answer |
|--------|--------|
| Signing tool | `cosign (recommended) - Sigstore` |

The wizard shows the signing and verification stages. Those stages are currently guided simulations; use the standalone `model-cli sign` and `model-cli verify` commands for external tool execution. To omit this branch, run `./model-cli wizard --skip-signing` instead.

At the publish prompt, use:

| Prompt | Answer |
|--------|--------|
| Publish this artifact to an OCI registry? | `Yes` |
| Where is the OCI registry? | `a local Podman registry at localhost:5000` |
| Which client should publish the artifact? | `oras (recommended)` |

The wizard verifies `localhost:5000` before it pushes `test:v1`.

ORAS works with Harbor, GHCR, zot, and other OCI registries; those are destinations, not alternative clients. `modelpack` is the alternative CNCF ModelPack option.

#### Step 4: Manifest-Level Validation

The wizard retrieves `manifest.json` after the publish choice and displays its
annotation count without enforcing an evolving annotation contract. When you
chose the local Podman registry, it fetches
`localhost:5000/test:v1`; when you chose not to publish, it inspects the local
`~/test-model/manifest.json` instead. The Journey Complete summary then
shows the exact command to inspect the local manifest and, when published, the
registry copy.

Annotation conventions and model metadata are evolving as part of the
[CNCF AI inner-loop initiative #1740](https://github.com/cncf/toc/issues/1740),
so this wizard inspects them without enforcing a fixed contract.

#### Step 5: GitOps Admission & Policy Enforcement

| Prompt | Answer |
|--------|--------|
| GitOps tool | `flux (recommended)` |
| Do you have a Kubernetes cluster? | `No` |

After Step 4 inspects the manifest, the wizard asks how to promote the artifact.
For this Quick Test Drive, choose `No`: the wizard simulates the GitOps
admission and policy stage without requiring a cluster.

A connected-cluster path exists, but is still a prototype integration: it asks for the Git
repository URL and manifest path, then invokes the selected Flux or Argo CD client against a preconfigured cluster. Press Enter at the next context panel to continue to infrastructure orchestration.

#### Step 6: Infrastructure & Resource Orchestration

The wizard simulates how Kubernetes would match the artifact's declared
hardware requirements to cluster nodes. With a configured cluster,
`model-cli validate nodes` uses `kubectl` to check declared GPU type, vRAM, and
topology; Kubernetes still makes the final scheduling decision. Press Enter at
the next context panel to continue to runtime execution.

#### Step 7: Runtime Execution & Optimization

| Prompt | Answer |
|--------|--------|
| Serving topology | `vllm (recommended) - direct model server` |

The wizard offers direct `vllm (recommended)`, `kserve`, or `kserve-vllm` for a
KServe-managed vLLM deployment. This is a guided simulation: it does not yet
write the selected topology into the artifact or start a runtime. For an
artifact that declares runtime requirements, use `model-cli validate runtime`
against a cluster. `model-cli serve --runtime vllm` starts local vLLM; the
current KServe provider reports the intended InferenceService rather than
creating it.

The test creates these local files. When you publish, ORAS uploads the model
directory and its manifest annotations to the selected registry:

- `~/test-model/manifest.json` - an OCI manifest carrying CNCF AI Interoperability Profile, SBOM format, and MOF annotations
- `~/test-model/sbom.spdx-json` - the SBOM
- `~/test-model/mof.json` - the MOF metadata

Phases 5-7 are guided simulations when no Kubernetes cluster is connected.
The standalone `validate`, `deploy`, `schedule`, and `serve` commands provide
the corresponding cluster and runtime entry points; see their individual
prototype limits above.

Stop the temporary local registry when the test is complete:

```bash
podman stop model-cli-registry
```

### Non-Interactive Package And Harden

For CI or scripts, pass every required choice as a flag. This produces the same
local package and hardening results without prompts:

```bash
./model-cli package \
  --model test-model \
  --model-path ~/test-model \
  --artifact test:v1 \
  --registry oras \
  --registry-url "" \
  --accelerator cpu \
  --non-interactive

./model-cli harden \
  --model test-model \
  --model-path ~/test-model \
  --artifact test:v1 \
  --sbom-tool syft \
  --mof-class "" \
  --non-interactive
```

Signing and the SLSA provenance attestation are separate: `model-cli sign --artifact test:v1`.

A detailed, annotated walkthrough of this run is in [docs/trial-run-report.md](docs/trial-run-report.md).

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
| **[ORAS](https://oras.land/)** | The recommended tool for packaging and pushing OCI artifacts that are not container images. It works with any OCI registry, including [Harbor](https://goharbor.io/): `--registry oras --registry-url <harbor-host>/<project>`. |
| **[ModelPack](https://github.com/modelpack/model-spec)** | An alternative CNCF model packaging format. Model CLI drives it through the [modctl](https://github.com/modelpack/modctl) CLI with `--registry modelpack`; ORAS is also currently needed for annotations and referrers. |
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
| Package / push | Build manifest, inject annotations | `oras` (recommended) - any OCI registry: Harbor, GHCR, zot, ... ; or `modelpack` via `modctl` (also needs `oras` today) |
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
