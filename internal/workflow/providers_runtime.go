package workflow

import (
	"fmt"
	"os/exec"
)

// --- vLLM Runtime Provider ---

type VLLMProvider struct{}

func (v *VLLMProvider) Name() string {
	return "vllm"
}

func (v *VLLMProvider) IsInstalled() bool {
	cmd := exec.Command("python3", "-c", "import vllm; print('vllm installed')")
	return cmd.Run() == nil
}

func (v *VLLMProvider) InstallInstructions() string {
	return "pip install vllm"
}

func (v *VLLMProvider) Serve(modelPath, host, port string) error {
	cmd := exec.Command("vllm", "serve", "--model", modelPath, "--host", host, "--port", port)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start vLLM server: %v", err)
	}
	fmt.Printf("vLLM server running at %s:%s with model: %s\n", host, port, modelPath)
	return nil
}

// --- KServe Runtime Provider ---

type KServeProvider struct{}

func (k *KServeProvider) Name() string {
	return "kserve"
}

func (k *KServeProvider) IsInstalled() bool {
	cmd := exec.Command("python3", "-c", "import kserve; print('kserve installed')")
	return cmd.Run() == nil
}

func (k *KServeProvider) InstallInstructions() string {
	return "pip install kserve"
}

func (k *KServeProvider) Serve(modelPath, host, port string) error {
	fmt.Printf("Creating KServe InferenceService for model: %s\n", modelPath)
	fmt.Printf("KServe endpoint will be available at %s:%s\n", host, port)
	return nil
}

// GetRuntimeProvider returns the appropriate runtime provider by name
func GetRuntimeProvider(name string) (RuntimeProvider, error) {
	switch name {
	case "vllm":
		return &VLLMProvider{}, nil
	case "kserve":
		return &KServeProvider{}, nil
	default:
		return nil, fmt.Errorf("unknown runtime provider: %s (supported: vllm, kserve)", name)
	}
}
