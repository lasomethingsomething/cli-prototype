# Model CLI

**Your tour guide through the CNCF's "[Cloud Native and OCI Compliant Inner-Loop Tooling & Packaging for AI Engineers](https://github.com/cncf/toc/issues/1740)" initiative.**

## What is this, in plain words?

A trained ML model is a folder of files. Moving it safely from your laptop to production requires packaging, security checks, publishing, and deployment. Specialized tools in the CNCF ecosystem already handle each task.

model-cli is a thin orchestrator: it calls oras, syft, cosign, etc. via your PATH and does not bundle them. Model CLI guides you through that journey, collects the needed details, and lets you choose an appropriate tool for each task.

## The workflow, in seven action-led steps

1. **Develop & Package**: Creates a local OCI artifact manifest from the model folder and tags it with CNCF AI Interoperability Profile metadata.
2. **Local Hardening & Compliance**: Generates an SBOM and MOF classification, and records both in the packaged artifact.
3. **Supply Chain Check**: Cryptographically signs the artifact with Cosign so tampering can be detected later, and records a proof of origin (provenance) for the finished artifact.
4. **Manifest-Level Validation**: Delegates upload to a registry to ORAS or ModelPack.
5. **GitOps Admission & Policy Enforcement**: Commits the artifact manifest to your Git repository and pushes it. A connected Flux installation reconciles the change and rolls out the model with KServe. Does not auto-enforce policy.
6. **Infrastructure & Resource Orchestration**: Validates that infrastructure matches the artifact's runtime and hardware requirements.
7. **Runtime Execution & Optimization**: Validates serving runtime availability (vLLM, KServe). When deployed through the GitOps path, the KServe InferenceService is real and serving.

If terms like SBOM, MOF, or admission are new to you, see the [Concepts and Glossary](docs/concepts-and-glossary.md).

> **Status: prototype.** Model CLI guides ML-model packaging and delivery as OCI artifacts. It collects intent, writes standardized metadata, and delegates supported operations to selected tools such as ORAS, Syft, Cosign, Flux, and Argo CD. Some wizard stages and integrations remain simulated or incomplete; see Known limitations before relying on a step in production.

## System Requirements

macOS with Homebrew, Git, and an SSH key registered with GitHub are required.

Podman is required for the publish demonstration and local registry. On Apple Silicon (arm64), use the latest version: `brew install podman`. On Intel Macs (x86_64), Podman 6+ does not support the Apple hypervisor; install version 5.x instead: `brew install podman@5` or run `brew extract podman /opt/homebrew/Cellar/podman@5 5.1.2` to pin to a working version.

The publish demonstration uses Podman's Linux VM and the open-source OCI Distribution Registry at `localhost:5000`. It does not require Docker Desktop.

> **Note on the registry vs. the deployment:** the local registry at `localhost:5000` is only used for the publish and verification phases (Steps 3–4). The deployed KServe InferenceService fetches the model file directly from the raw GitHub URL in your repository, so no networking between minikube and the local registry is required.

## Test Drive (about 20 minutes, fully real)

This is the recommended path: a real Flux-managed cluster in minikube, a real local registry, real signing and publishing, and a real KServe deployment serving predictions. Every phase executes against live infrastructure.

No cluster? The wizard falls back to guided simulations for phases 5–7; see the Clusterless Quick Tour at the end.

### 0. Install

Download the pre-built binary from [GitHub Releases](https://github.com/lasomethingsomething/cli-prototype/releases/latest):

```bash
# macOS - map architecture: x86_64→amd64 (Intel), arm64→amd64 (Apple Silicon)
arch=$(uname -m | sed 's/x86_64/amd64/')
curl -sL https://github.com/lasomethingsomething/cli-prototype/releases/download/v0.1.1/model-cli_0.1.1_darwin_${arch}.tar.gz | tar xz
chmod +x model-cli
```

> **Note:** The release tag is `v0.1.1` but the asset filenames have no `v` prefix (e.g., `model-cli_0.1.1_darwin_amd64.tar.gz`). On Intel Macs, `uname -m` reports `x86_64` which the `sed` command maps to `amd64`. On Apple Silicon, `uname -m` reports `arm64` which already matches the asset naming.

For Linux, replace `darwin` with `linux` in the URL.

### 1. Fork, clone, and install prerequisites

The Test Drive needs the repository (sample model `models/iris`, Flux config `clusters/minikube/`, a fork to push to).

```bash
# In GitHub: fork this repository (lasomethingsomething/cli-prototype) to your account, then:
git clone https://github.com/your-username/cli-prototype.git
cd cli-prototype
```

> **Prerequisite:** An SSH key registered with GitHub is required for Flux to access your repository.

model-cli is a thin orchestrator: it delegates to external tools like oras, syft, cosign, flux, podman, kubectl, minikube, and notation. These must all be installed and on your PATH before the cluster steps. Run the following to install any missing tools:

```bash
./model-cli setup --yes
```

> **Note:** Replace `your-username` with your actual GitHub username in all commands below.

**Your working copy must be clean before starting the wizard** (`git status` shows nothing modified). The wizard commits the manifest and pushes; uncommitted changes will make the deploy step fail.

### 2. Start the cluster and bootstrap Flux (optional - the wizard can do this)

If you want to set up the cluster manually before running the wizard, or if you're using an existing cluster:

```bash
# Start minikube with the podman driver (recommended) and at least 6GB memory
# If you have Docker Desktop with <8GB allocated, use --memory=6g
minikube start --driver=podman --cpus=4 --memory=6g

# Bootstrap Flux using SSH transport (requires your SSH key from Step 1)
flux bootstrap git --url=ssh://git@github.com/your-username/cli-prototype.git \
  --branch=main --path=./clusters/minikube
```

> **Driver notes:**
> - On Intel Macs with podman 5.x: use `--driver=podman` (required for podman 5.x compatibility)
> - On Apple Silicon: `podman` is the default driver and works with latest podman
> - If using Docker Desktop with <8GB RAM allocated: use `--memory=6g` instead of `--memory=8g`

The repository contains a bootstrapped Flux cluster configuration in `clusters/minikube/`. The bootstrap uses SSH transport which requires an SSH key registered with GitHub.

> **Bootstrap behavior:** `flux bootstrap git` is idempotent. If it fails partway through (e.g., network interruption), simply re-run the same command. Flux will push any missing commits to your repository, then reconcile the cluster state. You will see output like "already exists" or "already up-to-date" for resources that were successfully created on the first attempt.

> **Storage URI placeholder:** The sample model's InferenceService manifest at `clusters/minikube/apps/demo-iris.yaml` uses `STORAGE_URI_PLACEHOLDER`. After forking, replace this with your fork's raw GitHub URL (e.g., `https://raw.githubusercontent.com/your-username/cli-prototype/main/models/iris/model.joblib`).

Wait until everything is reconciled:

```bash
kubectl get kustomizations -n flux-system
# cert-manager, flux-system, kserve, metrics-server, test-model — all True
```

Flux installs KServe (model serving), cert-manager, and metrics-server, and keeps them synced to the repo.

**Or:** Skip this step entirely — when you run the wizard and it asks "Do you have a Kubernetes cluster?", say No and the wizard will offer to set up minikube and Flux automatically.

### 3. Start the local registry (optional - the wizard can do this)

If you want to start the registry manually:

```bash
podman machine init    # first time only
podman machine start
podman run -d --rm --name model-cli-registry -p 5000:5000 registry:2
curl http://localhost:5000/v2/    # {} means ready
```

**Or:** Skip this step entirely — when you run the wizard and choose to publish to a local Podman registry, the wizard will offer to set it up automatically.

### 4. Run the wizard

```bash
./model-cli wizard
```

Answers that produce a fully real deployment, using the sample model in this repo:
 | Prompt | Answer |
 |---|---|
 | What's your model name? | `iris` |
 | Where are your model files? | `./models/iris` |
 | What should we call the artifact? | `test-model/iris` |
 | Which tool should generate the SBOM? | `syft` |
 | How should the Model Openness Framework class be set? | `auto (recommended)` |
 | Which signing approach should the wizard demonstrate? | `cosign` |
 | Publish this artifact to an OCI registry? | Yes |
 | Where is the OCI registry? | `a local Podman registry at localhost:5000` |
 | Which client should publish the artifact? | `oras` |
 | How would you like to promote the artifact? | `flux` |
 | Do you have a Kubernetes cluster? | **Yes** |
 | Git repository URL | `ssh://git@github.com/your-username/cli-prototype.git` |
 | Manifest path in repo | `clusters/minikube/apps` |
 | Which serving topology should the wizard demonstrate? | `kserve` |

### 5. Verify the deployment

```bash
kubectl get kustomizations -n flux-system        # new revision applied
kubectl get inferenceservice sklearn-iris -n test-model   # READY True

# Forward local port 8080 to the sklearn-iris-predictor deployment
# This allows you to send prediction requests to localhost:8080
kubectl port-forward -n test-model deploy/sklearn-iris-predictor 8080:8080
```

Then in another terminal, send a prediction. Note: the sklearn predictor speaks the **V1 protocol** — use `instances`, not the V2-style `inputs` payload:

```bash
curl -s http://localhost:8080/v1/models/sklearn-iris\:predict \
  -H "Content-Type: application/json" \
  -d '{"instances": [[5.1, 3.5, 1.4, 0.2]]}'
# -> {"predictions":[0]}   (setosa)
```

### What just happened

1. The model folder was packaged as an OCI artifact with AI Interoperability Profile annotations
2. An SBOM (Syft) and MOF classification were generated and attached
3. The artifact was signed (Cosign), verified, and pushed to your local registry (ORAS)
4. The wizard committed the manifest to your git repository and pushed
5. Flux reconciled the new revision and KServe rolled out a new predictor pod
6. The InferenceService serves predictions over the V1 protocol

### Cleanup

```bash
podman stop model-cli-registry
minikube delete
```

## Clusterless Quick Tour (5 minutes)

Try the complete local package, harden, compliance, and publish flow with a plain text file. No real model, registry account, GPU, or cluster required. Without a cluster, the wizard demonstrates phases 5–7 as guided simulations.

```bash
mkdir -p ~/test-model
echo "test" > ~/test-model/model.txt
./model-cli wizard
```

Key answers: model name `test-model`, model path `~/test-model`, artifact name `test:v1`, skip RAG context, SBOM tool `syft`, MOF class `auto`, signing tool `cosign`. At the publish prompt choose Yes with the local Podman registry at `localhost:5000` and the `oras` client. At the GitOps prompt (Step 5), answer **No** when asked whether you have a cluster: phases 5–7 run as guided simulations.

The test creates these local files. When you publish, ORAS uploads the model directory and its manifest annotations to the selected registry:

- `~/test-model/manifest.json` - an OCI manifest carrying CNCF AI Interoperability Profile, SBOM format, and MOF annotations
- `~/test-model/sbom.spdx-json` - the SBOM
- `~/test-model/mof.json` - the MOF metadata

Stop the temporary local registry when the test is complete:

```bash
podman stop model-cli-registry
```

## Known limitations

- The GitOps step assumes a clean git working copy; generated files are now gitignored but SBOM and manifest metadata are not deterministic (timestamps, UUIDs), which still dirties the repo on every run. Commit or clean them before re-deploying.
- The serving-topology recommendation does not yet account for the packaged model's runtime (a sklearn artifact is offered `vllm`).
- The sklearn predictor serves the V1 protocol; V2-style `inputs` payloads are rejected.

Annotation conventions and model metadata are evolving as part of the CNCF AI inner-loop initiative #1740, so the wizard inspects them without enforcing a fixed contract.

## Documentation

- [Command Reference](docs/command-reference.md)
- [Concepts and Glossary](docs/concepts-and-glossary.md)
- [Architecture](docs/architecture.md)
- [TUI Guide](docs/tui.md)
- [Resources](docs/resources.md)
- [Contributing](CONTRIBUTING.md)
