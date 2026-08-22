package workflow

import (
	"os"
	"testing"
)

// Test GitOpsProvider factory
func TestGetGitOpsProvider(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"argo", "argo", false},
		{"argocd", "argocd", false},
		{"flux", "flux", false},
		{"invalid", "invalid-tool", true},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetGitOpsProvider(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetGitOpsProvider(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got == nil {
				t.Errorf("GetGitOpsProvider(%q) = nil, want non-nil", tt.input)
			}
		})
	}
}

// Test RegistryProvider factory
func TestGetRegistryProvider(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"oras", "oras", false},
		{"modelpack", "modelpack", false},
		{"invalid", "invalid-registry", true},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetRegistryProvider(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetRegistryProvider(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got == nil {
				t.Errorf("GetRegistryProvider(%q) = nil, want non-nil", tt.input)
			}
		})
	}
}

// Test SigningProvider factory
func TestGetSigningProvider(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"sigstore", "sigstore", false},
		{"cosign", "cosign", false},
		{"notary", "notary", false},
		{"notaryv2", "notaryv2", false},
		{"notation", "notation", false},
		{"invalid", "invalid-signer", true},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetSigningProvider(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetSigningProvider(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got == nil {
				t.Errorf("GetSigningProvider(%q) = nil, want non-nil", tt.input)
			}
		})
	}
}

// Test RuntimeProvider factory
func TestGetRuntimeProvider(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"vllm", "vllm", false},
		{"kserve", "kserve", false},
		{"invalid", "invalid-runtime", true},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetRuntimeProvider(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetRuntimeProvider(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got == nil {
				t.Errorf("GetRuntimeProvider(%q) = nil, want non-nil", tt.input)
			}
		})
	}
}

// Test SBOMGenerator factory
func TestGetSBOMGenerator(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"syft", "syft", false},
		{"invalid", "invalid-sbom", true},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetSBOMGenerator(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetSBOMGenerator(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got == nil {
				t.Errorf("GetSBOMGenerator(%q) = nil, want non-nil", tt.input)
			}
		})
	}
}

// Test provider Name() methods
func TestProviderNames(t *testing.T) {
	providers := []struct {
		name          string
		getProvider   func() (interface{}, error)
		expectedName  string
	}{
		{"ArgoCD", func() (interface{}, error) { return GetGitOpsProvider("argo") }, "argocd"},
		{"Flux", func() (interface{}, error) { return GetGitOpsProvider("flux") }, "flux"},
		{"ORAS", func() (interface{}, error) { return GetRegistryProvider("oras") }, "oras"},
		{"ModelPack", func() (interface{}, error) { return GetRegistryProvider("modelpack") }, "modelpack"},
		{"Sigstore", func() (interface{}, error) { return GetSigningProvider("sigstore") }, "sigstore"},
		{"Notary", func() (interface{}, error) { return GetSigningProvider("notary") }, "notaryv2"},
		{"vLLM", func() (interface{}, error) { return GetRuntimeProvider("vllm") }, "vllm"},
		{"KServe", func() (interface{}, error) { return GetRuntimeProvider("kserve") }, "kserve"},
	}

	for _, tt := range providers {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := tt.getProvider()
			if err != nil {
				t.Fatalf("Failed to get provider: %v", err)
			}

			// Use type assertion to call Name()
			switch p := provider.(type) {
			case GitOpsProvider:
				if p.Name() != tt.expectedName {
					t.Errorf("Name() = %q, want %q", p.Name(), tt.expectedName)
				}
			case RegistryProvider:
				if p.Name() != tt.expectedName {
					t.Errorf("Name() = %q, want %q", p.Name(), tt.expectedName)
				}
			case SigningProvider:
				if p.Name() != tt.expectedName {
					t.Errorf("Name() = %q, want %q", p.Name(), tt.expectedName)
				}
			case RuntimeProvider:
				if p.Name() != tt.expectedName {
					t.Errorf("Name() = %q, want %q", p.Name(), tt.expectedName)
				}
			}
		})
	}
}

// Test provider InstallInstructions() methods
func TestInstallInstructions(t *testing.T) {
	providers := []struct {
		name          string
		getProvider   func() (interface{}, error)
		expectedPart  string // Partial string to check in instructions
	}{
		{"ArgoCD", func() (interface{}, error) { return GetGitOpsProvider("argo") }, "brew install argoproj"},
		{"Flux", func() (interface{}, error) { return GetGitOpsProvider("flux") }, "brew install fluxcd"},
		{"ORAS", func() (interface{}, error) { return GetRegistryProvider("oras") }, "brew install oras"},
		{"Sigstore", func() (interface{}, error) { return GetSigningProvider("sigstore") }, "brew install sigstore"},
		{"vLLM", func() (interface{}, error) { return GetRuntimeProvider("vllm") }, "pip install vllm"},
	}

	for _, tt := range providers {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := tt.getProvider()
			if err != nil {
				t.Fatalf("Failed to get provider: %v", err)
			}

			var instructions string
			switch p := provider.(type) {
			case GitOpsProvider:
				instructions = p.InstallInstructions()
			case RegistryProvider:
				instructions = p.InstallInstructions()
			case SigningProvider:
				instructions = p.InstallInstructions()
			case RuntimeProvider:
				instructions = p.InstallInstructions()
			}

			if !contains(instructions, tt.expectedPart) {
				t.Errorf("InstallInstructions() = %q, expected to contain %q", instructions, tt.expectedPart)
			}
		})
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (len(substr) == 0 || findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestAllProvidersIsInstalledNoPanic(t *testing.T) {
	providers := []struct {
		name string
		fn   func() bool
	}{
		{"ArgoCD", func() bool { p, _ := GetGitOpsProvider("argo"); return p.IsInstalled() }},
		{"Flux", func() bool { p, _ := GetGitOpsProvider("flux"); return p.IsInstalled() }},
		{"ORAS", func() bool { p, _ := GetRegistryProvider("oras"); return p.IsInstalled() }},
		{"ModelPack", func() bool { p, _ := GetRegistryProvider("modelpack"); return p.IsInstalled() }},
		{"Sigstore", func() bool { p, _ := GetSigningProvider("sigstore"); return p.IsInstalled() }},
		{"NotaryV2", func() bool { p, _ := GetSigningProvider("notary"); return p.IsInstalled() }},
		{"vLLM", func() bool { p, _ := GetRuntimeProvider("vllm"); return p.IsInstalled() }},
		{"KServe", func() bool { p, _ := GetRuntimeProvider("kserve"); return p.IsInstalled() }},
	}

	for _, tt := range providers {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("IsInstalled() panicked: %v", r)
				}
			}()
			_ = tt.fn()
		})
	}
}

func TestSigningProviderGetSignaturePath(t *testing.T) {
	tests := []struct {
		provider string
		artifact string
		want     string
	}{
		{"sigstore", "my-model:v1", "my-model:v1.sig"},
		{"notary", "my-model:v1", "my-model:v1.notation"},
	}

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			p, err := GetSigningProvider(tt.provider)
			if err != nil {
				t.Fatalf("GetSigningProvider(%q) error = %v", tt.provider, err)
			}
			got := p.GetSignaturePath(tt.artifact)
			if got != tt.want {
				t.Errorf("GetSignaturePath(%q) = %q, want %q", tt.artifact, got, tt.want)
			}
		})
	}
}

// Test provider Name() method consistency for error messages
func TestProviderNameConsistency(t *testing.T) {
	// Test that each provider's Name() returns the expected value
	// This ensures error messages use consistent naming
	nameTests := []struct {
		providerType string
		providerName string
		getFunc      func() (interface{}, error)
		expected     string
	}{
		{"GitOps", "argo", func() (interface{}, error) { return GetGitOpsProvider("argo") }, "argocd"},
		{"GitOps", "flux", func() (interface{}, error) { return GetGitOpsProvider("flux") }, "flux"},
		{"Registry", "oras", func() (interface{}, error) { return GetRegistryProvider("oras") }, "oras"},
		{"Registry", "modelpack", func() (interface{}, error) { return GetRegistryProvider("modelpack") }, "modelpack"},
		{"Signing", "sigstore", func() (interface{}, error) { return GetSigningProvider("sigstore") }, "sigstore"},
		{"Signing", "notary", func() (interface{}, error) { return GetSigningProvider("notary") }, "notaryv2"},
		{"Runtime", "vllm", func() (interface{}, error) { return GetRuntimeProvider("vllm") }, "vllm"},
		{"Runtime", "kserve", func() (interface{}, error) { return GetRuntimeProvider("kserve") }, "kserve"},
	}

	for _, tt := range nameTests {
		t.Run(tt.providerType+"_"+tt.providerName, func(t *testing.T) {
			provider, err := tt.getFunc()
			if err != nil {
				t.Fatalf("Failed to get %s provider %s: %v", tt.providerType, tt.providerName, err)
			}

			var name string
			switch p := provider.(type) {
			case GitOpsProvider:
				name = p.Name()
			case RegistryProvider:
				name = p.Name()
			case SigningProvider:
				name = p.Name()
			case RuntimeProvider:
				name = p.Name()
			}

			if name != tt.expected {
				t.Errorf("%s provider %s Name() = %q, want %q", tt.providerType, tt.providerName, name, tt.expected)
			}
		})
	}
}

func TestModelPackProviderIsInstalled(t *testing.T) {
	original := os.Getenv("PATH")
	os.Setenv("PATH", "")
	defer os.Setenv("PATH", original)

	p, err := GetRegistryProvider("modelpack")
	if err != nil {
		t.Fatalf("GetRegistryProvider(\"modelpack\") error = %v", err)
	}
	if p.IsInstalled() {
		t.Error("ModelPackProvider.IsInstalled() = true with empty PATH, want false")
	}
}
