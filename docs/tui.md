# TUI (Terminal User Interface)

Model CLI provides a clean, color-coded TUI built with [huh](https://github.com/charmbracelet/huh), [lipgloss](https://github.com/charmbracelet/lipgloss), and [bubbletea](https://github.com/charmbracelet/bubbletea) for advanced layouts.

## Design Principles

- **Clean and uncluttered** - Only show key information
- **Step-by-step guidance** - Clear workflow progression
- **Color-coded** - Success (green), warnings (yellow), info (blue)
- **Context panels** - side panels showing important info
- **Consistent** - Same look and feel across all commands

## Styling

| Style | Color | Usage |
|-------|-------|-------|
| Title | White (#FAFAFA), Bold | Command/completion headers |
| Subtitle | Gray (#999999) | Descriptive text |
| Step | Blue (#55AAFF), Bold | Workflow step indicators |
| Success | Green (#00FF88), Bold | Success messages with checkmarks |
| Info | Light Blue (#8888FF) | Information and tips |
| Warning | Orange (#FFAA00) | Warnings and issues |

## Context Side Panels

The Model CLI TUI features **context panels**. These panels appear at key points in the workflow and display important contextual information in an organized, easy-to-read format.

### Panel Layout

```
+----------------------------------------+
| Main Content Area                     |
| (huh prompts and workflow steps)       |
+----------------------------------------+

+------------------+
| Context Panel    |
|==================|
| Progress         |
|  Step 2/7 57%   |
|  [███████░░░░]  |
|                  |
| Configuration    |
|  Registry: oras  |
|  GitOps: argo    |
|  Signer: cosign  |
|                  |
| Model            |
|  Name: phi-4-mini|
|  Artifact: v1    |
|                  |
| Status           |
|  [x] Package     |
|  [x] Sign       |
|                  |
| Recent Logs      |
|  - Package done  |
+------------------+
```

### Panel Sections

1. **Progress** - Shows current step, total steps, and visual progress bar
2. **Configuration** - Displays selected tools (registry, GitOps, signer, runtime)
3. **Model** - Shows model name, path, and artifact name
4. **Status** - Checkmarks for completed workflow steps
5. **Recent Logs** - Last 3 log entries from the workflow

### Hardening TUI (Step 2)

The **Local Hardening & Compliance** workflow (`internal/tui/harden/`) features specialized tabs:

1. **Progress** - Step progress through hardening workflow
2. **Config** - Tool configuration
3. **Model** - Model information
4. **SBOM** - SBOM generation status, path, and tool info
5. **MOF** - MOF classification status, class, and version
6. **Logs** - Recent hardening operation logs
7. **Help** - Keyboard shortcuts
8. **Env** - Environment information

The hardening TUI provides:
- Interactive panels explaining SBOM and MOF concepts
- Real-time progress feedback during SBOM generation and MOF classification
- Results summary with SBOM path and MOF class
- Security annotations application status

## Interactive vs Non-Interactive Modes

Every command that supports interactive prompts also supports **non-interactive mode** via command-line flags. This provides parity between the two modes.

### Using Non-Interactive Mode

All interactive commands accept flags that correspond to their prompts. For automation (CI/CD, scripts), use flags instead of interactive input.

**Example:**
```bash
# Interactive (prompts for model name)
model-cli package

# Non-interactive (all values via flags)
model-cli package --model phi-4-mini --model-path ./models --artifact my-model:v1
```

## Command-Specific TUI

### wizard

The full guided journey with 7 steps:

1. **Setup Preferences** - Choose registry, GitOps, and signing tools
2. **Model Details** - Enter model name, path, artifact name, RAG info
3. **Kubernetes Setup** - Check for cluster availability
4. **Package** - Creates OCI artifact with SBOM and MOF
5. **Sign** - Signs artifact with chosen provider
6. **Verify** - Validates the signature
7. **Deploy** - Deploys to Kubernetes (if available)

**Skip flags:**
- `--skip-signing` - Skip steps 5-6
- `--skip-deploy` - Skip step 7

### package

Packages a model as an OCI artifact.

**Interactive prompts:**
- Registry tool (oras/modelpack)
- Model name
- Model path
- Artifact name
- Registry URL
- Include RAG context
- RAG context path
- Runtime (vllm/kserve)
- Accelerator (nvidia-gpu/amd-gpu/intel-gpu/cpu/none)
- Minimum CUDA version
- Minimum memory
- MOF Class (I/II/III)
- MOF Components

**Non-interactive flags:**
```bash
model-cli package \
  --registry oras \
  --model phi-4-mini \
  --model-path ./models \
  --artifact my-model:v1 \
  --registry-url ghcr.io/myorg \
  --runtime vllm \
  --accelerator nvidia-gpu \
  --cuda-min 12.1 \
  --memory-min 24GiB \
  --mof-class I \
  --mof-components "weights,training-data"
```

### sign

Signs an OCI artifact.

**Interactive prompts:**
- Artifact to sign
- Signing tool (sigstore/notary)
- Use specific key
- Key reference

**Non-interactive flags:**
```bash
model-cli sign \
  --artifact my-model:v1 \
  --signer sigstore \
  --key cosign-key.pub
```

### verify

Verifies an artifact signature.

**Interactive prompts:**
- Artifact to verify
- Signing tool used

**Non-interactive flags:**
```bash
model-cli verify \
  --artifact my-model:v1 \
  --signer sigstore
```

### deploy

Deploys to Kubernetes using GitOps.

**Interactive prompts:**
- GitOps tool (argo/flux)
- Registry tool (oras/modelpack)
- Model name
- Has Kubernetes cluster
- Git repository URL
- Manifest path

**Non-interactive flags:**
```bash
model-cli deploy \
  --gitops argo \
  --registry oras \
  --model my-model \
  --repo https://github.com/org/manifests \
  --path ./manifests
```

### push

Pushes an OCI artifact to a registry.

**Interactive prompts:**
- Artifact to push
- Registry tool (oras/modelpack)
- Destination registry

**Non-interactive flags:**
```bash
model-cli push \
  --artifact my-model:v1 \
  --registry oras \
  --destination ghcr.io/myorg
```

### serve

Serves a model using vLLM or KServe.

**Interactive prompts:**
- Runtime (vllm/kserve)
- Model path
- Host
- Port
- Model size
- Load agentic skills
- Skill reference

**Non-interactive flags:**
```bash
model-cli serve \
  --runtime vllm \
  --model-path ./models/phi-4-mini \
  --host 0.0.0.0 \
  --port 8080 \
  --model-size 14GB
```

### admit

Evaluates an artifact for GitOps admission.

**Interactive prompts:**
- Artifact to evaluate
- Target environment

**Non-interactive flags:**
```bash
model-cli validate admission \
  --artifact my-registry/my-model:latest \
  --env production \
  --strict
```

### validate

Validates an OCI artifact manifest.

**Interactive prompts:**
- Artifact to validate
- Check relationships

**Non-interactive flags:**
```bash
model-cli validate \
  --artifact my-registry/my-model:latest \
  --check-relationships
```

### schedule

Schedules a model workload to appropriate Kubernetes node.

**Interactive prompts:**
- Artifact to schedule

**Non-interactive flags:**
```bash
model-cli schedule \
  --artifact my-registry/my-model:latest \
  --list-nodes \
  --dry-run
```

### harden

Performs local hardening and compliance checks (SBOM + MOF).

**Interactive panels:**
- Introduction explaining hardening concepts
- SBOM generation configuration with explanations
- MOF classification configuration with class descriptions
- Review of all hardening options
- Progress display during hardening operations

**Features:**
- Generates SBOM using Syft and attaches to artifact layers
- Applies MOF classification (Class I, II, or III)
- Applies security annotations (Sigstore, SLSA)
- Context panel with SBOM, MOF, and progress tabs

**Non-interactive flags:**
```bash
model-cli harden \
  --model phi-4-mini \
  --model-path ./models \
  --artifact my-model:v1 \
  --generate-sbom \
  --include-mof
```

**Interactive TUI tabs (when run interactively):**
- Progress: Shows hardening step progress
- Config: Tool configuration
- Model: Model information
- SBOM: SBOM status and results
- MOF: MOF classification status and results
- Logs: Operation logs
- Help: Keyboard shortcuts
- Env: Environment info

### check

Performs local compliance check before pushing artifact to registry.

**Interactive panels:**
- Introduction explaining compliance check concepts
- Progress display during checks
- Results showing passed/failed checks
- Completion summary

**Features:**
- Validates required annotations (org.cncf.ai.artifact.type, runtime, accelerator)
- Checks SBOM presence in artifact layers
- Checks MOF classification presence
- Fails/blocks with clear message when required pieces are missing
- Wired into wizard flow ahead of push/sign steps

**Non-interactive flags:**
```bash
model-cli validate local \
  --model-path ./models \
  --artifact-path ./output \
  --strict
```

**Interactive TUI tabs (when run interactively):**
- Progress: Shows check step progress
- Config: Tool configuration
- Model: Model information
- Checks: List of required checks
- Results: Pass/fail status of each check
- Logs: Operation logs
- Help: Keyboard shortcuts
- Env: Environment info

## TUI Best Practices

1. **For users**: Start with `model-cli wizard` for the complete guided experience
2. **For scripts**: Use non-interactive flags for automation
3. **For CI/CD**: Use non-interactive flags with environment variables
4. **For debugging**: Use `--help` to see all available flags for each command

## Accessibility

The TUI uses color coding that works with most terminal color schemes. For accessibility:
- Colors are secondary to text content
- All information is readable without color
- Success indicators use text symbols (checkmarks) in addition to colors

## Customization

The TUI styling can be customized by modifying the style definitions in `cmd/wizard.go`:
- `titleStyle` - Main headers
- `subtitleStyle` - Subheaders and descriptions
- `stepStyle` - Step indicators
- `successStyle` - Success messages
- `infoStyle` - Information messages
- `warningStyle` - Warnings
