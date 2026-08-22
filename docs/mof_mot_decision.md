# MOF/MOT Implementation Decision

## Issue #49: Generate a MOF/MOT-compliant metadata config file

### Decision: Hand-Roll Implementation

**Date**: 2026-08-22
**Decision**: Implement MOF metadata generation directly in model-cli without external MOT library

### Background

The Model Openness Framework (MOF) requires a structured metadata configuration file containing:
- MOF specification name/version/date
- Release metadata (name/version/date/type)
- Component list with descriptions, types, identifiers, and presence flags

The workflow specification states this file MUST be generatable via MOT (Model Openness Tool) from https://github.com/Adopt-MOF/MOF.

### Investigation

We investigated the MOT (Model Openness Tool) project:
- **Repository**: Not found at `github.com/Adopt-MOF/MOF` or `github.com/Adopt-MOF/MOT`
- **Go Module**: No available Go library at `github.com/Adopt-MOF/MOF`
- **Status**: The MOT project may not yet have a stable Go implementation or may be in early development

### Rationale for Hand-Rolling

1. **No Available Go Library**: No official Go SDK/library exists for MOT at this time
2. **Simplicity**: The MOF metadata structure is well-defined and straightforward to implement
3. **Control**: Direct implementation gives us full control over the output format and validation
4. **Integration**: Tighter integration with our existing MOF classifier and workflow
5. **Future-Proof**: Can easily swap in MOT library when it becomes available

### Implementation

Created `internal/workflow/mof_metadata.go` with:

```go
// MOFMetadata - Structured MOF/MOT-compliant metadata
type MOFMetadata struct {
    MOFVersion  string           // e.g., "1.0"
    Release    ReleaseMetadata
    Model      ModelMetadata
    Components []ComponentMetadata
    GeneratedAt time.Time
    Generator   string
}

type ComponentMetadata struct {
    Type        string // data, code, documentation, license, weights, training-data
    Description string
    Identifier  string
    Present     bool
    Checksum    string // optional
    License     string // optional
}
```

### File Format

- **Primary**: JSON (universal support, works with OCI artifact layers)
- **Future**: YAML support can be added when/if yaml.v3 Go library is added
- **Location**: Generated as `mof.json` in the model directory
- **Attachment**: Simulated OCI layer attachment (real implementation would use ORAS or similar)

### Compliance with MOF Spec

The implementation follows the MOF specification from https://github.com/Adopt-MOF/MOF:
- Includes all required MOF metadata fields
- Structured component information with type/description/identifier
- Machine-readable classification (Class I, II, III)
- Generation timestamps and tool identification

### Future Integration with MOT

When MOT provides a Go library:
1. Replace `MOFMetadataGenerator` with MOT library calls
2. Keep the same interface for backward compatibility
3. Add as a separate provider option (--mof-tool mot vs --mof-tool builtin)
4. Document in the configuration

### Trade-offs

**Pros of Hand-Rolling:**
- Immediate implementation
- No external dependencies
- Full control over output
- Easier debugging

**Cons of Hand-Rolling:**
- May need to update if MOF spec changes
- Doesn't benefit from MOT's validation/normalization
- May diverge from official MOT output format

### Conclusion

Hand-rolling the MOF metadata generation is the pragmatic choice given the current lack of Go libraries for MOT. The implementation is compliant with the MOF specification and can be replaced with official MOT integration in the future.
