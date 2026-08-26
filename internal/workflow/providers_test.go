package workflow

import (
	"errors"
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
		{"harbor", "harbor", false},
		{"tuf", "tuf", false},
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
		{"in-toto", "in-toto", false},
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
		{"trivy", "trivy", false},
		{"cdxgen", "cdxgen", false},
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
		name         string
		getProvider  func() (interface{}, error)
		expectedName string
	}{
		{"ArgoCD", func() (interface{}, error) { return GetGitOpsProvider("argo") }, "argocd"},
		{"Flux", func() (interface{}, error) { return GetGitOpsProvider("flux") }, "flux"},
		{"ORAS", func() (interface{}, error) { return GetRegistryProvider("oras") }, "oras"},
		{"ModelPack", func() (interface{}, error) { return GetRegistryProvider("modelpack") }, "modelpack"},
		{"Harbor", func() (interface{}, error) { return GetRegistryProvider("harbor") }, "harbor"},
		{"TUF", func() (interface{}, error) { return GetRegistryProvider("tuf") }, "tuf"},
		{"Sigstore", func() (interface{}, error) { return GetSigningProvider("sigstore") }, "sigstore"},
		{"Notary", func() (interface{}, error) { return GetSigningProvider("notary") }, "notaryv2"},
		{"InToto", func() (interface{}, error) { return GetSigningProvider("in-toto") }, "in-toto"},
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
		name         string
		getProvider  func() (interface{}, error)
		expectedPart string // Partial string to check in instructions
	}{
		{"ArgoCD", func() (interface{}, error) { return GetGitOpsProvider("argo") }, "brew install argoproj"},
		{"Flux", func() (interface{}, error) { return GetGitOpsProvider("flux") }, "brew install fluxcd"},
		{"ORAS", func() (interface{}, error) { return GetRegistryProvider("oras") }, "brew install oras"},
		{"Sigstore", func() (interface{}, error) { return GetSigningProvider("sigstore") }, "brew install sigstore"},
		{"InToto", func() (interface{}, error) { return GetSigningProvider("in-toto") }, "pip install in-toto"},
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
		{"InToto", func() bool { p, _ := GetSigningProvider("in-toto"); return p.IsInstalled() }},
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
		{"Signing", "in-toto", func() (interface{}, error) { return GetSigningProvider("in-toto") }, "in-toto"},
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

// TestAnnotationArgs verifies annotationArgs produces deterministic,
// sorted "--annotation key=value" pairs for the ORAS CLI.
func TestAnnotationArgs(t *testing.T) {
	annotations := map[string]string{
		AnnotationAccelerator: "nvidia-gpu",
		AnnotationRuntime:     "vllm",
	}

	got := annotationArgs(annotations)
	want := []string{
		"--annotation", AnnotationAccelerator + "=nvidia-gpu",
		"--annotation", AnnotationRuntime + "=vllm",
	}

	if len(got) != len(want) {
		t.Fatalf("annotationArgs() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("annotationArgs()[%d] = %q, want %q", i, got[i], want[i])
		}
	}

	if args := annotationArgs(nil); len(args) != 0 {
		t.Errorf("annotationArgs(nil) = %v, want empty", args)
	}
}

// TestModelPackProviderIsNotImplemented verifies every ModelPack registry
// operation fails with ErrNotImplemented instead of pretending to succeed.
func TestModelPackProviderIsNotImplemented(t *testing.T) {
	p, err := GetRegistryProvider("modelpack")
	if err != nil {
		t.Fatalf("GetRegistryProvider(\"modelpack\") error = %v", err)
	}

	ops := map[string]func() error{
		"Push": func() error {
			_, err := p.Push("my-model:v1", "ghcr.io/my-org", t.TempDir(), NewAnnotationSet().ToMap())
			return err
		},
		"Pull":                     func() error { return p.Pull("my-model:v1", "ghcr.io/my-org") },
		"GetArtifactDigest":        func() error { _, err := p.GetArtifactDigest("my-model:v1", "ghcr.io/my-org"); return err },
		"PushReferrer":             func() error { return p.PushReferrer("my-model:v1", "ghcr.io/my-org", "t", nil, nil) },
		"GetReferrers":             func() error { _, err := p.GetReferrers("my-model:v1", "ghcr.io/my-org", "t"); return err },
		"Search":                   func() error { _, err := p.Search("ghcr.io/my-org", ""); return err },
		"FetchManifestAnnotations": func() error { _, err := p.FetchManifestAnnotations("ghcr.io/my-org/my-model:v1"); return err },
	}
	for name, op := range ops {
		if err := op(); !errors.Is(err, ErrNotImplemented) {
			t.Errorf("%s() error = %v, want ErrNotImplemented", name, err)
		}
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

// TestGetSignaturePath verifies signature path formats for signing providers.
func TestGetSignaturePath(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		artifact string
		want     string
	}{
		{"sigstore", "sigstore", "my-model:v1", "my-model:v1.sig"},
		{"notary", "notary", "my-model:v1", "my-model:v1.notation"},
		{"in-toto", "in-toto", "my-model:v1", "my-model:v1.in-toto"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

// TestIsInstalledDoesNotPanic verifies IsInstalled() on every provider returns
// without panicking. Bool value is not asserted since tool availability varies.
func TestIsInstalledDoesNotPanic(t *testing.T) {
	t.Run("ArgoCD", func(t *testing.T) { p, _ := GetGitOpsProvider("argo"); _ = p.IsInstalled() })
	t.Run("Flux", func(t *testing.T) { p, _ := GetGitOpsProvider("flux"); _ = p.IsInstalled() })
	t.Run("ORAS", func(t *testing.T) { p, _ := GetRegistryProvider("oras"); _ = p.IsInstalled() })
	t.Run("ModelPack", func(t *testing.T) { p, _ := GetRegistryProvider("modelpack"); _ = p.IsInstalled() })
	t.Run("Sigstore", func(t *testing.T) { p, _ := GetSigningProvider("sigstore"); _ = p.IsInstalled() })
	t.Run("NotaryV2", func(t *testing.T) { p, _ := GetSigningProvider("notary"); _ = p.IsInstalled() })
	t.Run("InToto", func(t *testing.T) { p, _ := GetSigningProvider("in-toto"); _ = p.IsInstalled() })
	t.Run("Syft", func(t *testing.T) { p, _ := GetSBOMGenerator("syft"); _ = p.IsInstalled() })
	t.Run("vLLM", func(t *testing.T) { p, _ := GetRuntimeProvider("vllm"); _ = p.IsInstalled() })
	t.Run("KServe", func(t *testing.T) { p, _ := GetRuntimeProvider("kserve"); _ = p.IsInstalled() })
}
