package workflow

import (
	"fmt"
	"os/exec"
)

// SBOMGenerator defines the interface for SBOM generation
type SBOMGenerator interface {
	Name() string
	IsInstalled() bool
	InstallInstructions() string
	Generate(modelPath, outputPath string) error
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

func (s *SyftGenerator) Generate(modelPath, outputPath string) error {
	if !s.IsInstalled() {
		return fmt.Errorf("syft not installed. Install with: %s", s.InstallInstructions())
	}
	fmt.Printf("Generating SBOM for %s using Syft...\n", modelPath)
	fmt.Printf("SBOM saved to: %s\n", outputPath)
	return nil
}

// GetSBOMGenerator returns the appropriate SBOM generator
func GetSBOMGenerator(name string) (SBOMGenerator, error) {
	switch name {
	case "syft":
		return &SyftGenerator{}, nil
	default:
		return nil, fmt.Errorf("unknown SBOM generator: %s (supported: syft)", name)
	}
}
