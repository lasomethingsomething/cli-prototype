# Command Reference

Installation, non-interactive use, command examples, tool choices, configuration,
and troubleshooting for Model CLI. The interactive Quick Test Drive is in the
repository README.

## Installation

```bash
go install github.com/lasomethingsomething/cli-prototype@latest
model-cli --help
```

This installs `model-cli` to `$GOPATH/bin`; ensure that directory is on your
`PATH`.

## Non-Interactive Package And Harden

In CI or scripts, prompts are disabled automatically when stdin is not a
terminal. Pass required choices as flags, or add `--non-interactive` explicitly:

```bash
model-cli package \
  --model test-model \
  --model-path ~/test-model \
  --artifact test:v1 \
  --registry oras \
  --registry-url "" \
  --accelerator cpu \
  --non-interactive

model-cli harden \
  --model test-model \
  --model-path ~/test-model \
  --artifact test:v1 \
  --sbom-tool syft \
  --mof-class "" \
  --non-interactive
```

## Commands

`model-cli wizard` is the guided tour. Each phase is also available as a
standalone command.

### Package And Secure

```bash
# Package a model with the recommended OCI client
model-cli package --model phi-4-mini --model-path ./models --artifact my-model:v1 --registry oras

# Package a ModelPack artifact
model-cli package --model phi-4-mini --model-path ./models --artifact my-model:v1 --registry modelpack

# Harden the packaged artifact with an SBOM and MOF classification
model-cli harden --model phi-4-mini --model-path ./models --artifact my-model:v1 --sbom-tool syft

# Sign and then verify an artifact
model-cli sign --artifact my-model:v1 --signer cosign
model-cli verify --artifact my-model:v1 --signer cosign

# Push an artifact to an OCI registry
model-cli push --artifact my-model:v1 --registry oras --destination ghcr.io/my-org
```

`package` can also package agentic skills with `--skill`. `harden` must run on
the same model path after `package`, because it records the SBOM and MOF results
in `manifest.json`.

### Validate

```bash
model-cli validate manifest --manifest ./models/manifest.json
model-cli validate local --model-path ./models
model-cli validate gitops --artifact ghcr.io/my-org/my-model:v1
model-cli validate admission --artifact ghcr.io/my-org/my-model:v1 --env production
model-cli validate nodes --artifact ghcr.io/my-org/my-model:v1
model-cli validate runtime --artifact ghcr.io/my-org/my-model:v1
```

Use `--quiet` for a one-line result or `--json-output` for CI on the validation
subcommands.

### Deploy And Serve

```bash
model-cli deploy --gitops flux --registry oras --model my-model --repo https://github.com/you/model-manifests --path ./k8s --has-k8s
model-cli serve --runtime vllm --model-path ./models/phi-4-mini --host 0.0.0.0 --port 8080
model-cli schedule --artifact ghcr.io/my-org/my-model:v1
model-cli map --model my-model --skill my-skill --pipeline my-pipeline
model-cli search --destination ghcr.io/my-org --registry-tool oras --type model
```

`deploy`, `schedule`, and KServe serving have prototype limits. See the README
and individual command help before relying on them in production.

## Tool Integration And Choices

Tools are selected per category. The recommended option is the default; tool
choices are remembered in `~/.model-cli.yaml`.

| Task | Recommended | Other choices and current role |
|------|-------------|--------------------------------|
| Package / push | `oras` | `modelpack` via `modctl`; both need ORAS today. Harbor, GHCR, and zot are registry destinations. |
| SBOM | `syft` | `trivy`, `cdxgen`; generate and attach SBOM metadata. |
| Sign / verify | `cosign` | `notary` via Notation; sign and check integrity/provenance. |
| Provenance | built in | Generated during the supported push path. |
| Deploy | `flux` | `argocd`; hand off to a configured GitOps provider. |
| Serve | `vllm` | `kserve`; vLLM starts locally, while KServe is an InferenceService handoff. |
| Cluster checks | `kubectl` | Reads cluster state for node and runtime validation. |

Only Go is needed to build Model CLI. Other tools are checked when their
corresponding command or workflow phase runs.

Standards: [OCI Image Spec](https://specs.opencontainers.org/image-spec/),
[OCI Distribution Spec](https://github.com/opencontainers/distribution-spec),
[OSSF Model Signing Spec](https://github.com/ossf/model-signing-spec),
[Model Openness Framework](https://isitopen.ai/), [SPDX](https://spdx.dev/),
[SLSA](https://slsa.dev/), and [JSON Schema](https://json-schema.org/).

## Configuration

Model CLI saves selected tools in `~/.model-cli.yaml`:

```yaml
gitops: flux
registry: oras
signer: cosign
runtime: vllm
serving-topology: vllm
```

Command-line flags override saved values. Use `model-cli <command> --help` for
the flags accepted by a particular command.

## Troubleshooting

**Command not found.** Ensure `$GOPATH/bin` is in `PATH`, or build locally with
`go build -o model-cli .` and run `./model-cli`.

**Tool not installed.** Install the tool named by the command's error message.
For example, Syft is required when `harden` generates an SBOM:

```bash
brew install anchore/syft/syft
```

**Terminal display issues.** Use a terminal with at least 100 columns and
24-bit color support, such as iTerm2, VS Code, or Alacritty.