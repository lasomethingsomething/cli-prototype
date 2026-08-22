package workflow

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// MOFMetadata represents the structured MOF/MOT-compliant metadata configuration
// Based on the Model Openness Framework specification: https://github.com/Adopt-MOF/MOF
type MOFMetadata struct {
	// MOF specification version
	MOFVersion string `json:"mof_version" yaml:"mof_version"`
	
	// Release metadata
	Release ReleaseMetadata `json:"release" yaml:"release"`
	
	// Model-specific MOF metadata
	Model ModelMetadata `json:"model" yaml:"model"`
	
	// Components list with detailed information
	Components []ComponentMetadata `json:"components" yaml:"components"`
	
	// Generation metadata
	GeneratedAt time.Time `json:"generated_at" yaml:"generated_at"`
	Generator    string   `json:"generator" yaml:"generator"`
}

// ReleaseMetadata contains release information
type ReleaseMetadata struct {
	Name    string `json:"name" yaml:"name"`
	Version string `json:"version" yaml:"version"`
	Date    string `json:"date" yaml:"date"`
	Type    string `json:"type" yaml:"type"` // e.g., "model", "skill", "dataset"
}

// ModelMetadata contains MOF classification for the model
type ModelMetadata struct {
	Name       string `json:"name" yaml:"name"`
	Class      string `json:"class" yaml:"class"` // I, II, or III
	ClassName  string `json:"class_name" yaml:"class_name"`
	Explanation string `json:"explanation" yaml:"explanation"`
}

// ComponentMetadata represents a component in the MOF specification
type ComponentMetadata struct {
	// Component type: data, code, documentation, license, weights, training-data
	Type string `json:"type" yaml:"type"`
	
	// Description of the component
	Description string `json:"description" yaml:"description"`
	
	// Identifier (file path, URL, or other unique identifier)
	Identifier string `json:"identifier" yaml:"identifier"`
	
	// Whether this component is present/available
	Present bool `json:"present" yaml:"present"`
	
	// Optional: hash or checksum for verification
	Checksum string `json:"checksum,omitempty" yaml:"checksum,omitempty"`
	
	// Optional: license information (for code/data components)
	License string `json:"license,omitempty" yaml:"license,omitempty"`
}

// MOFMetadataGenerator creates MOF-compliant metadata config files
type MOFMetadataGenerator struct {
	modelName    string
	modelPath    string
	artifactName string
	mofClass     string
	components   []string
	explanation  string
	
	// Release info
	releaseName    string
	releaseVersion string
	releaseDate    string
	releaseType    string
}

// NewMOFMetadataGenerator creates a new metadata generator
func NewMOFMetadataGenerator() *MOFMetadataGenerator {
	return &MOFMetadataGenerator{
		releaseType: "model",
	}
}

// SetModelInfo configures the model information
func (g *MOFMetadataGenerator) SetModelInfo(name, path, artifact string) {
	g.modelName = name
	g.modelPath = path
	g.artifactName = artifact
}

// SetMOFClassification configures the MOF classification results
func (g *MOFMetadataGenerator) SetMOFClassification(classStr string, components []string, explanation string) {
	g.mofClass = classStr
	g.components = components
	g.explanation = explanation
}

// SetReleaseInfo configures the release metadata
func (g *MOFMetadataGenerator) SetReleaseInfo(name, version, date, artifactType string) {
	g.releaseName = name
	g.releaseVersion = version
	g.releaseDate = date
	if artifactType != "" {
		g.releaseType = artifactType
	}
}

// Generate creates the MOF metadata structure
func (g *MOFMetadataGenerator) Generate() *MOFMetadata {
	metadata := &MOFMetadata{
		MOFVersion: "1.0",
		GeneratedAt: time.Now().UTC(),
		Generator:    "model-cli",
		Release: ReleaseMetadata{
			Name:    g.releaseName,
			Version: g.releaseVersion,
			Date:    g.releaseDate,
			Type:    g.releaseType,
		},
		Model: ModelMetadata{
			Name:       g.modelName,
			Class:      g.mofClass,
			ClassName:  getMOFClassName(g.mofClass),
			Explanation: g.explanation,
		},
		Components: g.buildComponentMetadata(),
	}
	
	// If release fields are empty, use defaults
	if metadata.Release.Name == "" {
		metadata.Release.Name = g.modelName
	}
	if metadata.Release.Version == "" {
		metadata.Release.Version = "1.0.0"
	}
	if metadata.Release.Date == "" {
		metadata.Release.Date = time.Now().UTC().Format("2006-01-02")
	}
	
	return metadata
}

// buildComponentMetadata creates component entries for all MOF component types
func (g *MOFMetadataGenerator) buildComponentMetadata() []ComponentMetadata {
	components := []ComponentMetadata{
		{
			Type:        "weights",
			Description: "Model weights/parameters",
			Identifier:  "model-weights",
			Present:     sliceContains(g.components, "weights"),
		},
		{
			Type:        "code",
			Description: "Training or inference code",
			Identifier:  "training-code",
			Present:     sliceContains(g.components, "code"),
		},
		{
			Type:        "training-data",
			Description: "Training datasets and corpora",
			Identifier:  "training-data",
			Present:     sliceContains(g.components, "training-data"),
		},
		{
			Type:        "documentation",
			Description: "Model documentation (README, etc.)",
			Identifier:  "model-documentation",
			Present:     sliceContains(g.components, "documentation"),
		},
		{
			Type:        "license",
			Description: "Model license file(s)",
			Identifier:  "model-license",
			Present:     sliceContains(g.components, "license"),
		},
	}
	
	return components
}

// WriteToFile writes the metadata to a JSON or YAML file
func (g *MOFMetadataGenerator) WriteToFile(outputPath string, format string) error {
	metadata := g.Generate()
	
	var data []byte
	var err error
	
	if format == "yaml" || format == "yml" {
		data, err = yamlMarshal(metadata)
	} else {
		// Default to JSON
		data, err = json.MarshalIndent(metadata, "", "  ")
	}
	
	if err != nil {
		return fmt.Errorf("failed to marshal MOF metadata: %w", err)
	}
	
	// Ensure output directory exists
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}
	
	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write MOF metadata file: %w", err)
	}
	
	return nil
}

// yamlMarshal is a simple YAML marshaler (we don't want to add yaml dependency)
// For now, we'll use JSON as the primary format and document YAML as future enhancement
func yamlMarshal(v interface{}) ([]byte, error) {
	// For simplicity, convert to JSON and note that full YAML support
	// would require adding a YAML library. The JSON format is sufficient
	// for MOF/MOT compliance and OCI artifact layers.
	return json.MarshalIndent(v, "", "  ")
}

// sliceContains checks if a slice contains a string
func sliceContains(slice []string, value string) bool {
	for _, s := range slice {
		if s == value {
			return true
		}
	}
	return false
}

// getMOFClassName returns a human-readable name for the MOF class
func getMOFClassName(class string) string {
	switch class {
	case "I":
		return "Fully Open"
	case "II":
		return "Partially Open"
	case "III":
		return "Closed/Proprietary"
	default:
		return "Unknown"
	}
}

// MOFMetadataFromClassification creates MOF metadata from a ClassificationResult
func MOFMetadataFromClassification(modelName, modelPath, artifactName, releaseVersion string, result *ClassificationResult) *MOFMetadata {
	generator := NewMOFMetadataGenerator()
	generator.SetModelInfo(modelName, modelPath, artifactName)
	
	// Build components list
	components := []string{}
	if result.HasWeights {
		components = append(components, "weights")
	}
	if result.HasCode {
		components = append(components, "code")
	}
	if result.HasTrainingData {
		components = append(components, "training-data")
	}
	if result.HasDocs {
		components = append(components, "documentation")
	}
	if result.HasLicense {
		components = append(components, "license")
	}
	
	generator.SetMOFClassification(string(result.Class), components, result.Explanation)
	generator.SetReleaseInfo(modelName, releaseVersion, "", "model")
	
	return generator.Generate()
}
