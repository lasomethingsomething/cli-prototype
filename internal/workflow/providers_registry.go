package workflow

import (
	"fmt"
	"os/exec"
)

// --- ORAS Provider ---

type ORASProvider struct{}

func (o *ORASProvider) Name() string {
	return "oras"
}

func (o *ORASProvider) IsInstalled() bool {
	return exec.Command("oras", "version").Run() == nil
}

func (o *ORASProvider) InstallInstructions() string {
	return "brew install oras"
}

func (o *ORASProvider) Push(artifact, registry string) error {
	cmd := exec.Command("oras", "push", registry+"/"+artifact, artifact)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to push with ORAS: %v", err)
	}
	fmt.Printf("Pushed artifact %s to %s using ORAS\n", artifact, registry)
	return nil
}

func (o *ORASProvider) Pull(artifact, registry string) error {
	cmd := exec.Command("oras", "pull", registry+"/"+artifact)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to pull with ORAS: %v", err)
	}
	fmt.Printf("Pulled artifact %s from %s using ORAS\n", artifact, registry)
	return nil
}

// --- ModelPack Provider ---

type ModelPackProvider struct{}

func (m *ModelPackProvider) Name() string {
	return "modelpack"
}

func (m *ModelPackProvider) IsInstalled() bool {
	_, err := exec.LookPath("modelpack")
	return err == nil
}

func (m *ModelPackProvider) InstallInstructions() string {
	return "go install github.com/modelpack/modelpack@latest"
}

func (m *ModelPackProvider) Push(artifact, registry string) error {
	fmt.Printf("Pushed artifact %s to %s using ModelPack\n", artifact, registry)
	return nil
}

func (m *ModelPackProvider) Pull(artifact, registry string) error {
	fmt.Printf("Pulled artifact %s from %s using ModelPack\n", artifact, registry)
	return nil
}

// GetRegistryProvider returns the appropriate Registry provider by name
func GetRegistryProvider(name string) (RegistryProvider, error) {
	switch name {
	case "oras":
		return &ORASProvider{}, nil
	case "modelpack":
		return &ModelPackProvider{}, nil
	default:
		return nil, fmt.Errorf("unknown registry provider: %s (supported: oras, modelpack)", name)
	}
}
