package workflow

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// SLSA Provenance v0.2 schema (simplified for Model CLI)
// Based on: https://slsa.dev/spec/v0.2/provenance
// This is a minimal implementation suitable for AI/ML model artifacts

type ProvenanceAttestation struct {
	// Standard in-toto provenance statement fields
	StatementHeader `json:"statement_header"`
	Statement       Statement `json:"statement"`
}

type StatementHeader struct {
	Type          string    `json:"type"`           // "https://in-toto.io/Statement/v0.1"
	PredicateType string    `json:"predicate_type"` // "https://slsa.dev/provenance/v0.2"
	Subject       []Subject `json:"subject"`        // The artifacts being attested
}

type Statement struct {
	// The artifact(s) being attested
	Subject []Subject `json:"subject,omitempty"`

	// The provenance predicate
	Predicate ProvenancePredicate `json:"predicate"`
}

type Subject struct {
	Name   string            `json:"name"`             // e.g., "ghcr.io/my-org/my-model:v1"
	Digest map[string]string `json:"digest,omitempty"` // e.g., {"sha256": "abc123..."}
}

// ProvenancePredicate contains the actual provenance information
type ProvenancePredicate struct {
	// SLSA v0.2 Provenance fields
	BuildType   string       `json:"buildType"`             // e.g., "https://model-cli.dev/build/v1"
	BuildID     string       `json:"buildId"`               // Unique identifier for this build
	Builder     Builder      `json:"builder"`               // Information about the builder
	Recipe      Recipe       `json:"recipe"`                // The recipe that produced the artifact
	Source      Source       `json:"source"`                // Source information
	Materials   []Material   `json:"materials,omitempty"`   // Build materials (dependencies)
	Metadata    Metadata     `json:"metadata"`              // Additional metadata
	Invocations []Invocation `json:"invocations,omitempty"` // Build invocations
}

type Builder struct {
	ID string `json:"id"` // e.g., "model-cli/v1.0.0"
}

type Recipe struct {
	Type       string `json:"type"`       // e.g., "https://model-cli.dev/recipe/v1"
	DefinedIn  string `json:"definedIn"`  // Reference to the recipe file/definition
	EntryPoint string `json:"entryPoint"` // The entry point for the build
}

type Source struct {
	ID     string            `json:"id"`               // e.g., "/path/to/model"
	URI    string            `json:"uri"`              // Optional URI
	Digest map[string]string `json:"digest,omitempty"` // Source digest
}

type Metadata struct {
	BuildStartedOn  time.Time    `json:"buildStartedOn"`
	BuildFinishedOn time.Time    `json:"buildFinishedOn"`
	Completeness    Completeness `json:"completeness"`
	Reproducible    bool         `json:"reproducible"`
}

type Completeness struct {
	Environment bool `json:"environment"` // All build steps in same environment
	Parameters  bool `json:"parameters"`  // All parameters recorded
	Materials   bool `json:"materials"`   // All materials identified
}

type Invocation struct {
	Configuration map[string]interface{} `json:"configuration,omitempty"`
	Parameters    map[string]string      `json:"parameters,omitempty"`
	Environment   map[string]string      `json:"environment,omitempty"`
}

// Material represents a build dependency/material per SLSA spec
type Material struct {
	URI    string            `json:"uri"`              // URI where the material can be fetched
	Digest map[string]string `json:"digest,omitempty"` // Digest of the material (e.g., sha256)
}

// ProvenanceGenerator creates SLSA/in-toto provenance attestations
type ProvenanceGenerator struct {
	// Build information
	builderID string
	buildType string

	// Source information
	sourceID  string
	sourceURI string

	// Recipe information
	recipeType       string
	recipeDefinedIn  string
	recipeEntryPoint string

	// Output information
	artifactName   string
	artifactDigest map[string]string

	// Materials (build dependencies)
	materials []Material

	// Invocation information
	invocationID     string
	invocationConfig map[string]interface{}
}

// NewProvenanceGenerator creates a new provenance generator
func NewProvenanceGenerator() *ProvenanceGenerator {
	return &ProvenanceGenerator{
		builderID:        "model-cli",
		buildType:        "https://model-cli.dev/build/v1",
		recipeType:       "https://model-cli.dev/recipe/v1",
		recipeDefinedIn:  "cmd:model-cli package",
		recipeEntryPoint: "package",
	}
}

// SetBuildInfo configures the build metadata
func (g *ProvenanceGenerator) SetBuildInfo(builderID, buildType string) {
	if builderID != "" {
		g.builderID = builderID
	}
	if buildType != "" {
		g.buildType = buildType
	}
}

// SetSourceInfo configures the source metadata
func (g *ProvenanceGenerator) SetSourceInfo(sourceID, sourceURI string) {
	g.sourceID = sourceID
	g.sourceURI = sourceURI
}

// SetRecipeInfo configures the recipe metadata
func (g *ProvenanceGenerator) SetRecipeInfo(recipeType, definedIn, entryPoint string) {
	if recipeType != "" {
		g.recipeType = recipeType
	}
	if definedIn != "" {
		g.recipeDefinedIn = definedIn
	}
	if entryPoint != "" {
		g.recipeEntryPoint = entryPoint
	}
}

// SetArtifactInfo configures the output artifact information
func (g *ProvenanceGenerator) SetArtifactInfo(name string, digest map[string]string) {
	g.artifactName = name
	g.artifactDigest = digest
}

// AddMaterial adds a build material/dependency
func (g *ProvenanceGenerator) AddMaterial(uri string, digest map[string]string) {
	g.materials = append(g.materials, Material{
		URI:    uri,
		Digest: digest,
	})
}

// SetMaterials sets the complete list of materials
func (g *ProvenanceGenerator) SetMaterials(materials []Material) {
	g.materials = materials
}

// SetInvocationInfo configures the build invocation metadata
func (g *ProvenanceGenerator) SetInvocationInfo(id string, config map[string]interface{}) {
	g.invocationID = id
	g.invocationConfig = config
}

// Generate creates a SLSA provenance attestation for the artifact
func (g *ProvenanceGenerator) Generate() *ProvenanceAttestation {
	buildStartTime := time.Now().UTC().Add(-5 * time.Second) // Approximate build start
	buildFinishTime := time.Now().UTC()
	buildID := g.invocationID
	if buildID == "" {
		buildID = fmt.Sprintf("model-cli-%d", buildFinishTime.UnixNano())
	}

	// Add materials if any
	materials := g.materials
	if len(materials) == 0 && g.sourceURI != "" {
		// If no materials specified, add the source as a material
		materials = []Material{
			{
				URI:    g.sourceURI,
				Digest: nil,
			},
		}
	}

	// Build invocation configuration
	invocationConfig := g.invocationConfig
	if invocationConfig == nil {
		invocationConfig = map[string]interface{}{
			"builder":   g.builderID,
			"buildType": g.buildType,
		}
	}

	// Build parameters
	parameters := map[string]string{
		"artifact": g.artifactName,
	}
	if g.artifactDigest != nil {
		for algo, digest := range g.artifactDigest {
			parameters["digest."+algo] = digest
		}
	}

	// Create invocation
	invocations := []Invocation{
		{
			Configuration: invocationConfig,
			Parameters:    parameters,
			Environment:   map[string]string{},
		},
	}

	// Determine completeness based on what we have
	completeness := Completeness{
		Environment: true,
		Parameters:  true,
		Materials:   len(materials) > 0,
	}

	attestation := &ProvenanceAttestation{
		StatementHeader: StatementHeader{
			Type:          "https://in-toto.io/Statement/v0.1",
			PredicateType: "https://slsa.dev/provenance/v0.2",
			Subject: []Subject{
				{
					Name:   g.artifactName,
					Digest: g.artifactDigest,
				},
			},
		},
		Statement: Statement{
			Subject: []Subject{
				{
					Name:   g.artifactName,
					Digest: g.artifactDigest,
				},
			},
			Predicate: ProvenancePredicate{
				BuildType: g.buildType,
				BuildID:   buildID,
				Builder:   Builder{ID: g.builderID},
				Recipe: Recipe{
					Type:       g.recipeType,
					DefinedIn:  g.recipeDefinedIn,
					EntryPoint: g.recipeEntryPoint,
				},
				Source: Source{
					ID:     g.sourceID,
					URI:    g.sourceURI,
					Digest: nil,
				},
				Materials: materials,
				Metadata: Metadata{
					BuildStartedOn:  buildStartTime,
					BuildFinishedOn: buildFinishTime,
					Completeness:    completeness,
					Reproducible:    false,
				},
				Invocations: invocations,
			},
		},
	}

	return attestation
}

// GenerateFromWorkflow creates a provenance attestation from a PackageWorkflow
func (g *ProvenanceGenerator) GenerateFromWorkflow(w *PackageWorkflow) *ProvenanceAttestation {
	g.SetSourceInfo(w.modelPath, "")
	g.SetArtifactInfo(w.artifactName, nil)

	// Set recipe based on registry
	g.SetRecipeInfo(
		"https://model-cli.dev/recipe/package/v1",
		fmt.Sprintf("registry:%s", w.registry),
		"package",
	)

	return g.Generate()
}

// WriteToFile writes the attestation to a JSON file
func (g *ProvenanceGenerator) WriteToFile(outputPath string) (*ProvenanceAttestation, error) {
	attestation := g.Generate()

	data, err := json.MarshalIndent(attestation, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal provenance attestation: %w", err)
	}

	// Ensure output directory exists
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return nil, fmt.Errorf("failed to write provenance attestation: %w", err)
	}

	return attestation, nil
}

// GetAttestationPath returns the standard path for a provenance attestation
func GetAttestationPath(artifactName string) string {
	// For local artifacts, store alongside the manifest
	// For remote artifacts, this would be stored in the registry as a referrer
	return artifactName + ".provenance.json"
}

// ValidateAttestation validates a provenance attestation structure
func ValidateAttestation(attestation *ProvenanceAttestation) error {
	if attestation == nil {
		return fmt.Errorf("attestation is nil")
	}

	// Validate header
	if attestation.StatementHeader.Type != "https://in-toto.io/Statement/v0.1" {
		return fmt.Errorf("invalid statement header type: %s", attestation.StatementHeader.Type)
	}
	if attestation.StatementHeader.PredicateType != "https://slsa.dev/provenance/v0.2" {
		return fmt.Errorf("invalid predicate type: %s", attestation.StatementHeader.PredicateType)
	}

	// Validate predicate
	predicate := attestation.Statement.Predicate
	if predicate.BuildType == "" {
		return fmt.Errorf("buildType is empty")
	}
	if predicate.Builder.ID == "" {
		return fmt.Errorf("builder.ID is empty")
	}

	// Validate timestamps
	if predicate.Metadata.BuildFinishedOn.IsZero() {
		return fmt.Errorf("buildFinishedOn timestamp is zero")
	}

	return nil
}
