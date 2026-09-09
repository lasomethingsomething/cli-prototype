---
name: model-cli
description: Maintain Model CLI's OCI artifact workflow, provider integrations, wizard, and documentation with accurate prototype boundaries.
---

# Model CLI

Model CLI is a Go orchestrator for packaging and delivering AI artifacts as OCI
artifacts. It collects intent, writes manifest metadata, and delegates supported
operations to selected external tools rather than reimplementing them.

Use this skill when changing commands, workflows, providers, TUI behavior,
metadata, validation, registry integration, or related documentation.

## Repository Structure

| Path | Responsibility |
|---|---|
| `cmd/` | Cobra command definitions, flags, prompts, and wizard flow |
| `internal/workflow/` | Workflow logic, provider interfaces, manifests, validation, and metadata |
| `internal/tui/` | Bubble Tea context panel and package/harden/check TUI models |
| `config/` | Saved tool preferences in `~/.model-cli.yaml` |
| `docs/command-reference.md` | Installation, commands, tool choices, configuration, and troubleshooting |
| `docs/concepts-and-glossary.md` | Durable concepts, terminology, and trial notes |
| `docs/architecture.md` | Implementation architecture and prototype boundaries |
| `README.md` | Product overview and interactive macOS Quick Test Drive |

## Seven-Phase Wizard

Keep the wizard and context-panel progress in this order:

1. Develop & Package
2. Local Hardening & Compliance
3. Supply Chain Check
4. Manifest-Level Validation
5. GitOps Admission & Policy Enforcement
6. Infrastructure & Resource Orchestration
7. Runtime Execution & Optimization

At each context-panel pause, state what Enter does next. Keep documentation's
prompt answers next to the relevant phase in the README.

Current execution boundaries:

- The wizard executes local packaging, hardening, local compliance, optional
  registry publication, and manifest retrieval.
- Wizard signing and verification are demonstrations. Standalone `sign` and
  `verify` delegate to their selected providers.
- Without a Kubernetes cluster, wizard phases 5-7 are guided simulations.
- `serve --runtime vllm` starts vLLM. The KServe provider currently reports the
  intended `InferenceService`; it does not create one.
- The `kserve-vllm` wizard option is a serving-topology demonstration, not a
  combined runtime provider.
- RAG paths are currently demonstrated in the wizard but are not packaged.

## Provider Choices

Provider choices are centralized in `internal/workflow/options.go`. Use those
option lists for every prompt and help string; do not duplicate choice lists.

| Category | Recommended | Other choices |
|---|---|---|
| Registry | `oras` | `modelpack` via `modctl` |
| SBOM | `syft` | `trivy`, `cdxgen` |
| Signing | `cosign` | `notary` via Notation |
| GitOps | `flux` | `argocd` |
| Runtime | `vllm` | `kserve` |

Harbor, GHCR, zot, and `registry:2` are OCI registry destinations, not registry
client choices. ModelPack currently needs both `modctl` and ORAS because ORAS
adds annotations and handles reads and referrers.

The local package workflow currently uses ORAS internally to create the local
artifact. The user-facing registry-client choice belongs to the wizard's
publication stage.

## Metadata And Validation

`package` writes a unified OCI manifest and AI config. `harden` runs after
package on the same model directory, generates an SBOM, classifies MOF, writes
`mof.json`, and updates `manifest.json`.

Keep these validation paths distinct:

- `validate gitops`, `validate admission`, `validate nodes`, and `validate
  runtime` share trust, infrastructure, and environment policy evaluation.
- `validate manifest` and `enforce` evaluate the separate `ai.assets` metadata
  contract.

Annotation conventions are evolving with the
[CNCF AI inner-loop initiative #1740](https://github.com/cncf/toc/issues/1740).
The wizard's manifest phase must inspect and report annotations without blocking
the guided flow on a fixed annotation contract.

## Prompt And Configuration Rules

- A command-line flag wins over saved configuration and prompting.
- An explicit empty flag is intentional and must not trigger a prompt.
- Non-interactive mode is active for `--non-interactive`,
  `MODEL_CLI_NO_INTERACTIVE=1`, or non-TTY stdin. Missing required values must
  fail with a message naming the relevant flag.
- The wizard requires an interactive terminal.
- Saved settings are defaults or remembered selections, not a reason to hide a
  required user-facing choice in the guided flow.
- Expand `~` and `~/...` entered into wizard path prompts before filesystem use.

## Documentation Rules

- Keep the README focused on the interactive Quick Test Drive. It is the
  canonical walkthrough, including macOS/Podman preparation and exact answers.
- Put command syntax, non-interactive use, tool tables, configuration, and
  troubleshooting in `docs/command-reference.md`.
- Put concepts and terminology in `docs/concepts-and-glossary.md`.
- Put architecture and known technical limits in `docs/architecture.md`.
- Do not document simulated behavior as executed behavior. State whether an
  integration is a demonstration, handoff, validation, or actual execution.

## Validation

Run focused tests for touched components:

```bash
go test ./cmd ./config ./internal/tui ./internal/workflow
git diff --check
```

For a wider change, also run:

```bash
go build ./...
go test ./...
gofmt -l .
go vet ./...
```