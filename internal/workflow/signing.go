package workflow

import (
	"fmt"
	"os/exec"
)

// SigningProvider defines the interface for signing tools (cosign, notation).
// Name() is the option name from SignerOptions.
type SigningProvider interface {
	Name() string
	IsInstalled() bool
	InstallInstructions() string
	Sign(artifact, keyRef string) error
	Verify(artifact string) error
	GetSignaturePath(artifact string) string
}

// --- Sigstore Signing Provider (cosign) ---

type SigstoreProvider struct{}

func (s *SigstoreProvider) Name() string {
	return "cosign"
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
	return "notary"
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

// GetSigningProvider returns the signing provider for an option name from
// SignerOptions. "sigstore" and "notaryv2"/"notation" are accepted aliases.
func GetSigningProvider(name string) (SigningProvider, error) {
	switch name {
	case "sigstore", "cosign":
		return &SigstoreProvider{}, nil
	case "notary", "notaryv2", "notation":
		return &NotaryV2Provider{}, nil
	default:
		return nil, fmt.Errorf("unknown signing provider: %s (supported: cosign, notary)", name)
	}
}
