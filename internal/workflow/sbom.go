package workflow

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// SBOMFormat represents the format of the generated SBOM
type SBOMFormat string

const (
	// SPDXJSON is the SPDX format in JSON
	SPDXJSON SBOMFormat = "spdx-json"
	// SPDXTagValue is the SPDX format in tag-value
	SPDXTagValue SBOMFormat = "spdx-tag-value"
	// CycloneDXJSON is the CycloneDX format in JSON
	CycloneDXJSON SBOMFormat = "cyclonedx-json"
	// CycloneDXXML is the CycloneDX format in XML
	CycloneDXXML SBOMFormat = "cyclonedx-xml"
)

// AllSBOMFormats lists every format an SBOM can be generated in.
var AllSBOMFormats = []SBOMFormat{SPDXJSON, SPDXTagValue, CycloneDXJSON, CycloneDXXML}

// SBOMFileName returns the file name `harden` writes the SBOM to inside the
// model directory, e.g. "sbom.spdx-json". `validate local` uses the same
// rule to find it again.
func SBOMFileName(format SBOMFormat) string {
	return "sbom." + string(format)
}

// SBOMGenerator defines the interface for SBOM generation
type SBOMGenerator interface {
	Name() string
	IsInstalled() bool
	InstallInstructions() string
	Generate(modelPath, outputPath string, format SBOMFormat) error
	DefaultFormat() SBOMFormat
}

// --- Syft SBOM Generator ---

type SyftGenerator struct{}

func (s *SyftGenerator) Name() string {
	return "syft"
}

func (s *SyftGenerator) IsInstalled() bool {
	return exec.Command("syft", "version").Run() == nil
}

func (s *SyftGenerator) InstallInstructions() string {
	return "brew install anchore/syft/syft"
}

func (s *SyftGenerator) DefaultFormat() SBOMFormat {
	return SPDXJSON
}

func (s *SyftGenerator) Generate(modelPath, outputPath string, format SBOMFormat) error {
	if !s.IsInstalled() {
		return fmt.Errorf("syft not installed. Install with: %s", s.InstallInstructions())
	}

	// Ensure output directory exists
	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Map our format to syft's output format
	syftFormat := mapFormatToSyft(format)

	// Build syft command
	args := []string{"dir:" + modelPath, "--output", string(syftFormat) + "=" + outputPath}
	cmd := exec.Command("syft", args...)

	// Set up output for logging
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Printf("Generating SBOM for %s using Syft (format: %s)...\n", modelPath, format)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("syft generation failed: %w", err)
	}

	// Verify the output file was created
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		return fmt.Errorf("SBOM file not created at %s", outputPath)
	}

	fmt.Printf("SBOM generated: %s (format: %s)\n", outputPath, format)
	return nil
}

// mapFormatToSyft maps our SBOMFormat to syft's output format
func mapFormatToSyft(format SBOMFormat) string {
	switch format {
	case SPDXJSON:
		return "spdx-json"
	case SPDXTagValue:
		return "spdx"
	case CycloneDXJSON:
		return "cyclonedx-json"
	case CycloneDXXML:
		return "cyclonedx-xml"
	default:
		return "spdx-json"
	}
}

// --- Trivy SBOM Generator ---

type TrivyGenerator struct{}

func (t *TrivyGenerator) Name() string {
	return "trivy"
}

func (t *TrivyGenerator) IsInstalled() bool {
	return exec.Command("trivy", "--version").Run() == nil
}

func (t *TrivyGenerator) InstallInstructions() string {
	return "brew install aquasecurity/trivy/trivy"
}

func (t *TrivyGenerator) DefaultFormat() SBOMFormat {
	return CycloneDXJSON
}

func (t *TrivyGenerator) Generate(modelPath, outputPath string, format SBOMFormat) error {
	if !t.IsInstalled() {
		return fmt.Errorf("trivy not installed. Install with: %s", t.InstallInstructions())
	}

	// Ensure output directory exists
	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Map our format to trivy's output format
	trivyFormat := mapFormatToTrivy(format)

	// Build trivy command
	args := []string{"fs", "--security-checks", "vulnerability,secret", "--format", trivyFormat, modelPath}
	cmd := exec.Command("trivy", args...)

	// Capture output to file
	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()

	cmd.Stdout = outFile
	cmd.Stderr = os.Stderr

	fmt.Printf("Generating SBOM for %s using Trivy (format: %s)...\n", modelPath, format)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("trivy generation failed: %w", err)
	}

	// Verify the output file was created
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		return fmt.Errorf("SBOM file not created at %s", outputPath)
	}

	fmt.Printf("SBOM generated: %s (format: %s)\n", outputPath, format)
	return nil
}

// mapFormatToTrivy maps our SBOMFormat to trivy's output format
func mapFormatToTrivy(format SBOMFormat) string {
	switch format {
	case SPDXJSON:
		return "spdx"
	case CycloneDXJSON:
		return "cyclonedx"
	case CycloneDXXML:
		// Trivy doesn't support XML, fall back to JSON
		return "cyclonedx"
	default:
		return "spdx"
	}
}

// --- cdxgen SBOM Generator ---

type CdxgenGenerator struct{}

func (c *CdxgenGenerator) Name() string {
	return "cdxgen"
}

func (c *CdxgenGenerator) IsInstalled() bool {
	return exec.Command("cdxgen", "--version").Run() == nil
}

func (c *CdxgenGenerator) InstallInstructions() string {
	return "npm install -g @cyclonedx/cdxgen"
}

func (c *CdxgenGenerator) DefaultFormat() SBOMFormat {
	return CycloneDXJSON
}

func (c *CdxgenGenerator) Generate(modelPath, outputPath string, format SBOMFormat) error {
	if !c.IsInstalled() {
		return fmt.Errorf("cdxgen not installed. Install with: %s", c.InstallInstructions())
	}

	// Ensure output directory exists
	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Map our format to cdxgen's output type
	cdxgenType := mapFormatToCdxgen(format)

	// Build cdxgen command
	args := []string{"-t", cdxgenType, "-o", outputPath, modelPath}
	cmd := exec.Command("cdxgen", args...)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Printf("Generating SBOM for %s using cdxgen (format: %s)...\n", modelPath, format)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("cdxgen generation failed: %w", err)
	}

	// Verify the output file was created
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		return fmt.Errorf("SBOM file not created at %s", outputPath)
	}

	fmt.Printf("SBOM generated: %s (format: %s)\n", outputPath, format)
	return nil
}

// mapFormatToCdxgen maps our SBOMFormat to cdxgen's output type
func mapFormatToCdxgen(format SBOMFormat) string {
	// cdxgen primarily generates CycloneDX
	switch format {
	case CycloneDXJSON, CycloneDXXML, SPDXJSON, SPDXTagValue:
		return "dir" // directory scan
	default:
		return "dir"
	}
}

// GetSBOMGenerator returns the appropriate SBOM generator
func GetSBOMGenerator(name string) (SBOMGenerator, error) {
	switch strings.ToLower(name) {
	case "syft":
		return &SyftGenerator{}, nil
	case "trivy":
		return &TrivyGenerator{}, nil
	case "cdxgen":
		return &CdxgenGenerator{}, nil
	default:
		return nil, fmt.Errorf("unknown SBOM generator: %s (supported: syft, trivy, cdxgen)", name)
	}
}

// GetAllSBOMGenerators returns a list of all available SBOM generators
func GetAllSBOMGenerators() []SBOMGenerator {
	return []SBOMGenerator{
		&SyftGenerator{},
		&TrivyGenerator{},
		&CdxgenGenerator{},
	}
}

// GetDefaultSBOMGenerator returns the default SBOM generator (syft)
func GetDefaultSBOMGenerator() SBOMGenerator {
	return &SyftGenerator{}
}
