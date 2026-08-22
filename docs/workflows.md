### Workflows

#### Local Development

Package model without deploying:
```bash
model-cli wizard --skip-deploy --skip-signing
```

Later, when ready for production:
```bash
model-cli sign --artifact my-model:v1
model-cli verify --artifact my-model:v1
model-cli deploy
```

#### CI/CD Pipeline

```bash
#!/bin/bash
set -e

model-cli package --model phi-4-mini --registry oras --output my-model:v1
model-cli sign --artifact my-model:v1 --signer sigstore --key $SIGNING_KEY
model-cli verify --artifact my-model:v1

export KUBECONFIG=staging-config
model-cli deploy --gitops argo --repo $REPO_URL --path ./manifests

export KUBECONFIG=prod-config
model-cli verify --artifact my-model:v1
model-cli deploy --gitops argo --repo $REPO_URL --path ./manifests
```

#### Team Onboarding

```bash
model-cli wizard
model-cli --help
model-cli package --help
model-cli sign --help
```
