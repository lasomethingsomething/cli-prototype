package workflow

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// makeSBOMDeterministic post-processes an SBOM file to remove non-deterministic elements
// This ensures that running the hardening workflow multiple times doesn't dirty the git tree
func makeSBOMDeterministic(sbomPath string) error {
	// Read the SBOM file
	data, err := os.ReadFile(sbomPath)
	if err != nil {
		return fmt.Errorf("failed to read SBOM file: %w", err)
	}

	// Try to parse as JSON and remove non-deterministic fields
	var sbomMap map[string]interface{}
	if err := json.Unmarshal(data, &sbomMap); err != nil {
		// Not JSON, might be another format - try to handle as text
		return removeNonDeterministicText(string(data), sbomPath)
	}

	// Remove non-deterministic fields
	removeNonDeterministicFields(sbomMap)

	// Marshal back to JSON
	cleanData, err := json.MarshalIndent(sbomMap, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cleaned SBOM: %w", err)
	}

	// Write back to file
	if err := os.WriteFile(sbomPath, cleanData, 0644); err != nil {
		return fmt.Errorf("failed to write cleaned SBOM: %w", err)
	}

	return nil
}

// removeNonDeterministicFields recursively removes non-deterministic fields from a map
func removeNonDeterministicFields(data map[string]interface{}) {
	// Fields to remove - these typically contain timestamps, UUIDs, or random data
	nonDeterministicFields := []string{
		"serialNumber",
		"uuid",
		"uid",
		"timestamp",
		"created",
		"createdAt",
		"generated",
		"generatedAt",
		"buildTimestamp",
		"date",
		"documentNamespace",
	}

	for key := range data {
		// Check if this key should be removed
		if containsIgnoreCase(nonDeterministicFields, key) {
			delete(data, key)
			continue
		}

		// Recurse into nested structures
		switch v := data[key].(type) {
		case map[string]interface{}:
			removeNonDeterministicFields(v)
		case []interface{}:
			for i, item := range v {
				if m, ok := item.(map[string]interface{}); ok {
					removeNonDeterministicFields(m)
					v[i] = m
				}
			}
		}
	}
}

// containsIgnoreCase checks if a slice contains a string (case-insensitive)
func containsIgnoreCase(slice []string, value string) bool {
	lowerValue := strings.ToLower(value)
	for _, s := range slice {
		if strings.ToLower(s) == lowerValue {
			return true
		}
	}
	return false
}

// removeNonDeterministicText handles non-JSON SBOM formats by removing lines with timestamps/UUIDs
func removeNonDeterministicText(content string, sbomPath string) error {
	lines := strings.Split(content, "\n")
	var cleanLines []string

	for _, line := range lines {
		// Skip lines containing non-deterministic elements
		lowerLine := strings.ToLower(line)
		if strings.Contains(lowerLine, "serialnumber:") ||
			strings.Contains(lowerLine, "uuid:") ||
			strings.Contains(lowerLine, "uid:") ||
			strings.Contains(lowerLine, "timestamp:") ||
			strings.Contains(lowerLine, "created:") ||
			strings.Contains(lowerLine, "generated:") ||
			strings.Contains(lowerLine, "date:") {
			continue
		}
		cleanLines = append(cleanLines, line)
	}

	cleanContent := strings.Join(cleanLines, "\n")
	if err := os.WriteFile(sbomPath, []byte(cleanContent), 0644); err != nil {
		return fmt.Errorf("failed to write cleaned SBOM: %w", err)
	}

	return nil
}

// GenerateDeterministicSBOM generates an SBOM and makes it deterministic
func GenerateDeterministicSBOM(generator SBOMGenerator, modelPath, outputPath string, format SBOMFormat) error {
	// First, generate the SBOM normally
	if err := generator.Generate(modelPath, outputPath, format); err != nil {
		return err
	}

	// Then, make it deterministic
	if err := makeSBOMDeterministic(outputPath); err != nil {
		return fmt.Errorf("failed to make SBOM deterministic: %w", err)
	}

	return nil
}

// HardenWorkflow needs to be updated to use deterministic SBOM generation
// This will be done by modifying the harden.go file to call makeSBOMDeterministic after generation

// EnsureDeterministicSBOM ensures an SBOM file is deterministic
// This is a helper function that can be called after SBOM generation
func EnsureDeterministicSBOM(sbomPath string) error {
	if sbomPath == "" {
		return nil
	}

	// Check if file exists
	if _, err := os.Stat(sbomPath); os.IsNotExist(err) {
		return nil
	}

	return makeSBOMDeterministic(sbomPath)
}
