# Model CLI

**Your tour guide through the CNCF's "[Cloud Native and OCI Compliant Inner-Loop Tooling & Packaging for AI Engineers](https://github.com/cncf/toc/issues/1740)" initiative.**

## What is this, in plain words?

A trained ML model is a folder of files. Moving it safely from your laptop to production requires packaging, security checks, publishing, and deployment. Specialized tools in the [CNCF ecosystem](https://www.cncf.io/) already handle each task.

Model CLI guides you through that journey, collects the needed details, and lets you choose an appropriate tool for each task.

The workflow, in seven action-led steps:

1. **Develop & Package**: Creates a local OCI artifact manifest from the model folder and tags it with CNCF AI Interoperability Profile metadata.
2. **Local Hardening & Compliance**: Generates an SBOM and MOF classification, and records both in the packaged artifact.
3. **Supply Chain Check**: Cryptographically signs the artifact so tampering can be detected later, and records a proof of origin (provenance) for the finished artifact.
4. **Manifest-Level Validation**: Delegates upload to a registry to ORAS or ModelPack.
5. **GitOps Admission & Policy Enforcement**: Hands the artifact to Argo CD or Flux as a prototype. _Does not auto-enforce policy_.
6. **Infrastructure & Resource Orchestration**: Validates that infrastructure matches the artifact’s runtime and hardware requirements.
7. **Runtime Execution & Optimization**: Validates serving runtime availability (vLLM, KServe). _Does not execute or optimize_.

If terms like *SBOM*, *MOF*, or *admission* are new to you, see the
[Concepts and Glossary](docs/concepts-and-glossary.md).

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

## Documentation

- [Command Reference](docs/command-reference.md)
- [Concepts and Glossary](docs/concepts-and-glossary.md)
- [Architecture](docs/architecture.md)
- [TUI Guide](docs/tui.md)
- [Resources](docs/resources.md)
- [Contributing](CONTRIBUTING.md)
