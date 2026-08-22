# Model CLI

Your tour guide through the secure ML model deployment journey.

## Documentation

See [docs](docs/) for detailed information.

## Overview

Start with the full guided journey:

```bash
model-cli wizard
```

This walks you through packaging, signing, verifying, and deploying your model.

Individual commands are also available:

```bash
model-cli package
model-cli sign
model-cli verify
model-cli deploy
```

## Contributing

Follow the provider pattern for new tool integrations:

1. Add interface in internal/workflow/providers.go
2. Implement concrete provider
3. Register in factory function (Get*Provider)
4. Update CLI commands to use it

## License

Apache License 2.0
