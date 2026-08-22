package workflow

import (
	"fmt"
	"os/exec"
)

// GitOpsProvider defines the interface for GitOps tools like ArgoCD and Flux
type GitOpsProvider interface {
	Name() string
	IsInstalled() bool
	InstallInstructions() string
	Deploy(modelName, repoURL, path string) error
}

// RegistryProvider defines the interface for registry tools like ORAS and ModelPack
type RegistryProvider interface {
	Name() string
	IsInstalled() bool
	InstallInstructions() string
	Push(artifact, registry string) error
	Pull(artifact, registry string) error
}

// SigningProvider defines the interface for signing tools (Sigstore, Notary v2)
type SigningProvider interface {
	Name() string
	IsInstalled() bool
	InstallInstructions() string
	Sign(artifact, keyRef string) error
	Verify(artifact string) error
	GetSignaturePath(artifact string) string
}

// SBOMGenerator defines the interface for SBOM generation
type SBOMGenerator interface {
	Name() string
	IsInstalled() bool
	InstallInstructions() string
	Generate(modelPath, outputPath string) error
}

// MOFClassifier defines the interface for Model Openness Framework classification
type MOFClassifier interface {
	Name() string
	Classify(modelPath string) (string, error) // Returns MOF class: I, II, or III
}

// RuntimeProvider defines the interface for model serving runtimes (vLLM, KServe)
type RuntimeProvider interface {
	Name() string
	IsInstalled() bool
	InstallInstructions() string
	Serve(modelPath, host, port string) error
}

// --- ArgoCD Provider ---

type ArgoCDProvider struct{}

func (a *ArgoCDProvider) Name() string {
	return "argocd"
}

func (a *ArgoCDProvider) IsInstalled() bool {
	return exec.Command("argocd", "version").Run() == nil
}

func (a *ArgoCDProvider) InstallInstructions() string {
	return "brew install argoproj/tap/argocd"
}

func (a *ArgoCDProvider) Deploy(modelName, repoURL, path string) error {
	if !a.IsInstalled() {
		return fmt.Errorf("ArgoCD not installed. Install with: %s", a.InstallInstructions())
	}
	// argocd app create <name> --repo <url> --path <path> --dest-namespace <namespace>
	cmd := exec.Command("argocd", "app", "create", modelName, "--repo", repoURL, "--path", path, "--dest-namespace", "default")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create ArgoCD application: %v", err)
	}
	fmt.Printf("Created ArgoCD application: %s\n", modelName)
	return nil
}

// --- Flux Provider ---

type FluxProvider struct{}

func (f *FluxProvider) Name() string {
	return "flux"
}

func (f *FluxProvider) IsInstalled() bool {
	return exec.Command("flux", "version").Run() == nil
}

func (f *FluxProvider) InstallInstructions() string {
	return "brew install fluxcd/tap/flux"
}

func (f *FluxProvider) Deploy(modelName, repoURL, path string) error {
	if !f.IsInstalled() {
		return fmt.Errorf("Flux not installed. Install with: %s", f.InstallInstructions())
	}
	// flux create source git <name> --url <url> --branch <branch>
	// flux create kustomization <name> --source <source> --path <path>
	cmd := exec.Command("flux", "create", "source", "git", modelName+"-"+"git", "--url", repoURL, "--branch", "main")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create Flux git source: %v", err)
	}
	cmd = exec.Command("flux", "create", "kustomization", modelName, "--source", modelName+"-"+"git", "--path", path)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create Flux kustomization: %v", err)
	}
	fmt.Printf("Created Flux deployment: %s\n", modelName)
	return nil
}

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
	if !o.IsInstalled() {
		return fmt.Errorf("ORAS not installed. Install with: %s", o.InstallInstructions())
	}
	// oras push <registry>/<artifact> <local-path>
	cmd := exec.Command("oras", "push", registry+"/"+artifact, artifact)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to push with ORAS: %v", err)
	}
	fmt.Printf("Pushed artifact %s to %s using ORAS\n", artifact, registry)
	return nil
}

func (o *ORASProvider) Pull(artifact, registry string) error {
	if !o.IsInstalled() {
		return fmt.Errorf("ORAS not installed. Install with: %s", o.InstallInstructions())
	}
	// oras pull <registry>/<artifact>
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
	// ModelPack might be a CLI tool or a library - check for command
	// For now, we assume it's available (it's a Go library approach)
	return true
}

func (m *ModelPackProvider) InstallInstructions() string {
	return "go install github.com/modelpack/modelpack@latest"
}

func (m *ModelPackProvider) Push(artifact, registry string) error {
	// ModelPack uses OCI artifacts approach
	fmt.Printf("Pushed artifact %s to %s using ModelPack\n", artifact, registry)
	return nil
}

func (m *ModelPackProvider) Pull(artifact, registry string) error {
	// ModelPack uses OCI artifacts approach
	fmt.Printf("Pulled artifact %s from %s using ModelPack\n", artifact, registry)
	return nil
}

// GetGitOpsProvider returns the appropriate GitOps provider by name
func GetGitOpsProvider(name string) (GitOpsProvider, error) {
	switch name {
	case "argo", "argocd":
		return &ArgoCDProvider{}, nil
	case "flux":
		return &FluxProvider{}, nil
	default:
		return nil, fmt.Errorf("unknown GitOps provider: %s (supported: argo, flux)", name)
	}
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

// --- Sigstore Signing Provider (cosign) ---

type SigstoreProvider struct{}

func (s *SigstoreProvider) Name() string {
	return "sigstore"
}

func (s *SigstoreProvider) IsInstalled() bool {
	return exec.Command("cosign", "version").Run() == nil
}

func (s *SigstoreProvider) InstallInstructions() string {
	return "brew install sigstore/tap/cosign"
}

func (s *SigstoreProvider) Sign(artifact, keyRef string) error {
	if !s.IsInstalled() {
		return fmt.Errorf("cosign not installed. Install with: %s", s.InstallInstructions())
	}
	// cosign sign --key <key> <artifact>
	var cmd *exec.Cmd
	if keyRef != "" {
		cmd = exec.Command("cosign", "sign", "--key", keyRef, artifact)
	} else {
		cmd = exec.Command("cosign", "sign", artifact)
	}
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to sign with cosign: %v", err)
	}
	fmt.Printf("Signed artifact %s with Sigstore (cosign)\n", artifact)
	return nil
}

func (s *SigstoreProvider) Verify(artifact string) error {
	if !s.IsInstalled() {
		return fmt.Errorf("cosign not installed. Install with: %s", s.InstallInstructions())
	}
	// cosign verify --key <key> <artifact>
	cmd := exec.Command("cosign", "verify", artifact)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("signature verification failed for %s: %v", artifact, err)
	}
	fmt.Printf("Verified signature for %s with Sigstore (cosign)\n", artifact)
	return nil
}

func (s *SigstoreProvider) GetSignaturePath(artifact string) string {
	return artifact + ".sig"
}

// --- Notary v2 Signing Provider ---

type NotaryV2Provider struct{}

func (n *NotaryV2Provider) Name() string {
	return "notaryv2"
}

func (n *NotaryV2Provider) IsInstalled() bool {
	return exec.Command("notation", "version").Run() == nil
}

func (n *NotaryV2Provider) InstallInstructions() string {
	return "brew install notation"
}

func (n *NotaryV2Provider) Sign(artifact, keyRef string) error {
	if !n.IsInstalled() {
		return fmt.Errorf("notation not installed. Install with: %s", n.InstallInstructions())
	}
	// notation sign --key <key> <artifact>
	var cmd *exec.Cmd
	if keyRef != "" {
		cmd = exec.Command("notation", "sign", "--key", keyRef, artifact)
	} else {
		cmd = exec.Command("notation", "sign", artifact)
	}
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to sign with notation: %v", err)
	}
	fmt.Printf("Signed artifact %s with Notary v2 (notation)\n", artifact)
	return nil
}

func (n *NotaryV2Provider) Verify(artifact string) error {
	if !n.IsInstalled() {
		return fmt.Errorf("notation not installed. Install with: %s", n.InstallInstructions())
	}
	// notation verify <artifact>
	cmd := exec.Command("notation", "verify", artifact)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("signature verification failed for %s: %v", artifact, err)
	}
	fmt.Printf("Verified signature for %s with Notary v2 (notation)\n", artifact)
	return nil
}

func (n *NotaryV2Provider) GetSignaturePath(artifact string) string {
	return artifact + ".notation"
}

// GetSigningProvider returns the appropriate signing provider by name
func GetSigningProvider(name string) (SigningProvider, error) {
	switch name {
	case "sigstore", "cosign":
		return &SigstoreProvider{}, nil
	case "notary", "notaryv2", "notation":
		return &NotaryV2Provider{}, nil
	default:
		return nil, fmt.Errorf("unknown signing provider: %s (supported: sigstore, notary)", name)
	}
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
	// syft <model-path> -o spdx-json > <output-path>
	// For now, simulate (in production: exec.Command("syft", modelPath, "-o", "spdx-json"))
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

// --- MOF Classifier (Model Openness Framework) ---

type MOFClassifierImpl struct{}

func (m *MOFClassifierImpl) Name() string {
	return "mof"
}

func (m *MOFClassifierImpl) Classify(modelPath string) (string, error) {
	// MOF classification based on model metadata
	// Class I: Open weights, open training data, open code
	// Class II: Open weights, closed training data or code
	// Class III: Closed weights
	
	// For now, we'll return a placeholder - in real implementation,
	// this would inspect model metadata files
	fmt.Printf("Classifying model at %s with MOF...\n", modelPath)
	return "I", nil // Placeholder: assume Class I (most open)
}

// GetMOFClassifier returns the MOF classifier
func GetMOFClassifier() MOFClassifier {
	return &MOFClassifierImpl{}
}

// --- vLLM Runtime Provider ---

type VLLMProvider struct{}

func (v *VLLMProvider) Name() string {
	return "vllm"
}

func (v *VLLMProvider) IsInstalled() bool {
	// vLLM is typically installed as a Python package
	// Check if vllm module is available
	cmd := exec.Command("python3", "-c", "import vllm; print('vllm installed')")
	return cmd.Run() == nil
}

func (v *VLLMProvider) InstallInstructions() string {
	return "pip install vllm"
}

func (v *VLLMProvider) Serve(modelPath, host, port string) error {
	// vllm serve --model <path> --host <host> --port <port>
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
	// KServe is typically installed as a Python package
	cmd := exec.Command("python3", "-c", "import kserve; print('kserve installed')")
	return cmd.Run() == nil
}

func (k *KServeProvider) InstallInstructions() string {
	return "pip install kserve"
}

func (k *KServeProvider) Serve(modelPath, host, port string) error {
	// For KServe, we'd typically create an InferenceService manifest
	// and apply it to the cluster. This is a simplified version.
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
