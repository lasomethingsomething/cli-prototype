package workflow

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
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
	// Search queries the registry for artifacts matching the given filters
	// Returns manifest bytes for matching artifacts, which can be parsed for metadata
	Search(registry, filters string) ([][]byte, error)
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

func (o *ORASProvider) Search(registry, filters string) ([][]byte, error) {
	// For zot registries (which have the search extension enabled),
	// try using the zot search API first. If that fails, fall back to
	// the OCI Distribution Spec approach which works with any registry.
	
	// Check if this might be a zot registry by trying the search extension
	if manifests, err := tryZotSearch(registry, filters); err == nil {
		return manifests, nil
	}
	
	// Fall back to OCI Distribution Spec approach: catalog -> tags/list -> manifest fetch
	// This works with any OCI-compliant registry
	return searchOCIDistribution(registry, filters)
}

// tryZotSearch attempts to use zot's search extension API
func tryZotSearch(registry, filters string) ([][]byte, error) {
	// Try the zot search extension endpoint
	// Zot's search extension uses HTTP GET with query parameters
	// Format: /v2/_zot/ext/search?n=<name>&a=<artifact-type>&l=<label>=<value>
	
	url := fmt.Sprintf("http://%s/v2/_zot/ext/search", registry)
	
	// Build query parameters from filters
	// Filters are in format "key=value&key2=value2"
	params := make(map[string]string)
	
	// Parse the filter string
	if filters != "" {
		for _, pair := range strings.Split(filters, "&") {
			if pair != "" {
				kv := strings.SplitN(pair, "=", 2)
				if len(kv) == 2 {
					key, value := kv[0], kv[1]
					// Map filter keys to zot search parameters
					// Zot uses: n=name, t=tag, a=artifact-type, l=label/annotation
					if key == AnnotationArtifactType {
						// This is the artifact type
						params["a"] = "application/vnd.cncf.ai." + value
					} else if strings.HasPrefix(key, "ai.") {
						// AI annotations - zot uses l=label format
						params["l"] = key + "=" + value
					} else {
						// Other annotations
						params["l"] = key + "=" + value
					}
				}
			}
		}
	}
	
	// If no artifact type filter, search for all AI artifact types
	if _, ok := params["a"]; !ok {
		params["a"] = "application/vnd.cncf.ai.%"
	}
	
	// Build the URL with query parameters
	queryString := ""
	for k, v := range params {
		if queryString == "" {
			queryString = "?" + k + "=" + v
		} else {
			queryString += "&" + k + "=" + v
		}
	}
	
	fullURL := url + queryString
	
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(fullURL)
	if err != nil {
		return nil, fmt.Errorf("failed to query zot search API: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("zot search API returned status %d", resp.StatusCode)
	}

	// Parse the response - zot returns a list of repositories with artifacts
	var result zotSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse zot search response: %v", err)
	}

	// Fetch manifests for all found artifacts
	var manifests [][]byte
	for _, repo := range result.Repositories {
		for _, artifact := range repo.Artifacts {
			manifest, err := fetchManifest(registry, repo.Name, artifact.Digest)
			if err != nil {
				continue
			}
			manifests = append(manifests, manifest)
		}
	}

	return manifests, nil
}

// zotSearchResponse represents the response from zot's search extension API
type zotSearchResponse struct {
	Repositories []struct {
		Name      string `json:"name"`
		Artifacts []struct {
			Digest    string `json:"digest"`
			MediaType string `json:"mediaType"`
			Size      int64  `json:"size"`
		} `json:"artifacts"`
	} `json:"repositories"`
}

// searchOCIDistribution performs a search using OCI Distribution Spec
// This works with any registry that exposes the catalog endpoint
func searchOCIDistribution(registry, filters string) ([][]byte, error) {
	// Step 1: Fetch the catalog from /v2/_catalog
	catalog, err := fetchCatalog(registry)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch catalog: %v", err)
	}

	// Step 2: For each repository, fetch tags/list
	var manifests [][]byte
	for _, repo := range catalog.Repositories {
		tags, err := fetchTags(registry, repo)
		if err != nil {
			continue // Skip repos we can't fetch tags for
		}

		// Step 3: For each tag, fetch the manifest
		for _, tag := range tags.Tags {
			manifest, err := fetchManifest(registry, repo, tag.Digest)
			if err != nil {
				continue // Skip manifests we can't fetch
			}

			// Filter by artifact type if specified
			if filters != "" && !matchesFilter(manifest, filters) {
				continue
			}

			manifests = append(manifests, manifest)
		}
	}

	return manifests, nil
}

// fetchCatalog fetches the repository catalog from /v2/_catalog
type catalogResponse struct {
	Repositories []string `json:"repositories"`
}

func fetchCatalog(registry string) (*catalogResponse, error) {
	url := fmt.Sprintf("http://%s/v2/_catalog", registry)
	client := &http.Client{Timeout: 30 * time.Second}

	// First, try with no auth (for zot and local registries)
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch catalog: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		// Try with empty auth (some registries require this)
		req, _ := http.NewRequest("GET", url, nil)
		req.SetBasicAuth("", "")
		resp, err = client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch catalog with auth: %v", err)
		}
		defer resp.Body.Close()
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("catalog request returned status %d", resp.StatusCode)
	}

	var catalog catalogResponse
	if err := json.NewDecoder(resp.Body).Decode(&catalog); err != nil {
		return nil, fmt.Errorf("failed to parse catalog response: %v", err)
	}

	return &catalog, nil
}

// fetchTags fetches the tags list from /v2/<repo>/tags/list
type tagsListResponse struct {
	Name string   `json:"name"`
	Tags []tagInfo `json:"tags"`
}

type tagInfo struct {
	Name   string `json:"name"`
	Digest string `json:"digest"`
	Size   int64  `json:"size"`
}

func fetchTags(registry, repo string) (*tagsListResponse, error) {
	url := fmt.Sprintf("http://%s/v2/%s/tags/list", registry, repo)
	client := &http.Client{Timeout: 30 * time.Second}

	// Try with no auth first
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch tags for %s: %v", repo, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		// Try with empty auth
		req, _ := http.NewRequest("GET", url, nil)
		req.SetBasicAuth("", "")
		resp, err = client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch tags for %s with auth: %v", repo, err)
		}
		defer resp.Body.Close()
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tags list request for %s returned status %d", repo, resp.StatusCode)
	}

	var tags tagsListResponse
	if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
		return nil, fmt.Errorf("failed to parse tags list for %s: %v", repo, err)
	}

	return &tags, nil
}

// fetchManifest fetches a manifest from /v2/<repo>/manifests/<digest>
func fetchManifest(registry, repo, digest string) ([]byte, error) {
	url := fmt.Sprintf("http://%s/v2/%s/manifests/%s", registry, repo, digest)
	client := &http.Client{Timeout: 30 * time.Second}

	// Try with no auth first
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch manifest %s: %v", digest, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		// Try with empty auth
		req, _ := http.NewRequest("GET", url, nil)
		req.SetBasicAuth("", "")
		resp, err = client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch manifest %s with auth: %v", digest, err)
		}
		defer resp.Body.Close()
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("manifest request for %s returned status %d", digest, resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

// matchesFilter checks if a manifest matches the given filter
func matchesFilter(manifest []byte, filters string) bool {
	// Parse the manifest to check annotations
	var manifestData map[string]interface{}
	if err := json.Unmarshal(manifest, &manifestData); err != nil {
		return false
	}

	// Check for AI artifact type annotation
	if at, ok := manifestData["annotations"].(map[string]interface{})[AnnotationArtifactType]; ok {
		if atStr, ok := at.(string); ok {
			if strings.Contains(filters, atStr) {
				return true
			}
		}
	}

	// If no specific filter match, include it
	return true
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

func (m *ModelPackProvider) Search(registry, filters string) ([][]byte, error) {
	return nil, m.notImplemented("search")
}

func (m *ModelPackProvider) FetchManifestAnnotations(artifactRef string) (map[string]string, error) {
	return nil, m.notImplemented("manifest fetch")
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
