package workflow

import (
	"fmt"
)

// ServeWorkflow orchestrates model serving with pluggable runtimes
type ServeWorkflow struct {
	runtime        string
	runtimeProvider RuntimeProvider
	modelPath      string
	host           string
	port           string
}

// NewServeWorkflow creates a new serving workflow
func NewServeWorkflow(runtime string) (*ServeWorkflow, error) {
	runtimeProvider, err := GetRuntimeProvider(runtime)
	if err != nil {
		return nil, err
	}

	return &ServeWorkflow{
		runtime:        runtime,
		runtimeProvider: runtimeProvider,
	}, nil
}

// SetServeInfo sets the serving details
func (w *ServeWorkflow) SetServeInfo(modelPath, host, port string) {
	w.modelPath = modelPath
	w.host = host
	w.port = port
}

// Run executes the serving workflow
func (w *ServeWorkflow) Run() error {
	fmt.Printf("Starting %s server for model: %s\n", w.runtime, w.modelPath)

	// Check if runtime is available
	if !w.runtimeProvider.IsInstalled() {
		return fmt.Errorf("%s not installed. Install with: %s", w.runtime, w.runtimeProvider.InstallInstructions())
	}

	fmt.Printf("Using %s as runtime...\n", w.runtime)
	
	if err := w.runtimeProvider.Serve(w.modelPath, w.host, w.port); err != nil {
		return err
	}

	fmt.Printf("\n✓ Model server is running\n")
	fmt.Printf("✓ Endpoint: %s:%s\n", w.host, w.port)
	fmt.Printf("✓ Model: %s\n", w.modelPath)

	return nil
}
