# Model CLI

**Your tour guide through the CNCF's "[Cloud Native and OCI Compliant Inner-Loop Tooling & Packaging for AI Engineers](https://github.com/cncf/toc/issues/1740)" initiative.**

## What is this, in plain words?

A trained ML model is a folder of files. Moving it safely from your laptop to production requires packaging, security checks, publishing, and deployment. Specialized tools in the CNCF ecosystem already handle each task.

model-cli is a thin orchestrator: it calls oras, syft, cosign, etc. via your PATH and does not bundle them. Model CLI guides you through that journey, collects the needed details, and lets you choose an appropriate tool for each task.

## System Requirements

macOS + Homebrew; run `model-cli doctor --fix`

> **Note:** The wizard pushes to your current Git repository. Ensure you own the repo or are working in a fork. An SSH key is required for Git operations.

## Quickstart (the golden path)

This is the recommended path. Each command does one thing:

```bash
# 1. Install the CLI (Intel: darwin_amd64, Apple Silicon: darwin_arm64)
curl -sL https://github.com/lasomethingsomething/cli-prototype/releases/latest/download/model-cli_darwin_amd64.tar.gz | tar xz
chmod +x model-cli
mv model-cli /usr/local/bin

# 2. Install missing tools and verify your environment
model-cli doctor --fix

# 3. Fork this repo on GitHub, then clone your fork and enter it
git clone ssh://git@github.com/your-username/cli-prototype.git
cd cli-prototype

# 4. Set up a Flux-managed minikube cluster with all dependencies (one-shot, ~10 minutes)
model-cli setup --yes

# 5. Package, publish, and deploy the sample iris model
model-cli wizard
```

On a cold start, `model-cli setup --yes` automatically starts minikube, installs Flux, bootstraps the cluster, and waits for all components to be ready. You will see a brief pause (~60s) when the cert-manager webhook race fires; the tool auto-recovers with a flux reconcile and continues — this is expected behavior, not a failure.

The wizard finishes by running a real in-cluster prediction automatically and prints `✓ Prediction served: ...` followed by `✓ Verified: model served a prediction` in the recap. The signing and verification steps in the wizard are labeled as "simulated" — this is intentional and honest.

## Manual setup (optional)

If you prefer to set up the environment manually instead of using `model-cli setup --yes`:

```bash
# Start minikube (use Docker driver on macOS)
minikube start --driver=docker --cpus=4 --memory=6g

# Start a local registry
podman run -d --rm --name model-cli-registry -p 5000:5000 registry:2

# Bootstrap Flux (token auth recommended)
GITHUB_TOKEN=$(gh auth token) flux bootstrap git --url=https://github.com/your-username/cli-prototype.git --branch=main --path=./clusters/minikube --token-auth

# Wait for convergence
kubectl wait --for=condition=Ready kustomization/models -n flux-system --timeout=300s
```

## Clusterless Quick Tour (5 minutes)

Try the complete local package, harden, compliance, and publish flow with a plain text file. No cluster required.

```bash
mkdir -p ~/test-model
echo "test" > ~/test-model/model.txt
model-cli wizard
```

Key answers: type model name `test-model`, model path `~/test-model`, artifact name `test:v1` (no defaults apply outside the repo). At the cluster prompt, answer **No** — phases 5-7 run as guided simulations.

The test creates these local files:
- `~/test-model/manifest.json` - an OCI manifest carrying CNCF AI Interoperability Profile, SBOM format, and MOF annotations
- `~/test-model/sbom.spdx-json` - the SBOM
- `~/test-model/mof.json` - the MOF metadata

## Known limitations

- The GitOps step assumes a clean git working copy. Generated files are gitignored but some metadata may not be fully deterministic yet.
- The sklearn predictor serves the V1 protocol; V2-style `inputs` payloads are rejected.
- Annotation conventions and model metadata are evolving as part of the CNCF AI inner-loop initiative #1740.

## Documentation

- [Command Reference](docs/command-reference.md)
- [Concepts and Glossary](docs/concepts-and-glossary.md)
- [Architecture](docs/architecture.md)
- [TUI Guide](docs/tui.md)
- [Resources](docs/resources.md)
- [Contributing](CONTRIBUTING.md)
