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

1. Add interface in `internal/workflow/providers.go`
2. Implement concrete provider
3. Register in factory function (`Get*Provider`)
4. Add CLI command
5. Support both interactive and non-interactive modes

## Testing

Run the test suite:
```bash
go test ./...
```

## Pull Requests

- Follow the pluggable provider pattern for new tools
- Keep TUI clean and uncluttered
- Add clear error messages with installation instructions
- Document new features
- Ensure all workflows align with relevant standards (OCI, OSSF, MOF, CNCF AI Interoperability Profile)
