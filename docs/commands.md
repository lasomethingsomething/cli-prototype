### Commands

#### wizard
Full guided workflow through package, sign, verify, deploy.

Options:
```bash
model-cli wizard --skip-signing
model-cli wizard --skip-deploy
model-cli wizard --skip-signing --skip-deploy
```

#### package
Package model as OCI artifact.

Prompts for: model name, model path, artifact name, registry URL, include RAG context.

#### sign
Sign OCI model artifact with Sigstore or Notary v2.

Prompts for: artifact to sign, signing tool, key to use.

#### verify
Verify signature of an OCI model artifact.

Prompts for: artifact to verify, signing tool used.

#### deploy
Deploy to Kubernetes using GitOps.

Prompts for: GitOps tool, registry tool, model name, has Kubernetes, git repository URL, manifest path.
