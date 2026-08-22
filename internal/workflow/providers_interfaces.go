package workflow

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
