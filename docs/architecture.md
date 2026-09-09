# Architecture

Model CLI is a Go command-line orchestrator. It collects deployment intent,
writes OCI manifest metadata, and delegates supported operations to external
tools rather than reimplementing registry, SBOM, signing, GitOps, or serving
systems.

```
User
  |
  v
Cobra commands and interactive prompts
  - wizard, package, harden, sign, verify, push
  - validate, deploy, serve, schedule, enforce, map, search
  |
  v
Workflows and validation
  - package, harden, registry, provenance, relationships
  - profile/policy evaluation and metadata-contract validation
  |
  v
Provider interfaces
  - registry, SBOM, signing, GitOps, runtime
  |
  v
External tools and systems
  - ORAS, modctl, Syft, Cosign, Notation, Flux, Argo CD, vLLM, KServe
  - OCI registries and Kubernetes clusters
```

## Commands And Workflows

Commands in `cmd/` collect flags and prompts, load saved preferences, and
create workflows in `internal/workflow/`.

| Workflow or command family | Responsibility |
|---|---|
| `PackageWorkflow` | Builds a unified OCI manifest and AI config from a model directory; can push when a registry destination is provided. |
| `HardenWorkflow` | Generates an SBOM, classifies MOF, and records results in the packaged local manifest. |
| `push` | Pushes an artifact and its manifest annotations, and can generate provenance and sign at push time. |
| `sign`, `verify` | Delegate signing and signature/provenance verification to the selected provider. |
| `validate` | Provides local, manifest, GitOps, admission, node, and runtime validation targets. |
| `deploy`, `serve`, `schedule` | Provide GitOps handoff, runtime handoff, and scheduling-related entry points. |
| `enforce`, `map`, `search` | Enforce the metadata contract, write relationships, and discover annotated registry artifacts. |

## Interactive TUI

The wizard combines Huh prompt forms with a Bubble Tea context panel. The
context panel has `Progress`, `Config`, `Model`, `Logs`, `Env`, and `Help` tabs.
It follows the seven-phase journey:

1. Develop & Package
2. Local Hardening & Compliance
3. Supply Chain Check
4. Manifest-Level Validation
5. GitOps Admission & Policy Enforcement
6. Infrastructure & Resource Orchestration
7. Runtime Execution & Optimization

The wizard executes local packaging, hardening, local compliance, optional OCI
publication, and manifest retrieval. Its signing, verification, and no-cluster
phases are guided simulations. With a configured cluster, it can hand off to a
selected GitOps provider, but that remains a prototype integration.

Separate package, harden, and check TUI models live under `internal/tui/` for
their respective command experiences.

## Provider Pattern

Each external integration has an interface and factory in
`internal/workflow/`. Workflows depend on the interface, not a concrete CLI.
Providers expose their name, installation check, installation instructions, and
operation methods; tests can substitute fakes.

| Category | Current choices |
|---|---|
| Registry | `oras` (recommended), `modelpack` via `modctl` |
| SBOM | `syft` (recommended), `trivy`, `cdxgen` |
| Signing | `cosign` (recommended), `notary` via Notation |
| GitOps | `flux` (recommended), `argocd` |
| Runtime | `vllm` (recommended), `kserve` |

The wizard also offers a **serving topology** demonstration: direct vLLM,
KServe, or KServe-managed vLLM. The combined KServe + vLLM option is not a
single runtime provider and is currently simulated.

## Registry Integration

```
Model CLI
  |
  +-- ORAS provider ------> ORAS CLI ------> OCI registry
  |
  +-- ModelPack provider -> modctl build/push -> OCI registry
                          -> ORAS annotations, reads, and referrers
```

ORAS is the recommended registry client. Harbor, GHCR, zot, and the local OCI
Distribution Registry are registry destinations, not ORAS alternatives.

ModelPack is implemented through `modctl`, but modctl cannot set manifest
annotations or read referrers. Model CLI therefore uses ORAS alongside modctl
for those operations; both tools are required for the ModelPack path.

`search` walks OCI registries through the chosen client and filters candidates
client-side. Registry-wide discovery requires a registry with a catalog API,
such as Harbor, zot, or `registry:2`.

## Metadata And Validation

`package` writes OCI manifest annotations for the CNCF AI Interoperability
Profile, including artifact type, security, MOF, runtime, infrastructure, and
execution metadata. `harden` updates the same local manifest with SBOM and MOF
results.

The repository has two validation paths:

- Policy-oriented `validate gitops`, `admission`, `nodes`, and `runtime` share
  profile, trust, infrastructure, and environment evaluation with text, quiet,
  and JSON report formats.
- `validate manifest` and `enforce` evaluate the separate `ai.assets` metadata
  contract.

Annotation conventions and model metadata are evolving as part of the
[CNCF AI inner-loop initiative #1740](https://github.com/cncf/toc/issues/1740).
The wizard's manifest step therefore retrieves and reports annotations without
enforcing a fixed annotation set.

## GitOps And Admission

Model CLI provides preflight checks and GitOps handoff; a configured cluster and
its policy engines make the final admission and scheduling decisions.

- `validate gitops` checks the artifact's trust-profile and infrastructure
  annotations before promotion.
- `validate admission` adds environment evaluation for development, staging,
  production, air-gapped, and hybrid-cloud targets. In hybrid-cloud mode, a
  region is evaluated against data-residency metadata; `--strict` turns a
  mismatch into a failure.
- `deploy` delegates to Flux (recommended) or Argo CD after the user supplies
  the Git repository and manifest path. It requires a configured cluster and
  the selected client.
- Signature verification and admission enforcement remain the responsibility of
  external systems such as Cosign/Sigstore policy tooling, OPA/Gatekeeper, or
  Kyverno. Model CLI does not install or configure their cluster policies.

## Execution Boundaries

The following boundaries are deliberate in the current prototype:

- `serve --runtime vllm` starts the vLLM process. The KServe provider currently
  reports the intended `InferenceService` rather than creating one.
- `validate nodes` reads a configured cluster with `kubectl` and checks declared
  GPU type, vRAM, and topology; Kubernetes makes the scheduling decision.
- GitOps deployment invokes Flux or Argo CD after the user supplies a
  repository and manifest path. It assumes a configured cluster and provider.
- Local test drives can publish to a Podman-hosted OCI registry at
  `localhost:5000`, without Docker Desktop or a cloud account.
- The ModelPack path needs both `modctl` and ORAS: modctl builds and pushes,
  while ORAS applies annotations and handles reads and referrers.
- The AI config is written both as `config.json` and inline in the local
  `manifest.json`. Registry readers do not yet load that config blob back.
