package workflow

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"os"
	"os/exec"
	"path"
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
	// PackagingFormat returns the value the provider stamps into the
	// org.cncf.ai.packaging.format annotation: the format the artifact is
	// actually packaged in, not the tool name (see PackagingFormatOCI).
	PackagingFormat() string
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

func (o *ORASProvider) PackagingFormat() string {
	return PackagingFormatOCI
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

// annotateManifest merges annotations into the manifest stored at ref and
// pushes the result back under the same reference: `oras manifest fetch`,
// edit .annotations, `oras manifest push`. Every other manifest field is
// carried over untouched. It exists for providers whose own CLI cannot set
// manifest annotations (modctl).
func (o *ORASProvider) annotateManifest(ref string, annotations map[string]string) error {
	fetch := exec.Command("oras", "manifest", "fetch", ref)
	fetch.Stderr = os.Stderr
	raw, err := fetch.Output()
	if err != nil {
		return fmt.Errorf("failed to fetch manifest %s with ORAS: %v", ref, err)
	}
	merged, err := mergeManifestAnnotations(raw, annotations)
	if err != nil {
		return fmt.Errorf("failed to annotate manifest %s: %v", ref, err)
	}

	tmpDir, err := os.MkdirTemp("", "manifest-*")
	if err != nil {
		return fmt.Errorf("failed to create temp dir for manifest: %v", err)
	}
	defer os.RemoveAll(tmpDir)
	manifestPath := filepath.Join(tmpDir, "manifest.json")
	if err := os.WriteFile(manifestPath, merged, 0644); err != nil {
		return fmt.Errorf("failed to write annotated manifest: %v", err)
	}

	// ORAS takes the media type from the manifest's own mediaType field.
	push := exec.Command("oras", "manifest", "push", ref, manifestPath)
	push.Stderr = os.Stderr
	if err := push.Run(); err != nil {
		return fmt.Errorf("failed to push annotated manifest %s with ORAS: %v", ref, err)
	}
	return nil
}

// mergeManifestAnnotations adds annotations to the "annotations" object of
// the OCI manifest JSON raw, overwriting keys that already exist and keeping
// every other field as is.
func mergeManifestAnnotations(raw []byte, annotations map[string]string) ([]byte, error) {
	var manifest map[string]json.RawMessage
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return nil, fmt.Errorf("invalid manifest JSON: %v", err)
	}
	merged := map[string]string{}
	if existing, ok := manifest["annotations"]; ok {
		if err := json.Unmarshal(existing, &merged); err != nil {
			return nil, fmt.Errorf("invalid manifest annotations: %v", err)
		}
		if merged == nil { // "annotations": null
			merged = map[string]string{}
		}
	}
	for k, v := range annotations {
		merged[k] = v
	}
	encoded, err := json.Marshal(merged)
	if err != nil {
		return nil, err
	}
	manifest["annotations"] = encoded
	return json.Marshal(manifest)
}

// --- ModelPack Provider ---

// ModelPackProvider implements RegistryProvider with modctl, the CNCF
// ModelPack CLI (https://github.com/modelpack/modctl). modctl packages a
// model directory as a Model Spec artifact (`modctl build`) and uploads it
// (`modctl push`), but it has no flag for manifest annotations and no
// referrer commands. Those parts go through ORAS against the same registry,
// which speaks plain OCI: Push merges the CNCF annotations into the pushed
// manifest with ORAS, and the read-side operations delegate to ORASProvider.
// Both modctl and oras therefore need to be installed.
type ModelPackProvider struct{}

func (m *ModelPackProvider) Name() string {
	return "modelpack"
}

func (m *ModelPackProvider) PackagingFormat() string {
	return PackagingFormatModelPack
}

func (m *ModelPackProvider) IsInstalled() bool {
	_, err := exec.LookPath("modctl")
	return err == nil
}

func (m *ModelPackProvider) InstallInstructions() string {
	return "go install github.com/modelpack/modctl@latest"
}

// modelfileName is the file modctl reads the artifact layout from.
const modelfileName = "Modelfile"

// Push packages sourcePath with `modctl build -t <ref> -f <Modelfile> <dir>`
// and uploads it with `modctl push <ref>`. A Modelfile in the source
// directory is used as is; otherwise one is generated in a temp dir that
// lists every file below sourcePath (modctl builds one layer per file and
// has no directory layers). modctl cannot set manifest annotations, so the
// CNCF annotations are merged into the pushed manifest with ORAS afterwards;
// the returned digest is that of the final manifest.
func (m *ModelPackProvider) Push(artifact, registry, sourcePath string, annotations map[string]string) (string, error) {
	absSource, err := filepath.Abs(sourcePath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve source path %q: %v", sourcePath, err)
	}
	info, err := os.Stat(absSource)
	if err != nil {
		return "", fmt.Errorf("source path %q is not accessible: %v", sourcePath, err)
	}
	oras := &ORASProvider{}
	if _, err := exec.LookPath("oras"); err != nil {
		return "", fmt.Errorf("the modelpack provider needs oras to annotate the pushed manifest; install it with: %s", oras.InstallInstructions())
	}

	// modctl builds from a directory context; a single file is built from its parent.
	var contextDir, modelfilePath string
	var files []string
	if info.IsDir() {
		contextDir = absSource
		if existing := filepath.Join(absSource, modelfileName); fileExists(existing) {
			modelfilePath = existing
		} else if files, err = modelFiles(absSource); err != nil {
			return "", err
		}
	} else {
		contextDir = filepath.Dir(absSource)
		files = []string{filepath.Base(absSource)}
	}
	if modelfilePath == "" {
		if len(files) == 0 {
			return "", fmt.Errorf("source path %q has no files to package", sourcePath)
		}
		tmpDir, err := os.MkdirTemp("", "modelfile-*")
		if err != nil {
			return "", fmt.Errorf("failed to create temp dir for Modelfile: %v", err)
		}
		defer os.RemoveAll(tmpDir)
		modelfilePath = filepath.Join(tmpDir, modelfileName)
		if err := os.WriteFile(modelfilePath, modelfileFor(modelNameOf(artifact), files), 0644); err != nil {
			return "", fmt.Errorf("failed to write Modelfile: %v", err)
		}
	}

	ref := registry + "/" + artifact
	plainHTTP := plainHTTPArgs(registry)
	buildArgs := append([]string{"build", "-t", ref, "-f", modelfilePath}, plainHTTP...)
	buildArgs = append(buildArgs, contextDir)
	if err := runModctl(buildArgs...); err != nil {
		return "", fmt.Errorf("failed to build with modctl: %v", err)
	}
	pushArgs := append([]string{"push"}, plainHTTP...)
	pushArgs = append(pushArgs, ref)
	if err := runModctl(pushArgs...); err != nil {
		return "", fmt.Errorf("failed to push with modctl: %v", err)
	}
	fmt.Printf("Pushed artifact %s to %s using ModelPack\n", artifact, registry)

	if len(annotations) > 0 {
		if err := oras.annotateManifest(ref, annotations); err != nil {
			return "", err
		}
		fmt.Printf("Attached %d CNCF AI annotation(s) to the manifest\n", len(annotations))
	}
	return oras.GetArtifactDigest(artifact, registry)
}

// runModctl runs modctl with args, streaming its progress output.
func runModctl(args ...string) error {
	cmd := exec.Command("modctl", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// plainHTTPArgs returns modctl's --plain-http flag for registries on the
// local host, where no TLS is expected. ORAS applies that default on its
// own; modctl has to be told.
func plainHTTPArgs(registry string) []string {
	host := registry
	if i := strings.Index(host, "/"); i >= 0 {
		host = host[:i]
	}
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	switch host {
	case "localhost", "127.0.0.1", "::1":
		return []string{"--plain-http"}
	}
	return nil
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// modelNameOf returns the last path element of an artifact's repository,
// e.g. "my-model" for "my-org/my-model:v1", for the Modelfile NAME.
func modelNameOf(artifact string) string {
	return path.Base(repositoryOf(artifact))
}

// modelFiles lists the files below dir relative to it, in lexical order.
// Hidden entries (".git", ".DS_Store", ...) are skipped, as `modctl
// modelfile generate` does.
func modelFiles(dir string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if p != dir && strings.HasPrefix(d.Name(), ".") {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.IsDir() {
			rel, err := filepath.Rel(dir, p)
			if err != nil {
				return err
			}
			files = append(files, filepath.ToSlash(rel))
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list files in %s: %v", dir, err)
	}
	return files, nil
}

// modelfileFor writes a Modelfile (see modctl's docs/getting-started.md)
// that names the model and lists each file under the Model Spec directive
// for its role: CONFIG for JSON/YAML, DOC for Markdown and text, CODE for
// Python, MODEL for everything else. Paths are relative to the build
// context; ones with whitespace are quoted for the Modelfile parser.
func modelfileFor(name string, files []string) []byte {
	var b strings.Builder
	b.WriteString("# Generated by model-cli for modctl build\n")
	if name != "" && name != "." {
		fmt.Fprintf(&b, "NAME %s\n", name)
	}
	for _, f := range files {
		arg := f
		if strings.ContainsAny(arg, " \t") {
			arg = `"` + arg + `"`
		}
		fmt.Fprintf(&b, "%s %s\n", modelfileDirective(f), arg)
	}
	return []byte(b.String())
}

func modelfileDirective(file string) string {
	switch strings.ToLower(filepath.Ext(file)) {
	case ".json", ".yaml", ".yml":
		return "CONFIG"
	case ".md", ".txt":
		return "DOC"
	case ".py":
		return "CODE"
	}
	return "MODEL"
}

// Pull downloads the artifact into modctl's local store with `modctl pull`.
func (m *ModelPackProvider) Pull(artifact, registry string) error {
	args := append([]string{"pull"}, plainHTTPArgs(registry)...)
	args = append(args, registry+"/"+artifact)
	if err := runModctl(args...); err != nil {
		return fmt.Errorf("failed to pull with modctl: %v", err)
	}
	fmt.Printf("Pulled artifact %s from %s using ModelPack\n", artifact, registry)
	return nil
}

// The remaining operations read or attach plain OCI manifests, which modctl
// has no commands for (`modctl inspect` shows the Model Spec config, not the
// manifest digest or annotations, and it has no referrer support). They
// delegate to ORAS against the same registry.

func (m *ModelPackProvider) GetArtifactDigest(artifact, registry string) (string, error) {
	return (&ORASProvider{}).GetArtifactDigest(artifact, registry)
}

func (m *ModelPackProvider) PushReferrer(artifact, registry, referrerType string, data []byte, annotations map[string]string) error {
	return (&ORASProvider{}).PushReferrer(artifact, registry, referrerType, data, annotations)
}

func (m *ModelPackProvider) GetReferrers(artifact, registry, referrerType string) ([][]byte, error) {
	return (&ORASProvider{}).GetReferrers(artifact, registry, referrerType)
}

func (m *ModelPackProvider) Search(registry string, query *SearchQuery) ([]ManifestCandidate, error) {
	return (&ORASProvider{}).Search(registry, query)
}

func (m *ModelPackProvider) FetchManifestAnnotations(artifactRef string) (map[string]string, error) {
	return (&ORASProvider{}).FetchManifestAnnotations(artifactRef)
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
