# Enterprise OCI Registry Integration

Model CLI implements a complete Enterprise OCI Registry Integration workflow (Phase 2) that enables organizations to store, validate, discover, and cross-reference AI assets efficiently.

## Overview

The Enterprise OCI Registry Integration epic consists of 5 stories that together provide a comprehensive solution for managing AI artifacts in OCI registries:

| Story | Title | Purpose |
|-------|-------|---------|
| #58 | Push Unified OCI Manifest to Registry | Push artifacts with standardized metadata |
| #59 | Enforce Standardized Metadata Contract at Manifest Level | Validate and reject non-compliant pushes |
| #60 | Map Complex Relationships in Manifest | Embed dependency graphs in manifests |
| #61 | Cross-Reference Assets in Registry | Query and discover related assets |
| #62 | Validate Pushes Against Metadata Contract | Dry-run validation with clear error messages |

## Architecture

The workflow follows a **decoupled, handoff-based architecture**:

1. **CLI Orchestration**: The CLI (`model-cli`) orchestrates the workflow
2. **Registry Delegation**: Actual registry operations are delegated to tools like ORAS and ModelPack
3. **Client-Side Filtering**: Complex queries are filtered client-side when registry support is limited
4. **Manifest-Level Metadata**: All metadata is stored in OCI manifest annotations for efficient access

This ensures the CLI remains the "main plot" that users return to, while leveraging external tools for their specialized functions.

## Story #58: Push Unified OCI Manifest to Registry

Push OCI artifacts with standardized metadata to registries.

### Features
- Unified OCI manifest format with CNCF AI annotations
- Support for multiple registry tools (ORAS, ModelPack)
- Immutable provenance/attestation metadata
- SLSA compliance

### Usage

```bash
# Push with ORAS
model-cli push --artifact my-model:v1 --registry oras --destination ghcr.io/my-org

# Push with ModelPack
model-cli push --artifact my-model:v1 --registry modelpack --destination ghcr.io/my-org

# Push with custom metadata
model-cli push --artifact my-model:v1 --model-type llm --framework pytorch
```

### Metadata Annotations

The following CNCF AI Interoperability Profile annotations are attached:

- `org.cncf.ai.interop.profile.version` - Profile version
- `org.cncf.ai.artifact.type` - Artifact type (model, skill, pipeline)
- `org.cncf.ai.model.mof.class` - MOF classification (I, II, III)
- `org.cncf.ai.model.mof.version` - MOF version
- `org.cncf.ai.model.mof.components` - MOF components
- `org.cncf.ai.security.signing.framework` - Signing framework
- `org.cncf.ai.runtime` - Runtime requirements
- `org.cncf.ai.accelerator` - Hardware accelerator
- `ai.assets` - Standardized metadata contract (JSON)
- `ai.relationships` - Relationship graph (JSON)

## Story #59: Enforce Standardized Metadata Contract at Manifest Level

Validate and reject pushes that violate the Standardized Metadata Contract.

### Features
- Admission webhook for registry validation
- Validates required fields at manifest level
- Rejects non-compliant pushes with clear error messages
- Supports both HTTP webhook and CLI-based enforcement

### Usage

```bash
# Enforce contract validation on a manifest
model-cli enforce --manifest my-manifest.json

# Run as admission webhook server
model-cli enforce --webhook --port 8443

# Simulate a push to test validation
model-cli enforce --manifest my-manifest.json --simulate-push
```

### Validation Rules

**Models** require:
- `type` - Model type (llm, embedding, classification, etc.)
- `framework` - Framework (pytorch, tensorflow, onnx, etc.)

**Skills** require:
- `type` - Skill type (rag, classification, summarization, etc.)
- `dependencies` - At least one model dependency

**Pipelines** require:
- `type` - Pipeline type (inference, training, fine-tuning, etc.)
- `stages` - At least one pipeline stage

## Story #60: Map Complex Relationships in Manifest

Embed relationship maps between AI assets in the OCI manifest.

### Features
- Relationship graph embedded in manifest annotations
- Supports model → skill → pipeline dependencies
- Enables registry to parse dependencies without downloading binaries
- JSON-LD compatible format

### Usage

```bash
# Create relationship graph
model-cli map --model my-model:v1 --skill my-skill:v1 --pipeline my-pipeline:v1

# Map with explicit dependencies
model-cli map --model my-model:v1 --requires my-skill:v1

# Map skill to pipeline
model-cli map --skill my-skill:v1 --used-by my-pipeline:v1
```

### Relationship Graph Format

```json
{
  "ai.relationships": {
    "model": {
      "my-model:sha256:abc123": {
        "used_by": ["skill:sha256:def456", "pipeline:sha256:ghi789"]
      }
    },
    "skill": {
      "my-skill:sha256:def456": {
        "requires_models": ["model:sha256:abc123"],
        "used_by": ["pipeline:sha256:ghi789"]
      }
    },
    "pipeline": {
      "my-pipeline:sha256:ghi789": {
        "uses_models": ["model:sha256:abc123"],
        "uses_skills": ["skill:sha256:def456"]
      }
    }
  }
}
```

## Story #61: Cross-Reference Assets in Registry

Query the registry to discover and cross-reference AI assets.

### Features
- Registry filtering by metadata annotations
- CLI search commands with multiple filter options
- Client-side filtering for complex relationship queries
- Structured results with full metadata
- Support for JSON, YAML, and table output formats

### Usage

```bash
# Search all artifacts in a registry
model-cli search --destination ghcr.io/my-org

# Filter by artifact type
model-cli search --destination ghcr.io/my-org --type model
model-cli search --destination ghcr.io/my-org --type skill
model-cli search --destination ghcr.io/my-org --type pipeline

# Filter by specific asset
model-cli search --model my-llm --type pipeline
model-cli search --skill my-skill:v1

# Filter by relationships
model-cli search --uses-model model:sha256:abc123
model-cli search --uses-skill skill:sha256:def456
model-cli search --requires-model model:sha256:abc123
model-cli search --used-by pipeline:sha256:ghi789

# Filter by metadata
model-cli search --metadata ai.model.type=llm
model-cli search --metadata ai.model.framework=pytorch
model-cli search --metadata ai.skill.type=rag

# Output formats
model-cli search --destination ghcr.io/my-org --output json
model-cli search --destination ghcr.io/my-org --output yaml
```

### OCI Filter Support

The search command generates OCI-compliant filter strings:

- `org.cncf.ai.artifact.type=model` - Filter by artifact type
- `ai.model.type=llm` - Filter by model type
- `ai.model.framework=pytorch` - Filter by framework
- `ai.skill.type=rag` - Filter by skill type

## Story #62: Validate Pushes Against Metadata Contract

Dry-run validation of manifests against the Standardized Metadata Contract.

### Features
- JSON Schema-based validation
- Clear error messages for missing/invalid fields
- Strict mode for warning-as-error behavior
- Supports both manifest files and registry artifacts
- Lists missing required fields

### Usage

```bash
# Validate a manifest file (dry-run)
model-cli validate --manifest my-manifest.json

# Strict validation with JSON Schema
model-cli validate --manifest my-manifest.json --json-schema --strict

# Validate with artifact type override
model-cli validate --manifest my-manifest.json --artifact-type skill

# Validate a registry artifact
model-cli validate --artifact ghcr.io/my-org/my-model:v1

# Check relationships
model-cli validate --manifest my-manifest.json --check-relationships
```

### JSON Schema

The metadata contract is validated against the following JSON Schema:

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "$id": "https://model-cli.dev/schemas/metadata-contract/v1.json",
  "title": "Standardized Metadata Contract Schema",
  "type": "object",
  "required": ["ai.assets"],
  "properties": {
    "ai.assets": {
      "type": "object",
      "oneOf": [
        {"required": ["model"]},
        {"required": ["skill"]},
        {"required": ["pipeline"]}
      ]
    }
  }
}
```

### Required Fields by Artifact Type

**Model:**
- `ai.assets.model.type` - Required
- `ai.assets.model.framework` - Required

**Skill:**
- `ai.assets.skill.type` - Required
- `ai.assets.skill.dependencies` - Required (must include at least one model)

**Pipeline:**
- `ai.assets.pipeline.type` - Required
- `ai.assets.pipeline.stages` - Required (at least one stage)

### Error Messages

When validation fails, the CLI provides clear, actionable error messages:

```
 Metadata contract validation FAILED

Invalid: Metadata contract validation failed:
  - Error: model.type is required
  - Error: model.framework is required
  - Missing required fields: ai.assets.model.type, ai.assets.model.framework
```

## Complete Workflow Example

Here's how all the stories work together in a complete workflow:

```bash
# 1. Package your model with relationships
model-cli package --model my-model --model-path ./models \
  --model-type llm --framework pytorch \
  --relationships skill=my-skill:v1

# 2. Map the relationship graph
model-cli map --model my-model:v1 --skill my-skill:v1 --pipeline my-pipeline:v1

# 3. Dry-run validation
model-cli validate --manifest my-manifest.json --json-schema

# 4. Enforce at registry level (optional: run as webhook)
model-cli enforce --webhook --port 8443 &

# 5. Push to registry
model-cli push --artifact my-model:v1 --registry oras --destination ghcr.io/my-org

# 6. Search for related assets
model-cli search --destination ghcr.io/my-org --uses-model my-model:sha256:abc123
```

## Metadata Contract Reference

### Annotation Keys

| Annotation | Purpose | Example Value |
|------------|---------|---------------|
| `ai.assets` | Standardized metadata contract | `{"model": {"type": "llm", "framework": "pytorch"}}` |
| `ai.relationships` | Relationship graph | `{"model": {"abc123": {"used_by": ["def456"]}}}` |
| `org.cncf.ai.artifact.type` | Artifact type | `model`, `skill`, `pipeline` |

### Contract Structure

```json
{
  "ai.assets": {
    "model": {
      "type": "llm",
      "framework": "pytorch",
      "input": "text",
      "output": "text",
      "capabilities": ["chat", "completion"],
      "description": "My LLM model",
      "version": "1.0.0",
      "author": "my-org",
      "license": "Apache-2.0",
      "runtime": "vllm",
      "accelerator": "nvidia-gpu",
      "relationships": {
        "skills": ["my-skill:v1"],
        "pipelines": ["my-pipeline:v1"]
      }
    }
  }
}
```

## Registry Tool Support

### ORAS
- Full support for OCI artifact operations
- Manifest annotation support
- Discover command for listing artifacts
- Referrer support for provenance/attestations

### ModelPack
- AI-optimized OCI packaging
- Metadata-aware operations
- Stub implementation for search (delegates to registry API)

### Registry Requirements

For full functionality, use registries that support:
- OCI Image Spec annotations
- OCI Distribution Spec filtering
- Manifest discovery/listing

Recommended registries:
- GitHub Container Registry (ghcr.io)
- Docker Hub (docker.io)
- Google Container Registry (gcr.io)
- Azure Container Registry (azurecr.io)
- AWS Elastic Container Registry (ecr.aws)

## Troubleshooting

### Registry Search Not Supported
If your registry doesn't support server-side filtering:
```
Note: Registry search with filters failed: registry search not fully supported by ORAS CLI.
Falling back to client-side filtering...
```

This is expected. The CLI will fetch all manifests and filter client-side.

### Missing Metadata Contract
```
Error: no metadata contract found in annotations. Expected annotation: ai.assets
```

Add the metadata contract to your manifest:
```bash
model-cli package --model my-model --model-type llm --framework pytorch
```

### Validation Failed
```
Error: manifest validation failed: model.type is required; model.framework is required
```

Provide the required fields:
```bash
model-cli validate --manifest my-manifest.json --artifact-type model --model-type llm --framework pytorch
```

## Best Practices

1. **Always validate before pushing**
   ```bash
   model-cli validate --manifest my-manifest.json --strict
   ```

2. **Use JSON Schema for strict compliance**
   ```bash
   model-cli validate --manifest my-manifest.json --json-schema
   ```

3. **Map relationships explicitly**
   ```bash
   model-cli map --model my-model --skill my-skill --pipeline my-pipeline
   ```

4. **Use registry search for dependency discovery**
   ```bash
   model-cli search --uses-model my-model:latest
   ```

5. **Enforce at registry level for production**
   ```bash
   model-cli enforce --webhook --port 8443
   ```

## See Also

- [OCI Specification](https://specs.opencontainers.org/image-spec/)
- [OCI Distribution Spec](https://github.com/opencontainers/distribution-spec)
- [CNCF AI Interoperability Profile](https://github.com/opencontainers/image-spec/blob/main/annotations.md)
- [JSON Schema](https://json-schema.org/)
- [ORAS](https://oras.land/)
- [ModelPack](https://github.com/modelpack/model-spec)
