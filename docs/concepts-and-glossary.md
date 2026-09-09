# Concepts and Glossary

This reference explains the terms used throughout Model CLI. The interactive
Quick Test Drive is in the repository README.

## Key Concepts

**OCI artifacts.** Models are packaged in the same format Docker, Kubernetes,
and container registries already understand, so standard tooling can push, pull,
sign, and route them.

**Annotations.** Model CLI attaches metadata to OCI manifests under CNCF AI
Interoperability Profile keys:

- Profile: `org.cncf.ai.interop.profile.version`, `org.cncf.ai.artifact.type`
- Security: signing framework, SBOM format, provenance type, packaging format
- MOF: openness class (I / II / III), version, components
- Runtime: serving runtime, accelerator, minimum CUDA and memory
- Infrastructure: `ai.node.gpu.type`, `ai.node.vram.min`, `ai.node.gpu.topology`
- Execution: `ai.runtime.type`, layer deduplication, skill DLC endpoint, skill references

Policy engines, GitOps tools, and registries can inspect these without
downloading the model.

**Metadata contract.** The policy-oriented `validate` targets (`gitops`,
`admission`, `nodes`, and `runtime`) share annotation evaluation and report
formats (text, `--quiet`, or `--json-output`). `validate manifest` and
`enforce` evaluate the separate `ai.assets` metadata contract. Annotation
conventions are evolving as part of the
[CNCF AI inner-loop initiative #1740](https://github.com/cncf/toc/issues/1740).

## Glossary

| Term | What it means |
|------|---------------|
| **[OCI artifact](https://specs.opencontainers.org/image-spec/)** | A standardized package of files, checksums, and metadata that registries can store and move. |
| **[Registry](https://github.com/opencontainers/distribution-spec)** | A server that stores OCI artifacts, such as Harbor, GHCR, or zot. |
| **[Manifest](https://github.com/opencontainers/image-spec/blob/main/manifest.md)** | The artifact's table of contents: layers, checksums, and annotations. |
| **[Annotations](https://github.com/opencontainers/image-spec/blob/main/annotations.md)** | Key-value metadata on a manifest, such as runtime or hardware requirements. |
| **[SBOM](https://spdx.dev/)** | A Software Bill of Materials: an inventory of files and packages used for security review. Syft is the recommended generator. |
| **[Provenance / SLSA attestation](https://slsa.dev/spec/v1.0/provenance)** | A signed statement of what built an artifact, from which inputs, and when. |
| **[Sign / verify](https://github.com/ossf/model-signing-spec)** | Cryptographically establish and check artifact integrity. Cosign is recommended; Notation provides a Notary v2 option. |
| **[MOF](https://isitopen.ai/)** | The Model Openness Framework classifies how much of a model's weights, code, data, documentation, and license are open. |
| **[ORAS](https://oras.land/)** | The recommended client for pushing OCI artifacts to registries such as Harbor, GHCR, or zot. |
| **[ModelPack](https://github.com/modelpack/model-spec)** | An alternative model packaging format driven by `modctl`; Model CLI currently also needs ORAS for annotations and referrers. |
| **Serving topology** | The arrangement that serves a model: direct vLLM, KServe, or KServe-managed vLLM. The combined option is currently demonstrated by the wizard rather than executed as a combined provider. |
| **Layer deduplication** | Storing identical model layers once rather than once per artifact. |
| **GitOps** | A deployment approach where a controller, such as Flux or Argo CD, reconciles a cluster with the configuration in Git. |
| **Admission** | The cluster policy decision about whether an artifact may enter an environment. |
| **Air-gapped** | A network with no internet access, common in high-security environments. |
| **TUI** | Text User Interface: interactive menus and forms in the terminal. |

## Trial Notes

A dummy text file is sufficient to demonstrate packaging, hardening, local
compliance, and registry publication. The Quick Test Drive uses this to prove
the artifact flow without requiring a real model or GPU. It creates a local
manifest, SBOM, and MOF metadata; publishing to the optional local Podman
registry demonstrates OCI upload and manifest retrieval.