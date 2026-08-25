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

## Pull Requests

- Follow the pluggable provider pattern for new tools
- Keep TUI clean and uncluttered
- Add clear error messages with installation instructions
- Document new features
- Ensure all workflows align with relevant standards (OCI, OSSF, MOF, CNCF AI Interoperability Profile)
