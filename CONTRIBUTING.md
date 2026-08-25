# Contributing

## Building from Source

For development, build from source:

```bash
# Clone the repository
git clone https://github.com/lasomethingsomething/cli-prototype.git
cd cli-prototype

# Build the CLI
go build -o model-cli .

# Verify it works
./model-cli --help
```

## Development Workflow

Provider interfaces live in `internal/workflow/`, one file per tool family:

| File | Interface | Implementations |
|------|-----------|-----------------|
| `registry.go` | `RegistryProvider` | ORAS, ModelPack |
| `signing.go` | `SigningProvider` | Sigstore (cosign), Notary v2 (notation) |
| `sbom.go` | `SBOMGenerator` | Syft, Trivy, cdxgen |
| `gitops.go` | `GitOpsProvider` | Argo CD, Flux |
| `runtime.go` | `RuntimeProvider` | vLLM, KServe |
| `mof.go` | `MOFClassifier` | built-in classifier |

To add a tool:

1. Add the interface (or reuse an existing one) in the matching file
2. Implement the concrete provider
3. Register it in the factory function (`Get*Provider`)
4. Add or extend the CLI command in `cmd/`
5. Support both interactive and non-interactive modes

## Testing

```bash
go build ./...
go test ./...
gofmt -l .      # CI fails if this prints anything
go vet ./...
```

### End-to-end tests

`test/e2e` drives the built binary against a real OCI registry ([zot](https://zotregistry.dev)) with
real `oras`: package → push → fetch the manifest back → `validate gitops` / `validate admission`,
and `push` with a provenance referrer. CI runs it on every PR. Locally:

```bash
docker run -d --name zot -p 5000:5000 \
  -v "$PWD/test/e2e/zot-config.json:/etc/zot/config.json:ro" \
  ghcr.io/project-zot/zot-linux-amd64:latest
MODEL_CLI_E2E_REGISTRY=localhost:5000 go test -tags e2e -v ./test/e2e/
```

Without `MODEL_CLI_E2E_REGISTRY` the e2e package is skipped, so plain `go test ./...` stays fast.

## Pull Requests

- Follow the pluggable provider pattern for new tools
- Keep TUI clean and uncluttered
- Add clear error messages with installation instructions
- Document new features
- Ensure all workflows align with relevant standards (OCI, OSSF, MOF, CNCF AI Interoperability Profile)
