package workflow

import (
	"fmt"
)

// DeployWorkflow orchestrates the deployment using pluggable providers
type DeployWorkflow struct {
	gitOps             string
	registry           string
	gitOpsProvider     GitOpsProvider
	registryProvider   RegistryProvider
	modelName          string
	modelPath          string // Local path to the model files
	artifactRef        string // Full artifact reference with annotations (Story #63)
	repoURL            string
	manifestPath       string
	runtime            string // Serving runtime (e.g., "kserve-sklearnserver", "kserve-huggingfaceserver")
	hfModelID          string // Hugging Face model ID (e.g., "facebook/opt-125m") for LLM serving
	manifestGenerated  string // Path to generated manifest file, if not yet committed
	deployed           bool // True if a new deploy actually happened (not no-op)
	readyVerified      bool // True if InferenceService reached Ready
	predictionVerified bool // True if a prediction was successfully served
	newCommit          bool // True if a new commit was created
	quiet              bool // If true, skip prologue/banner printing
}

// NewDeployWorkflow creates a new deployment workflow with the given providers
func NewDeployWorkflow(gitOps, registry string) (*DeployWorkflow, error) {
	gitOpsProvider, err := GetGitOpsProvider(gitOps)
	if err != nil {
		return nil, err
	}

	registryProvider, err := GetRegistryProvider(registry)
	if err != nil {
		return nil, err
	}

	return &DeployWorkflow{
		gitOps:           gitOps,
		registry:         registry,
		gitOpsProvider:   gitOpsProvider,
		registryProvider: registryProvider,
	}, nil
}

// SetModelInfo sets the model deployment details
func (w *DeployWorkflow) SetModelInfo(modelName, modelPath, repoURL, manifestPath string) {
	w.modelName = modelName
	w.modelPath = modelPath
	w.repoURL = repoURL
	w.manifestPath = manifestPath
}

// SetRuntime sets the serving runtime for the InferenceService
func (w *DeployWorkflow) SetRuntime(runtime string) {
	w.runtime = runtime
}

// SetHFModelID sets the Hugging Face model ID for LLM serving
func (w *DeployWorkflow) SetHFModelID(hfModelID string) {
	w.hfModelID = hfModelID
}

// SetArtifactRef sets the full artifact reference (with Trust Profile annotations)
// This allows GitOps tools to pull the manifest and access the annotations for admission
func (w *DeployWorkflow) SetArtifactRef(ref string) {
	w.artifactRef = ref
}

// Deployed returns true if a new deploy actually happened (not a no-op)
func (w *DeployWorkflow) Deployed() bool {
	return w.deployed
}

// ReadyVerified returns true if InferenceService reached Ready
func (w *DeployWorkflow) ReadyVerified() bool {
	return w.readyVerified
}

// PredictionVerified returns true if a prediction was successfully served
func (w *DeployWorkflow) PredictionVerified() bool {
	return w.predictionVerified
}

// ManifestGenerated returns the path to the generated manifest file, if not yet committed
func (w *DeployWorkflow) ManifestGenerated() string {
	return w.manifestGenerated
}

// SetQuiet suppresses prologue/banner printing on Run()
func (w *DeployWorkflow) SetQuiet(quiet bool) {
	w.quiet = quiet
}

// NewCommit returns true if a new commit was created during this deploy
func (w *DeployWorkflow) NewCommit() bool {
	return w.newCommit
}

// Run executes the deployment workflow
func (w *DeployWorkflow) Run() error {
	// Reset state for this run
	w.manifestGenerated = ""
	
	if w.modelName == "" || w.repoURL == "" {
		return fmt.Errorf("model info not set: call SetModelInfo before Run()")
	}
	if !w.quiet {
		fmt.Printf("Starting deployment with GitOps: %s, Registry: %s\n", w.gitOps, w.registry)
	}

	// Check if GitOps provider is available
	if !w.gitOpsProvider.IsInstalled() {
		return fmt.Errorf("%s not installed. Install with: %s", w.gitOpsProvider.Name(), w.gitOpsProvider.InstallInstructions())
	}

	// Check if registry provider is available
	if !w.registryProvider.IsInstalled() {
		return fmt.Errorf("%s not installed. Install with: %s", w.registryProvider.Name(), w.registryProvider.InstallInstructions())
	}

	// Deploy using GitOps provider
	// For Story #63: Pass artifact reference so GitOps tools can access Trust Profile annotations
	if w.repoURL == "" {
		return fmt.Errorf("repository URL is required for GitOps deployment")
	}
	if w.manifestPath == "" {
		return fmt.Errorf("manifest path is required for GitOps deployment")
	}

	artifactToDeploy := w.modelName
	if w.artifactRef != "" {
		artifactToDeploy = w.artifactRef
	}
	if !w.quiet {
		fmt.Printf("Deploying artifact '%s' from repository '%s' with %s...\n",
			artifactToDeploy, w.repoURL, w.gitOps)
		fmt.Printf("  Trust Profile annotations will be available to GitOps admission policies\n")
	}

	// Propagate quiet flag to the underlying provider
	w.gitOpsProvider.SetQuiet(w.quiet)

	deployResult := w.gitOpsProvider.Deploy(artifactToDeploy, w.repoURL, w.manifestPath, w.modelPath, w.runtime, w.hfModelID)
	if deployResult.Error != nil {
		return deployResult.Error
	}
	
	// If manifest was generated but not committed, return it to the caller
	// The caller (cmd/wizard) will handle the commit prompt
	if deployResult.ManifestGenerated != "" {
		w.manifestGenerated = deployResult.ManifestGenerated
		return nil
	}

	w.deployed = deployResult.Deployed
	w.readyVerified = deployResult.Ready
	w.predictionVerified = deployResult.PredictionVerified
	w.newCommit = deployResult.NewCommit

	// Use registry provider
	if w.registry == "oras" || w.registry == "modelpack" {
		fmt.Printf("Using %s for registry operations\n", w.registry)
		// For now, just indicate the provider is ready
		// Actual push/pull will be added when we integrate with packaging commands
	}

	fmt.Println("Deployment workflow complete!")
	return nil
}
