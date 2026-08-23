# GitOps Admission & Policy Enforcement

This document describes how to configure GitOps tools (Argo CD, Flux) to use the **Trust Profile** and **Infrastructure Requirement** annotations attached by Model CLI for admission control and policy enforcement (Stories #63, #64, and #65).

## Overview

Model CLI ensures that AI artifacts pushed to registries include annotations in their OCI manifests that enable GitOps controllers and policy engines to enforce admission policies before deploying artifacts to your cluster.

**Key Principle:** Model CLI **orchestrates and hands off** to external tools. The CLI attaches metadata and provides pre-flight validation, but **does not implement admission logic itself**. Policy enforcement is the responsibility of:
- GitOps controllers (Argo CD, Flux)
- Policy engines (Sigstore Policy Controller, OPA/Gatekeeper, Kyverno)

### How It Works

1. **Package & Push (Phase 2):** Model CLI attaches Trust Profile and Infrastructure Requirement annotations to OCI manifests (Stories #63, #64)
2. **Pre-Flight Validation (Phase 3):** Model CLI provides `validate-gitops` command to check annotations before deployment (Story #65)
3. **GitOps Deployment:** Argo CD or Flux deploys artifacts, triggering admission webhooks
4. **Policy Enforcement:** External policy engines validate annotations and enforce admission policies

## Pre-Sync Validation (Story #65)

The `model-cli validate-gitops` command provides pre-flight validation of artifact annotations before GitOps tools attempt deployment. This allows you to catch missing metadata or compliance issues early in your CI/CD pipeline.

### When to Use

- **CI/CD Pipelines:** Validate artifacts before promoting to production
- **GitOps PreSync Hooks:** Use as a custom hook in Argo CD or Flux
- **Local Testing:** Verify artifacts before deploying
- **Air-Gapped Environments:** Pre-validate artifacts before they enter the air-gapped cluster

### Usage

```bash
# Basic validation
model-cli validate-gitops --artifact ghcr.io/my-org/my-model:v1.0.0

# With specific registry tool
model-cli validate-gitops --artifact ghcr.io/my-org/my-model:v1.0.0 --registry oras

# Quiet mode (just pass/fail exit code)
model-cli validate-gitops --artifact ghcr.io/my-org/my-model:v1.0.0 --quiet

# JSON output for CI/CD integration
model-cli validate-gitops --artifact ghcr.io/my-org/my-model:v1.0.0 --json-output
```

### Exit Codes

| Exit Code | Meaning | Use Case |
|-----------|---------|----------|
| 0 | All required annotations present | Deployment can proceed |
| 1 | Missing required annotations | Block deployment, check output |

### Environment-Specific Validation (Story #66)

The `validate-gitops` command supports environment-specific safety policy validation:

#### Air-Gapped Environments

Validates that artifacts are suitable for air-gapped deployment:
- Checks packaging format (recommends `modelpack` for air-gapped)
- Verifies SBOM is present for compliance
- Reminds to pre-load dependencies

```bash
model-cli validate-gitops --artifact my-registry/my-model:v1 --env air-gapped
```

#### Hybrid-Cloud Environments

Validates data residency and network requirements for hybrid-cloud:
- Checks data residency annotation matches target region
- Validates network access requirements (internal/private/public)

```bash
model-cli validate-gitops --artifact my-registry/my-model:v1 --env hybrid-cloud --region us-east-1
```

#### Standard Environments

For development, staging, and production environments, the command simply acknowledges the validation:

```bash
model-cli validate-gitops --artifact my-registry/my-model:v1 --env production
```

### Example Output

```
Validating GitOps annotations for: ghcr.io/my-org/my-model:v1.0.0

✓ Fetched artifact manifest from registry

=== Trust Profile Validation ===
  ✓ org.cncf.ai.security.signing.framework: sigstore-cosign
  ✓ org.cncf.ai.security.sbom.format: spdx-json
  ✓ org.cncf.ai.security.provenance.type: slsa-v1.0
  ✓ org.cncf.ai.interop.profile.version: 1.0.0
  ✓ org.cncf.ai.artifact.type: model

=== Infrastructure Requirements Validation ===
  ✓ org.cncf.ai.runtime: vllm
  ✓ org.cncf.ai.accelerator: nvidia-gpu
  ✓ org.cncf.ai.accelerator.cuda.min: 12.1
  ✓ org.cncf.ai.resource.memory.min: 24GiB

=== Result ===
✓ PASS: Artifact has all required annotations for GitOps admission

GitOps tools (Argo CD, Flux) can proceed with deployment.
External policy engines (Sigstore Policy Controller, OPA/Gatekeeper, Kyverno)
will perform the actual admission control based on these annotations.
```

### Using with Argo CD PreSync Hooks

Create a ConfigMap with a validation script that uses `model-cli validate-gitops`:

```yaml
# gitops-validation-hook.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: validate-model-annotations
  namespace: argocd
  labels:
    argocd.argoproj.io/hook: PreSync
    argocd.argoproj.io/hook-delete-policy: HookSucceeded
data:
  validate.sh: |
    #!/bin/bash
    set -euo pipefail
    
    ARTIFACT=$1
    
    echo "Validating GitOps annotations for $ARTIFACT..."
    model-cli validate-gitops --artifact $ARTIFACT --registry oras --quiet
    
    if [ $? -ne 0 ]; then
      echo "FAIL: Artifact $ARTIFACT is missing required annotations"
      exit 1
    fi
    
    echo "PASS: Artifact $ARTIFACT validated"
```

Reference this hook in your Argo CD Application:

```yaml
# argocd-application.yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: my-model
  namespace: argocd
  annotations:
    avp.kubevirtual.com/pre-sync-hooks: |
      - name: validate-annotations
        kind: ConfigMap
        name: validate-model-annotations
        namespace: argocd
spec:
  # ... your application spec
```

### Using with Flux Image Automation

Use `model-cli validate-gitops` in your Flux image policy:

```yaml
# image-policy.yaml
apiVersion: image.toolkit.fluxcd.io/v1beta1
kind: ImagePolicy
metadata:
  name: model-validation
  namespace: flux-system
spec:
  imageRepositoryRef:
    name: my-model
    namespace: flux-system
  policy:
    numerical:
      order: asc
  # Add validation step in your Flux automation
```

For full validation with Flux, combine with a Kustomization that runs the validation:

```bash
# In your CI/CD pipeline
model-cli validate-gitops --artifact $ARTIFACT --json-output
# Parse JSON output and only proceed if status is "pass"

# Or use a Flux ImageRepository with custom health checks
```

## Annotations Reference

Model CLI attaches two categories of annotations to every OCI artifact manifest:

### 1. Trust Profile Annotations (Story #63)

## Trust Profile Annotations

Model CLI attaches the following annotations to every OCI artifact manifest:

### Security Annotations (Trust Profile)

| Annotation | Description | Example Value | Required for Admission |
|------------|-------------|---------------|------------------------|
| `org.cncf.ai.security.signing.framework` | Signing framework used | `sigstore-cosign`, `notation` | ✅ Yes |
| `org.cncf.ai.security.sbom.format` | SBOM format | `spdx-json`, `cyclonedx` | ✅ Yes |
| `org.cncf.ai.security.provenance.type` | Provenance type | `slsa-v1.0`, `in-toto` | ✅ Yes |

### MOF (Model Openness Framework) Annotations

| Annotation | Description | Example Value | Required for Admission |
|------------|-------------|---------------|------------------------|
| `org.cncf.ai.model.mof.class` | Openness class | `I`, `II`, `III` | ⚠️ Depends on policy |
| `org.cncf.ai.model.mof.version` | MOF spec version | `1.0` | ⚠️ Depends on policy |
| `org.cncf.ai.model.mof.components` | Open components | `weights,training-data,code` | ⚠️ Depends on policy |

### Profile Annotations

| Annotation | Description | Example Value | Required |
|------------|-------------|---------------|----------|
| `org.cncf.ai.interop.profile.version` | Interoperability profile version | `1.0.0` | ✅ Yes |
| `org.cncf.ai.artifact.type` | Artifact type | `model`, `skill`, `pipeline` | ✅ Yes |

### 2. Infrastructure Requirement Annotations (Story #64)

These annotations allow policy engines to verify that the artifact's infrastructure requirements match the destination environment.

| Annotation | Description | Example Value | Required |
|------------|-------------|---------------|----------|
| `org.cncf.ai.runtime` | Runtime for serving | `vllm`, `kserve`, `tensorrt-llm` | ✅ Yes |
| `org.cncf.ai.accelerator` | Hardware accelerator requirement | `nvidia-gpu`, `amd-gpu`, `cpu` | ✅ Yes |
| `org.cncf.ai.accelerator.cuda.min` | Minimum CUDA version | `12.1`, `11.8` | ⚠️ Conditional |
| `org.cncf.ai.resource.memory.min` | Minimum memory requirement | `24GiB`, `16Gi` | ⚠️ Conditional |

## Configuration Options

There are several ways to configure GitOps admission using Trust Profile and Infrastructure Requirement annotations:

### Infrastructure Requirement Matching (Story #64)

The infrastructure requirement annotations allow policy engines to verify that an artifact can run in the destination cluster. For example:
- A model requiring `nvidia-gpu` should not be deployed to a CPU-only cluster
- A model requiring `vllm` runtime should only be deployed to clusters with vLLM installed
- A model requiring CUDA 12.1+ should only be deployed to nodes with compatible GPUs

#### Example: Kyverno Policy for GPU Requirement

```yaml
# kyverno-gpu-requirement.yaml
apiVersion: kyverno.io/v1
kind: ClusterPolicy
metadata:
  name: require-gpu-capability
  annotations:
    policies.kyverno.io/title: Require GPU Capability
    policies.kyverno.io/category: Workload Management
    policies.kyverno.io/severity: medium
spec:
  validationFailureAction: enforce
  background: true
  rules:
  - name: check-gpu-requirement
    match:
      any:
      - resources:
          kinds:
          - Pod
    context:
    - name: nodeSelector
      apiCall:
        url: "https://kubernetes.default.svc/api/v1"
        method: GET
        path: "/nodes"
    validate:
      message: "Pod requires GPU but cluster has no GPU nodes"
      pattern:
        spec:
          containers:
          - image: "*"
        metadata:
          annotations:
            org.cncf.ai.accelerator: "nvidia-gpu"
    # Note: Full node capability checking requires custom Rego logic
```

For production use, consider using a mutating webhook that:
1. Reads the artifact's infrastructure requirement annotations from the manifest
2. Compares them against cluster capabilities (node labels, custom resources)
3. Blocks admission if requirements don't match

#### Example: OPA/Gatekeeper Policy for Runtime Matching

```yaml
# runtime-matching-constraint.yaml
apiVersion: constraints.gatekeeper.sh/v1beta1
kind: RuntimeMatching
metadata:
  name: require-runtime-availability
spec:
  match:
    kinds:
    - apiGroups: [""]
      kinds: ["Pod"]
  parameters:
    requiredRuntime: "vllm"
  # Rego logic would check if the cluster has the required runtime available
```

## Configuration Options

There are several ways to configure GitOps admission using Trust Profile and Infrastructure Requirement annotations:

### Option 1: Sigstore Policy Controller (Recommended for Signature Verification)

[Sigstore Policy Controller](https://docs.sigstore.dev/policy-controller/) is a Kubernetes admission controller that verifies Sigstore signatures on container images and OCI artifacts.

#### Installation

```bash
# Install using Helm
helm repo add sigstore https://sigstore.github.io/helm-charts
helm repo update
helm install policy-controller sigstore/policy-controller \
  --namespace cosign-system --create-namespace \
  --set cosign.triage=true
```

#### Configure ClusterImagePolicy

Create a `ClusterImagePolicy` that requires signatures and checks for Trust Profile annotations:

```yaml
# cosign-cluster-image-policy.yaml
apiVersion: policy.sigstore.dev/v1beta1
kind: ClusterImagePolicy
metadata:
  name: model-cli-trust-profile
spec:
  images:
  - glob: "ghcr.io/my-org/*"
  authorities:
  - name: model-cli-signer
    # Require signature from your Sigstore keypair
    keyless:
      url: "https://sigstore-tuf-root.staging.sigstore.dev"
      identities:
      - issuer: "https://token.actions.githubusercontent.com"
        subject: "https://github.com/lasomethingsomething/cli-prototype/.github/workflows/*"
    # Require Trust Profile annotations
    annotations:
      required:
        - "org.cncf.ai.security.signing.framework"
        - "org.cncf.ai.security.sbom.format"
        - "org.cncf.ai.security.provenance.type"
        - "org.cncf.ai.interop.profile.version"
        - "org.cncf.ai.artifact.type"
    signers:
    - "https://github.com/lasomethingsomething/cli-prototype/.github/workflows/release.yml@refs/heads/main"
```

Apply the policy:

```bash
kubectl apply -f cosign-cluster-image-policy.yaml
```

#### How It Works

1. Model CLI pushes artifact with Trust Profile annotations to registry
2. Argo CD/Flux attempts to deploy the artifact
3. Kubernetes API server calls Sigstore Policy Controller admission webhook
4. Policy Controller:
   - Verifies the artifact has a valid Sigstore signature
   - Checks that all required Trust Profile annotations are present
   - Blocks admission if any check fails
5. Clear error messages are returned if validation fails

### Option 2: OPA/Gatekeeper

[OPA/Gatekeeper](https://open-policy-agent.github.io/gatekeeper/website/) is a policy engine that can enforce custom admission policies.

#### Installation

```bash
# Install Gatekeeper
kubectl apply -f https://raw.githubusercontent.com/open-policy-agent/gatekeeper/main/deploy/gatekeeper.yaml
```

#### Create ConstraintTemplate for Trust Profile

```yaml
# trust-profile-constraint-template.yaml
apiVersion: templates.gatekeeper.sh/v1
kind: ConstraintTemplate
metadata:
  name: require-trust-profile
spec:
  crd:
    spec:
      names:
        kind: RequireTrustProfile
      validation:
        # Schema for the constraint's CRD
        openAPIV3Schema:
          type: object
          properties:
            requiredAnnotations:
              type: array
              items: string
  targets:
  - target: admission.k8s.gatekeeper.sh
    rego: |
      package kubernetes.admission

      violation[{"msg": msg, "details": {"missing_annotations": missing}}] {
        container := input.review.object.spec.template.spec.containers[_]
        image := container.image
        
        # Get the image manifest annotations (requires image policy webhook or similar)
        # For demonstration, we check pod annotations which could be populated by a mutating webhook
        pod_annotations := input.review.object.metadata.annotations
        
        required := {"org.cncf.ai.security.signing.framework", "org.cncf.ai.security.sbom.format", "org.cncf.ai.security.provenance.type"}
        
        missing := {ann | ann := required[_]; not pod_annotations[ann]}
        count(missing) > 0
        
        msg := sprintf("Pod is missing required Trust Profile annotations: %v", [missing])
      }
```

#### Create Constraint

```yaml
# require-trust-profile-constraint.yaml
apiVersion: constraints.gatekeeper.sh/v1beta1
kind: RequireTrustProfile
metadata:
  name: model-cli-trust-profile
spec:
  match:
    kinds:
    - apiGroups: [""]
      kinds: ["Pod"]
    namespaces:
    - "*"
  parameters:
    requiredAnnotations:
    - "org.cncf.ai.security.signing.framework"
    - "org.cncf.ai.security.sbom.format"
    - "org.cncf.ai.security.provenance.type"
    - "org.cncf.ai.interop.profile.version"
    - "org.cncf.ai.artifact.type"
```

Apply the constraint:

```bash
kubectl apply -f trust-profile-constraint-template.yaml
kubectl apply -f require-trust-profile-constraint.yaml
```

#### Using Kyverno Instead

[Kyverno](https://kyverno.io/) is a simpler alternative to OPA/Gatekeeper:

```bash
# Install Kyverno
kubectl create -f https://releases.kyverno.io/artifacts/latest/install.sh | sh
```

Create a ClusterPolicy:

```yaml
# kyverno-trust-profile-policy.yaml
apiVersion: kyverno.io/v1
kind: ClusterPolicy
metadata:
  name: require-trust-profile
  annotations:
    policies.kyverno.io/title: Require Trust Profile
    policies.kyverno.io/category: Security
    policies.kyverno.io/severity: medium
spec:
  validationFailureAction: enforce
  background: true
  rules:
  - name: check-trust-profile-annotations
    match:
      any:
      - resources:
          kinds:
          - Pod
    validate:
      message: "Pod is missing required Trust Profile annotations"
      pattern:
        metadata:
          annotations:
            org.cncf.ai.security.signing.framework: "?*"
            org.cncf.ai.security.sbom.format: "?*"
            org.cncf.ai.security.provenance.type: "?*"
            org.cncf.ai.interop.profile.version: "?*"
            org.cncf.ai.artifact.type: "?*"
```

Apply the policy:

```bash
kubectl apply -f kyverno-trust-profile-policy.yaml
```

### Option 3: Argo CD Image Updater with Admission Policies

[Argo CD Image Updater](https://argocd-image-updater.readthedocs.io/) can be configured with custom health checks and admission policies.

#### Configure Argo CD for Trust Profile Validation

1. **Install Argo CD Image Updater:**

```bash
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj-labs/argocd-image-updater/stable/manifests/install.yaml
```

2. **Create a Custom Admission Policy:**

Argo CD supports [custom resource hooks](https://argo-cd.readthedocs.io/en/stable/operator-manual/custom-tools/) that can be used for validation.

Create a ConfigMap with a validation script:

```yaml
# trust-profile-validation-hook.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: trust-profile-validation
  namespace: argocd
  labels:
    argocd.argoproj.io/hook: PreSync
    argocd.argoproj.io/hook-delete-policy: HookSucceeded
data:
  validate.sh: |
    #!/bin/bash
    set -euo pipefail
    
    # Get the image reference from the Argo CD application manifest
    IMAGE=$1
    
    # Use ORAS or similar to fetch manifest annotations
    # This is a placeholder - in production, use your registry's API
    echo "Validating Trust Profile for $IMAGE"
    
    # Check for required annotations (simplified example)
    # In production, parse the manifest and check annotations
    if ! oras manifest fetch "$IMAGE" | jq -e '.annotations | has("org.cncf.ai.security.signing.framework")' > /dev/null 2>&1; then
      echo "FAIL: Missing signing framework annotation"
      exit 1
    fi
    
    if ! oras manifest fetch "$IMAGE" | jq -e '.annotations | has("org.cncf.ai.security.sbom.format")' > /dev/null 2>&1; then
      echo "FAIL: Missing SBOM format annotation"
      exit 1
    fi
    
    if ! oras manifest fetch "$IMAGE" | jq -e '.annotations | has("org.cncf.ai.security.provenance.type")' > /dev/null 2>&1; then
      echo "FAIL: Missing provenance type annotation"
      exit 1
    fi
    
    echo "PASS: Trust Profile validated"
```

3. **Reference the hook in your Argo CD Application:**

```yaml
# argocd-application.yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: my-model
  namespace: argocd
  annotations:
    avp.kubevirtual.com/pre-sync-hooks: |
      - name: validate-trust-profile
        kind: ConfigMap
        name: trust-profile-validation
        namespace: argocd
spec:
  source:
    repoURL: https://github.com/your-org/model-manifests.git
    path: manifests
    targetRevision: main
  destination:
    server: https://kubernetes.default.svc
    namespace: production
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
```

### Option 4: Flux with Image Policy

[Flux](https://fluxcd.io/) supports [image automation](https://fluxcd.io/flux/components/image/) with policies.

#### Install Flux Image Automation

```bash
flux create source git model-manifests \
  --url=https://github.com/your-org/model-manifests \
  --branch=main \
  --interval=5m

flux create image policy my-model \
  --image-ref=ghcr.io/my-org/my-model \
  --select-semver=">=1.0.0" \
  --author-name=my-org \
  --export > ./clusters/my-cluster/image-policy.yaml
```

#### Configure Flux for Trust Profile Validation

Flux can use [Image Verification](https://fluxcd.io/flux/components/image/verification/) with Cosign:

```yaml
# image-repository.yaml
apiVersion: image.toolkit.fluxcd.io/v1beta2
kind: ImageRepository
metadata:
  name: my-model
  namespace: flux-system
spec:
  image: ghcr.io/my-org/my-model
  interval: 5m
  provider: generic
  verification:
    provider: cosign
    secretRef:
      name: cosign-public-key
```

Create a secret with your Cosign public key:

```bash
kubectl create secret generic cosign-public-key \
  --namespace=flux-system \
  --from-file=cosign.pub=cosign.pub
```

#### Use ImagePolicy with Annotation Checks

Create a custom ImagePolicy that checks for Trust Profile annotations:

```yaml
# image-policy.yaml
apiVersion: image.toolkit.fluxcd.io/v1beta1
kind: ImagePolicy
metadata:
  name: model-trust-policy
  namespace: flux-system
spec:
  imageRepositoryRef:
    name: my-model
    namespace: flux-system
  filterTags:
    pattern: "^[0-9]+\.[0-9]+\.[0-9]+(-[a-z0-9]+)?$"
  policy:
    numerical:
      order: asc
  # Note: Flux doesn't natively check OCI manifest annotations
  # For annotation checking, use Flux + OPA/Gatekeeper together
```

For full Trust Profile validation with Flux, combine with OPA/Gatekeeper as shown in Option 2.

## Example: Complete GitOps Workflow with Trust Profile

### 1. Package and Push with Model CLI

```bash
# Package model with Trust Profile annotations
model-cli package --model my-model --model-path ./models \
  --model-type llm --framework pytorch \
  --sign --signer sigstore

# Push to registry (annotations are attached to manifest)
model-cli push --artifact my-model:v1.0.0 \
  --registry oras \
  --destination ghcr.io/my-org
```

### 2. Configure GitOps with Policy Enforcement

```bash
# Install Sigstore Policy Controller
helm install policy-controller sigstore/policy-controller \
  --namespace cosign-system --create-namespace

# Apply Trust Profile policy
kubectl apply -f cosign-cluster-image-policy.yaml
```

### 3. Deploy with Argo CD

```bash
# Create Argo CD application
argocd app create my-model \
  --repo https://github.com/your-org/model-manifests.git \
  --path ./manifests \
  --dest-namespace production \
  --dest-server https://kubernetes.default.svc

# Argo CD will:
# 1. Detect new version in git
# 2. Kubernetes API calls Sigstore Policy Controller webhook
# 3. Policy Controller verifies signature + Trust Profile annotations
# 4. If valid: deployment proceeds
# 5. If invalid: admission blocked with clear error message
```

### 4. Deploy with Flux

```bash
# Create Flux Kustomization
flux create kustomization my-model \
  --source-ref=main \
  --source-name=model-manifests \
  --path=./manifests \
  --prune=true \
  --validation=client \
  --health-checks=true \
  --interval=5m \
  --export > ./clusters/my-cluster/kustomization.yaml

# Flux will:
# 1. Monitor git repo for changes
# 2. Attempt to apply manifests
# 3. Kubernetes API calls Sigstore Policy Controller webhook
# 4. Policy Controller verifies Trust Profile + Infrastructure Requirements
# 5. If valid: manifests applied
# 6. If invalid: reconciliation fails with error
```

### 5. Deploy with Infrastructure Matching

For clusters with specific capabilities:

```bash
# Package with specific infrastructure requirements
model-cli package --model my-model \
  --runtime vllm \
  --accelerator nvidia-gpu \
  --cuda-min 12.1

model-cli push --artifact my-model:v1.0.0 \
  --registry oras \
  --destination ghcr.io/my-org

# The manifest now includes:
# - org.cncf.ai.runtime: vllm
# - org.cncf.ai.accelerator: nvidia-gpu
# - org.cncf.ai.accelerator.cuda.min: 12.1

# Configure policy to check infrastructure matching
kubectl apply -f kyverno-gpu-requirement.yaml
kubectl apply -f runtime-matching-constraint.yaml
```

## Troubleshooting

### Admission Blocked: Missing Annotations

**Error:**
```
Error from server: admission webhook "policy.sigstore.dev" denied the request: \
image ghcr.io/my-org/my-model:v1.0.0 is not signed with a trusted key and \
missing required annotation: org.cncf.ai.security.signing.framework
```

**Solution:**
```bash
# Re-package with signing enabled
model-cli package --model my-model --sign --signer sigstore

# Re-push
model-cli push --artifact my-model:v1.0.0 --registry oras --destination ghcr.io/my-org
```

### Admission Blocked: Invalid Signature

**Error:**
```
Error from server: admission webhook "policy.sigstore.dev" denied the request: \
signature verification failed for ghcr.io/my-org/my-model:v1.0.0
```

**Solution:**
```bash
# Verify the signature locally first
model-cli verify --artifact ghcr.io/my-org/my-model:v1.0.0

# If verification fails, re-sign
model-cli sign --artifact ghcr.io/my-org/my-model:v1.0.0 --signer sigstore --force
```

### Policy Controller Not Intercepting Requests

**Error:** Admission requests bypass the policy controller.

**Solution:**
```bash
# Check if webhook is registered
kubectl get validatingwebhookconfigurations

# Check policy controller logs
kubectl logs -n cosign-system -l app=policy-controller

# Ensure your GitOps tool is deploying to the correct namespace
# and the webhook is configured to intercept requests for that namespace
```

### Admission Blocked: Incompatible Infrastructure

**Error:**
```
Error from server: admission webhook "kyverno.svc" denied the request: \
resource Pod/default/my-model-pod was blocked due to the following policies
require-gpu-capability: validation error: Pod requires GPU but cluster has no GPU nodes
```

**Solution:**
```bash
# Option 1: Deploy to a cluster with compatible infrastructure
# Use a cluster with NVIDIA GPU nodes

# Option 2: Package for CPU-only deployment
model-cli package --model my-model --accelerator cpu
model-cli push --artifact my-model:v1.0.0-cpu

# Option 3: Update your policy to allow CPU fallbacks
# Modify your Kyverno policy to allow CPU when GPU is not available
```

### Admission Blocked: Missing Runtime

**Error:**
```
Error from server: admission webhook "gatekeeper-validating-webhook-configuration" denied the request: \
constraint RuntimeMatching violated: Pod requires vllm runtime but cluster doesn't have it
```

**Solution:**
```bash
# Option 1: Install the required runtime in your cluster
# Install vLLM, KServe, etc.

# Option 2: Package for a different runtime
model-cli package --model my-model --runtime kserve
model-cli push --artifact my-model:v1.0.0-kserve

# Option 3: Update your GitOps manifests to include runtime installation
# Use Kustomize or Helm to install runtime dependencies
```

## Best Practices

### 1. Always Sign Before Pushing

```bash
model-cli package --model my-model --sign --signer sigstore
model-cli push --artifact my-model:v1.0.0
```

### 2. Use Multiple Policy Layers

- **Sigstore Policy Controller:** Signature + annotation verification
- **OPA/Gatekeeper:** Custom business logic (MOF class, registry allowlists)
- **Kyverno:** Simple annotation presence checks

### 3. Test Policies Before Enforcement

```bash
# Dry-run with Sigstore Policy Controller
kubectl apply -f cosign-cluster-image-policy.yaml --dry-run=client

# Test with a sample pod
kubectl create --dry-run=client -oyaml -f test-pod.yaml
```

### 4. Monitor Policy Decisions

```bash
# View policy controller audit logs
kubectl logs -n cosign-system -l app=policy-controller -f

# View Argo CD sync status (will show admission failures)
argocd app get my-model

# View Flux reconciliation status
flux get kustomizations
```

### 5. Start with Audit Mode

Before enforcing policies in production, run in audit mode:

```yaml
# For Sigstore Policy Controller
apiVersion: policy.sigstore.dev/v1beta1
kind: ClusterImagePolicy
metadata:
  name: model-cli-trust-profile-audit
spec:
  # ... same spec as before ...
  # Add this to run in audit mode (doesn't block, just logs)
  mode: audit
```

## Summary

| Tool | Purpose | Configuration | Notes |
|------|---------|---------------|-------|
| Sigstore Policy Controller | Signature + Trust Profile verification | `ClusterImagePolicy` | Best for signature-based trust |
| OPA/Gatekeeper | Custom admission policies | `ConstraintTemplate` + `Constraint` | Most flexible, supports Rego |
| Kyverno | Simple policy enforcement | `ClusterPolicy` | Easier YAML-based policies |
| Argo CD | GitOps deployment | Custom hooks | Use with PreSync hooks |
| Flux | GitOps deployment | `ImagePolicy` + Cosign | Use with ImageRepository |
| Custom Webhook | Infrastructure matching | Custom admission controller | Best for cluster-specific logic |

**Model CLI's Role:**
- ✅ Attach Trust Profile annotations to OCI manifests (Story #63)
- ✅ Attach Infrastructure Requirement annotations to OCI manifests (Story #64)
- ✅ Provide pre-flight validation for GitOps deployment (Story #65)
- ✅ Validate air-gapped and hybrid-cloud safety policies (Story #66)
- ✅ Pass artifact references to GitOps tools
- ✅ Document how to configure policy enforcement
- ❌ Does NOT implement admission webhooks
- ❌ Does NOT verify signatures at deployment time
- ❌ Does NOT enforce policies

**External Tools' Role:**
- ✅ Verify signatures (Sigstore Policy Controller, Cosign)
- ✅ Enforce admission policies (OPA/Gatekeeper, Kyverno)
- ✅ Deploy artifacts (Argo CD, Flux)
- ✅ Access OCI manifest annotations from registries

## See Also

- [Sigstore Policy Controller](https://docs.sigstore.dev/policy-controller/)
- [OPA/Gatekeeper](https://open-policy-agent.github.io/gatekeeper/website/)
- [Kyverno](https://kyverno.io/)
- [Argo CD](https://argo-cd.readthedocs.io/)
- [Flux](https://fluxcd.io/)
- [Cosign](https://docs.sigstore.dev/cosign/overview/)
- [Model CLI Trust Profile Annotations](#trust-profile-annotations)
