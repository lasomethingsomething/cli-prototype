### Contributing

We welcome contributions.

#### Adding a New Tool

Follow the provider pattern:

1. Add interface in internal/workflow/providers.go
2. Implement concrete provider
3. Register in factory function (Get*Provider)
4. Update CLI commands to use it

#### Development Setup

```bash
git clone https://github.com/lasomethingsomething/cli-prototype.git
cd cli-prototype
go build -o model-cli .
./model-cli --help
```

#### Running Tests

```bash
go test ./...
```
