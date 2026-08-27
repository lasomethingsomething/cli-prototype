package workflow

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// RegistryProvider defines the interface for registry tools like ORAS and ModelPack
type RegistryProvider interface {
	Name() string
	IsInstalled() bool
	InstallInstructions() string
	// Push uploads the file or directory at sourcePath to registry as
	// artifact, attaching annotations (e.g. from AnnotationSet.ToMap()) to
	// the resulting OCI manifest. annotations may be nil or empty when no
	// manifest-level annotations should be set. It returns the digest of
	// the pushed manifest, or "" if the tool did not report one.
	Push(artifact, registry, sourcePath string, annotations map[string]string) (string, error)
	Pull(artifact, registry string) error
	// GetArtifactDigest returns the manifest digest of an artifact as stored
	// in the registry. Parity verification compares it with the digest
	// reported by Push.
	GetArtifactDigest(artifact, registry string) (string, error)
	// PushReferrer pushes a referrer (like provenance attestation) to the registry
	// The referrer is associated with the artifact and can be fetched later
	PushReferrer(artifact, registry, referrerType string, data []byte, annotations map[string]string) error
	// GetReferrers fetches all referrers of a given type for an artifact from the registry
	GetReferrers(artifact, registry, referrerType string) ([][]byte, error)
	// Search lists the tagged manifests in registry that carry annotations
	// matching query's artifact-type and metadata filters (see
	// SearchQuery.MatchesAnnotations); a provider may evaluate those
	// filters server-side. Relationship filters are applied afterwards by
	// SearchResults.FilterByRelationship.
	Search(registry string, query *SearchQuery) ([]ManifestCandidate, error)
	// FetchManifestAnnotations fetches the OCI manifest for an artifact and returns its annotations
	// This is used for GitOps pre-sync validation (Story #65)
	FetchManifestAnnotations(artifactRef string) (map[string]string, error)
}

// annotationArgs converts an annotation map into repeated "--annotation
// key=value" arguments, sorted by key for deterministic ordering.
func annotationArgs(annotations map[string]string) []string {
	keys := make([]string, 0, len(annotations))
	for k := range annotations {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	args := make([]string, 0, len(keys)*2)
	for _, k := range keys {
		args = append(args, "--annotation", fmt.Sprintf("%s=%s", k, annotations[k]))
	}
	return args
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

func (o *ORASProvider) Push(artifact, registry, sourcePath string, annotations map[string]string) (string, error) {
	absSource, err := filepath.Abs(sourcePath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve source path %q: %v", sourcePath, err)
	}
	if _, err := os.Stat(absSource); err != nil {
		return "", fmt.Errorf("source path %q is not accessible: %v", sourcePath, err)
	}

	args := []string{"push", registry + "/" + artifact, "--artifact-type", artifactTypeFor(annotations), "--format", "json"}
	args = append(args, annotationArgs(annotations)...)
	configArgs, err := configBlobArgs(absSource)
	if err != nil {
		return "", err
	}
	args = append(args, configArgs...)
	// Run from the parent directory and push the base name so that the layer
	// title ORAS records is the model directory (or file) name, not an
	// absolute path (which ORAS rejects without --disable-path-validation).
	args = append(args, filepath.Base(absSource))

	cmd := exec.Command("oras", args...)
	cmd.Dir = filepath.Dir(absSource)
	cmd.Stderr = os.Stderr
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to push with ORAS: %v", err)
	}
	fmt.Printf("Pushed artifact %s to %s using ORAS\n", artifact, registry)
	if len(annotations) > 0 {
		fmt.Printf("Attached %d CNCF AI annotation(s) to the manifest\n", len(annotations))
	}
	return digestFromORASOutput(output), nil
}

// configBlobArgs returns the "--config <file>:<mediatype>" arguments that
// make ORAS push the AI config blob written by `package` as the manifest's
// config, so the registry manifest's config descriptor is the real blob and
// matches the local manifest.json. It returns nil when absSource is not a
// packaged directory, i.e. it lacks config.json or manifest.json (a model
// directory may ship its own unrelated config.json). The file path is
// relative to the parent of absSource, which is where Push runs ORAS.
func configBlobArgs(absSource string) ([]string, error) {
	if _, err := os.Stat(filepath.Join(absSource, ConfigBlobFileName)); err != nil {
		return nil, nil
	}
	manifestPath := filepath.Join(absSource, "manifest.json")
	if _, err := os.Stat(manifestPath); err != nil {
		return nil, nil
	}
	manifest, err := ReadUnifiedOCIManifest(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("cannot push config blob: %v", err)
	}
	if manifest.Config.MediaType == "" {
		return nil, fmt.Errorf("cannot push config blob: %s has no config media type", manifestPath)
	}
	configRef := filepath.Join(filepath.Base(absSource), ConfigBlobFileName)
	return []string{"--config", configRef + ":" + manifest.Config.MediaType}, nil
}

// digestFromORASOutput extracts the manifest digest from `oras ... --format
// json` output. Older ORAS versions ignore --format and print a
// "Digest: sha256:..." line instead, which is handled by the fallback scan.
func digestFromORASOutput(output []byte) string {
	var parsed struct {
		Digest string `json:"digest"`
	}
	if err := json.Unmarshal(output, &parsed); err == nil && parsed.Digest != "" {
		return parsed.Digest
	}
	return digestPattern.FindString(string(output))
}

var digestPattern = regexp.MustCompile(`sha256:[0-9a-f]{64}`)

// artifactTypeFor derives the OCI artifactType for a push from the
// org.cncf.ai.artifact.type annotation, defaulting to a model.
func artifactTypeFor(annotations map[string]string) string {
	t := annotations[AnnotationArtifactType]
	if t == "" {
		t = "model"
	}
	return "application/vnd.cncf.ai." + t
}

func (o *ORASProvider) Pull(artifact, registry string) error {
	cmd := exec.Command("oras", "pull", registry+"/"+artifact)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to pull with ORAS: %v", err)
	}
	fmt.Printf("Pulled artifact %s from %s using ORAS\n", artifact, registry)
	return nil
}

func (o *ORASProvider) GetArtifactDigest(artifact, registry string) (string, error) {
	// `oras manifest fetch --descriptor` prints the OCI descriptor of the
	// manifest as stored in the registry, including its content digest.
	cmd := exec.Command("oras", "manifest", "fetch", "--descriptor", registry+"/"+artifact)
	cmd.Stderr = os.Stderr
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to fetch manifest descriptor with ORAS: %v", err)
	}
	var descriptor struct {
		Digest string `json:"digest"`
	}
	if err := json.Unmarshal(output, &descriptor); err != nil {
		return "", fmt.Errorf("failed to parse manifest descriptor from ORAS: %v", err)
	}
	if descriptor.Digest == "" {
		return "", fmt.Errorf("ORAS manifest descriptor for %s/%s has no digest", registry, artifact)
	}
	return descriptor.Digest, nil
}

// PushReferrer attaches data to artifact as an OCI referrer of the given
// artifact type with `oras attach`, leaving the subject manifest untouched.
func (o *ORASProvider) PushReferrer(artifact, registry, referrerType string, data []byte, annotations map[string]string) error {
	subject := registry + "/" + artifact

	tmpDir, err := os.MkdirTemp("", "referrer-*")
	if err != nil {
		return fmt.Errorf("failed to create temp dir for referrer: %v", err)
	}
	defer os.RemoveAll(tmpDir)
	const fileName = "referrer.json"
	if err := os.WriteFile(filepath.Join(tmpDir, fileName), data, 0644); err != nil {
		return fmt.Errorf("failed to write referrer data: %v", err)
	}

	args := []string{"attach", subject, "--artifact-type", referrerType}
	args = append(args, annotationArgs(annotations)...)
	// <file>:<layer media type>; run from the temp dir so ORAS records a plain title.
	args = append(args, fileName+":"+referrerType)

	cmd := exec.Command("oras", args...)
	cmd.Dir = tmpDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to attach referrer with ORAS: %v", err)
	}
	fmt.Printf("Attached %s referrer to %s\n", referrerType, subject)
	return nil
}

// GetReferrers lists the referrers of the given artifact type with
// `oras discover` and returns the content of each referrer's layer blobs
// (e.g. the attestation JSON pushed by PushReferrer).
func (o *ORASProvider) GetReferrers(artifact, registry, referrerType string) ([][]byte, error) {
	fullArtifact := registry + "/" + artifact
	repo := registry + "/" + repositoryOf(artifact)

	discover := exec.Command("oras", "discover", "--artifact-type", referrerType, "--format", "json", fullArtifact)
	discover.Stderr = os.Stderr
	output, err := discover.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to discover referrers with ORAS: %v", err)
	}

	refs, err := parseDiscoverOutput(output)
	if err != nil {
		return nil, err
	}

	var blobs [][]byte
	for _, ref := range refs {
		if ref.Digest == "" {
			continue
		}
		fetch := exec.Command("oras", "manifest", "fetch", repo+"@"+ref.Digest)
		fetch.Stderr = os.Stderr
		manifestJSON, err := fetch.Output()
		if err != nil {
			return nil, fmt.Errorf("failed to fetch referrer manifest %s with ORAS: %v", ref.Digest, err)
		}
		var manifest struct {
			Layers []struct {
				Digest string `json:"digest"`
			} `json:"layers"`
		}
		if err := json.Unmarshal(manifestJSON, &manifest); err != nil {
			return nil, fmt.Errorf("failed to parse referrer manifest %s: %v", ref.Digest, err)
		}
		for _, layer := range manifest.Layers {
			blob := exec.Command("oras", "blob", "fetch", "--output", "-", repo+"@"+layer.Digest)
			blob.Stderr = os.Stderr
			data, err := blob.Output()
			if err != nil {
				return nil, fmt.Errorf("failed to fetch referrer blob %s with ORAS: %v", layer.Digest, err)
			}
			blobs = append(blobs, data)
		}
	}
	return blobs, nil
}

// discoveredReferrer is one entry of `oras discover --format json`.
type discoveredReferrer struct {
	Digest       string `json:"digest"`
	ArtifactType string `json:"artifactType"`
}

// parseDiscoverOutput reads the referrer list from `oras discover --format
// json`. ORAS 1.2 lists them under "manifests", ORAS 1.3+ under "referrers";
// both are accepted.
func parseDiscoverOutput(output []byte) ([]discoveredReferrer, error) {
	var discovered struct {
		Manifests []discoveredReferrer `json:"manifests"`
		Referrers []discoveredReferrer `json:"referrers"`
	}
	if err := json.Unmarshal(output, &discovered); err != nil {
		return nil, fmt.Errorf("failed to parse ORAS discover output: %v", err)
	}
	return append(discovered.Manifests, discovered.Referrers...), nil
}

// repositoryOf strips the tag or digest from an artifact reference such as
// "my-model:v1" or "my-model@sha256:...", leaving the repository path.
func repositoryOf(artifact string) string {
	if i := strings.Index(artifact, "@"); i >= 0 {
		return artifact[:i]
	}
	// A colon after the last slash separates the tag; earlier colons could be a port.
	lastSlash := strings.LastIndex(artifact, "/")
	if i := strings.LastIndex(artifact, ":"); i > lastSlash {
		return artifact[:i]
	}
	return artifact
}

// Search walks the registry with the ORAS CLI: `oras repo ls` lists the
// repositories, `oras repo tags` the tags of each, and every tag's manifest
// is fetched with `oras manifest fetch`. ORAS has no registry-wide search,
// so the query's artifact-type and metadata filters are applied client-side
// to each manifest's annotations. Manifests without annotations are not AI
// artifacts and are skipped; a repository whose tags cannot be listed (or a
// tag whose manifest cannot be fetched) is reported on stderr and skipped so
// that one broken repository does not hide the others.
//
// Like every other ORAS call here, authentication, TLS and plain-HTTP for
// localhost are left to the oras CLI and its own configuration.
func (o *ORASProvider) Search(registry string, query *SearchQuery) ([]ManifestCandidate, error) {
	repos, err := orasLines("repo", "ls", registry)
	if err != nil {
		return nil, fmt.Errorf("failed to list repositories of %s with ORAS: %v", registry, err)
	}

	var candidates []ManifestCandidate
	for _, repo := range repos {
		repoRef := registry + "/" + repo
		tags, err := orasLines("repo", "tags", repoRef)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: skipping %s: failed to list tags with ORAS: %v\n", repoRef, err)
			continue
		}
		for _, tag := range tags {
			ref := repoRef + ":" + tag
			candidate, err := o.fetchCandidate(ref)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: skipping %s: %v\n", ref, err)
				continue
			}
			if len(candidate.Annotations) == 0 || !query.MatchesAnnotations(candidate.Annotations) {
				continue
			}
			candidates = append(candidates, candidate)
		}
	}
	return candidates, nil
}

// fetchCandidate resolves ref to its manifest digest (`oras manifest fetch
// --descriptor`) and body (`oras manifest fetch`).
func (o *ORASProvider) fetchCandidate(ref string) (ManifestCandidate, error) {
	descriptorJSON, err := orasOutput("manifest", "fetch", "--descriptor", ref)
	if err != nil {
		return ManifestCandidate{}, fmt.Errorf("failed to fetch manifest descriptor with ORAS: %v", err)
	}
	var descriptor struct {
		Digest string `json:"digest"`
	}
	if err := json.Unmarshal(descriptorJSON, &descriptor); err != nil {
		return ManifestCandidate{}, fmt.Errorf("failed to parse manifest descriptor from ORAS: %v", err)
	}

	manifest, err := orasOutput("manifest", "fetch", ref)
	if err != nil {
		return ManifestCandidate{}, fmt.Errorf("failed to fetch manifest with ORAS: %v", err)
	}
	return NewManifestCandidate(ref, descriptor.Digest, manifest)
}

// orasOutput runs oras with args and returns its stdout. When oras fails,
// whatever it printed to stderr is included in the error.
func orasOutput(args ...string) ([]byte, error) {
	output, err := exec.Command("oras", args...).Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && len(exitErr.Stderr) > 0 {
			return nil, fmt.Errorf("%v: %s", err, strings.TrimSpace(string(exitErr.Stderr)))
		}
		return nil, err
	}
	return output, nil
}

// orasLines runs oras and returns the non-empty lines of its stdout, which is
// how `oras repo ls` and `oras repo tags` report their results.
func orasLines(args ...string) ([]string, error) {
	output, err := orasOutput(args...)
	if err != nil {
		return nil, err
	}
	var lines []string
	for _, line := range strings.Split(string(output), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			lines = append(lines, line)
		}
	}
	return lines, nil
}

func (o *ORASProvider) FetchManifestAnnotations(artifactRef string) (map[string]string, error) {
	// Use oras manifest fetch to get the manifest JSON, then extract annotations
	cmd := exec.Command("oras", "manifest", "fetch", artifactRef)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch manifest with ORAS: %v", err)
	}

	// Parse the manifest JSON to extract annotations
	var manifestData map[string]interface{}
	if err := json.Unmarshal(output, &manifestData); err != nil {
		return nil, fmt.Errorf("failed to parse manifest JSON: %v", err)
	}

	// Extract annotations if present
	annotations := make(map[string]string)
	if annotationsMap, ok := manifestData["annotations"].(map[string]interface{}); ok {
		for k, v := range annotationsMap {
			if vStr, ok := v.(string); ok {
				annotations[k] = vStr
			}
		}
	}

	// If no annotations found in top-level, try config.annotations
	if len(annotations) == 0 {
		if config, ok := manifestData["config"].(map[string]interface{}); ok {
			if annotationsMap, ok := config["annotations"].(map[string]interface{}); ok {
				for k, v := range annotationsMap {
					if vStr, ok := v.(string); ok {
						annotations[k] = vStr
					}
				}
			}
		}
	}

	return annotations, nil
}

// --- ModelPack Provider ---

// ErrNotImplemented is returned by provider operations that have no real
// implementation yet, so callers fail loudly instead of assuming success.
var ErrNotImplemented = errors.New("not implemented")

// ModelPackProvider is a placeholder for the ModelPack CLI integration.
// Tool detection works; every registry operation returns ErrNotImplemented
// until the integration is written. Use the ORAS provider in the meantime.
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

func (m *ModelPackProvider) notImplemented(op string) error {
	return fmt.Errorf("modelpack %s: %w (use --registry oras)", op, ErrNotImplemented)
}

func (m *ModelPackProvider) Push(artifact, registry, sourcePath string, annotations map[string]string) (string, error) {
	return "", m.notImplemented("push")
}

func (m *ModelPackProvider) Pull(artifact, registry string) error {
	return m.notImplemented("pull")
}

func (m *ModelPackProvider) GetArtifactDigest(artifact, registry string) (string, error) {
	return "", m.notImplemented("digest lookup")
}

func (m *ModelPackProvider) PushReferrer(artifact, registry, referrerType string, data []byte, annotations map[string]string) error {
	return m.notImplemented("referrer push")
}

func (m *ModelPackProvider) GetReferrers(artifact, registry, referrerType string) ([][]byte, error) {
	return nil, m.notImplemented("referrer fetch")
}

func (m *ModelPackProvider) Search(registry string, query *SearchQuery) ([]ManifestCandidate, error) {
	return nil, m.notImplemented("search")
}

func (m *ModelPackProvider) FetchManifestAnnotations(artifactRef string) (map[string]string, error) {
	return nil, m.notImplemented("manifest fetch")
}

// GetRegistryProvider returns the registry provider for an option name from
// RegistryOptions. Harbor and other OCI registries are used through "oras".
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
