package registry

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/types"
)

const (
	// ArtifactManifestMediaType is the manifest Content-Type for registry upload.
	// Zot v2+ and OCI Image Spec v1.1 use an OCI image manifest with artifactType
	// and an empty config descriptor instead of application/vnd.oci.artifact.manifest.v1+json.
	ArtifactManifestMediaType = "application/vnd.oci.image.manifest.v1+json"

	// VerityReleaseArtifactType identifies generic/multi-file releases (ADR-0001 layout).
	VerityReleaseArtifactType = "application/vnd.verity.release.v1+json"

	emptyConfigMediaType = "application/vnd.oci.empty.v1+json"

	// MaxLayersPerManifest is the maximum number of layers per release (ADR-0001).
	MaxLayersPerManifest = 256

	// MaxTotalReleaseSize is the maximum sum of layer sizes per release (ADR-0001).
	MaxTotalReleaseSize = 2 << 30

	annotationVerityPath        = "dev.verity.path"
	annotationImageTitle        = "org.opencontainers.image.title"
	annotationImageCreated      = "org.opencontainers.image.created"
	annotationVerityPublishRoot = "dev.verity.publish.root"
	annotationVerityFileCount   = "dev.verity.file.count"
)

// FileLayer is one file represented as an OCI manifest layer (ADR-0001).
type FileLayer struct {
	Path    string
	Data    []byte
	Created *time.Time
}

// ManifestOptions holds optional manifest-level annotations.
type ManifestOptions struct {
	PublishRoot string
}

// emptyConfigJSON is the OCI empty JSON blob (RFC 8785 canonical {}).
var emptyConfigJSON = []byte("{}")

var emptyConfigDescriptor v1.Descriptor

func init() {
	h, _, err := v1.SHA256(bytes.NewReader(emptyConfigJSON))
	if err != nil {
		panic(fmt.Sprintf("registry: empty config digest: %v", err))
	}
	emptyConfigDescriptor = v1.Descriptor{
		MediaType: emptyConfigMediaType,
		Digest:    h,
		Size:      int64(len(emptyConfigJSON)),
	}
}

type releaseManifest struct {
	SchemaVersion int64             `json:"schemaVersion"`
	MediaType     string            `json:"mediaType"`
	ArtifactType  string            `json:"artifactType"`
	Config        v1.Descriptor     `json:"config"`
	Annotations   map[string]string `json:"annotations,omitempty"`
	Layers        []v1.Descriptor   `json:"layers"`
}

// BuildArtifactManifest constructs canonical OCI Artifact manifest JSON per ADR-0001.
// Layers are sorted by dev.verity.path before serialization so identical file sets
// produce identical manifest digests (FR-PUB-007, AC-PUB-002).
func BuildArtifactManifest(layers []FileLayer, opts ManifestOptions) ([]byte, v1.Hash, error) {
	if len(layers) == 0 {
		return nil, v1.Hash{}, fmt.Errorf("registry: at least one layer is required")
	}
	if len(layers) > MaxLayersPerManifest {
		return nil, v1.Hash{}, fmt.Errorf("%w (%d layers, limit %d)", ErrTooManyLayers, len(layers), MaxLayersPerManifest)
	}

	sorted := append([]FileLayer(nil), layers...)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Path < sorted[j].Path
	})

	manifestLayers := make([]v1.Descriptor, 0, len(sorted))
	var totalSize int64

	for _, layer := range sorted {
		if strings.TrimSpace(layer.Path) == "" {
			return nil, v1.Hash{}, fmt.Errorf("registry: layer path is required")
		}
		if strings.Contains(layer.Path, `\`) {
			return nil, v1.Hash{}, fmt.Errorf("registry: layer path %q must be POSIX-style", layer.Path)
		}
		if len(layer.Data) > MaxBlobSize {
			return nil, v1.Hash{}, fmt.Errorf("%w (%d bytes, limit %d)", ErrBlobTooLarge, len(layer.Data), MaxBlobSize)
		}

		totalSize += int64(len(layer.Data))
		if totalSize > MaxTotalReleaseSize {
			return nil, v1.Hash{}, fmt.Errorf("%w (%d bytes, limit %d)", ErrReleaseTooLarge, totalSize, MaxTotalReleaseSize)
		}

		digest, _, err := v1.SHA256(bytes.NewReader(layer.Data))
		if err != nil {
			return nil, v1.Hash{}, fmt.Errorf("registry: compute layer digest: %w", err)
		}

		annotations := map[string]string{
			annotationVerityPath: layer.Path,
			annotationImageTitle: path.Base(layer.Path),
		}
		if layer.Created != nil {
			annotations[annotationImageCreated] = layer.Created.UTC().Format(time.RFC3339)
		}

		manifestLayers = append(manifestLayers, v1.Descriptor{
			MediaType:   types.MediaType(layerMediaType(layer.Path)),
			Digest:      digest,
			Size:        int64(len(layer.Data)),
			Annotations: annotations,
		})
	}

	manifest := releaseManifest{
		SchemaVersion: 2,
		MediaType:     ArtifactManifestMediaType,
		ArtifactType:  VerityReleaseArtifactType,
		Config:        emptyConfigDescriptor,
		Layers:        manifestLayers,
	}

	if opts.PublishRoot != "" {
		manifest.Annotations = map[string]string{
			annotationVerityPublishRoot: opts.PublishRoot,
			annotationVerityFileCount:   fmt.Sprintf("%d", len(manifestLayers)),
		}
	} else {
		manifest.Annotations = map[string]string{
			annotationVerityFileCount: fmt.Sprintf("%d", len(manifestLayers)),
		}
	}

	data, err := json.Marshal(manifest)
	if err != nil {
		return nil, v1.Hash{}, fmt.Errorf("registry: marshal manifest: %w", err)
	}

	h, _, err := v1.SHA256(bytes.NewReader(data))
	if err != nil {
		return nil, v1.Hash{}, fmt.Errorf("registry: compute manifest digest: %w", err)
	}

	return data, h, nil
}

func layerMediaType(filePath string) string {
	lower := strings.ToLower(filePath)
	switch {
	case strings.HasSuffix(lower, ".tar.gz"), strings.HasSuffix(lower, ".tgz"):
		return "application/vnd.oci.image.layer.v1.tar+gzip"
	case strings.HasSuffix(lower, ".whl"):
		return "application/zip"
	case strings.HasSuffix(lower, ".tar"):
		return "application/vnd.oci.image.layer.v1.tar"
	case strings.HasSuffix(lower, ".zip"):
		return "application/zip"
	case strings.HasSuffix(lower, ".json"):
		return "application/json"
	default:
		return "application/octet-stream"
	}
}

// PushManifest uploads a release manifest. Identical bytes yield the same digest;
// an existing manifest is not re-uploaded.
func (c *Client) PushManifest(ctx context.Context, repo string, data []byte) (v1.Hash, error) {
	if _, err := c.PushBlob(ctx, repo, emptyConfigJSON); err != nil {
		return v1.Hash{}, fmt.Errorf("registry: upload empty config blob: %w", err)
	}

	h, _, err := v1.SHA256(bytes.NewReader(data))
	if err != nil {
		return v1.Hash{}, fmt.Errorf("registry: compute manifest digest: %w", err)
	}

	exists, err := c.ManifestExists(ctx, repo, h)
	if err != nil {
		return v1.Hash{}, err
	}
	if exists {
		return h, nil
	}

	manifestURL, err := c.manifestURL(repo, h.String())
	if err != nil {
		return v1.Hash{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, manifestURL, bytes.NewReader(data))
	if err != nil {
		return v1.Hash{}, fmt.Errorf("registry: create manifest upload request: %w", err)
	}
	req.Header.Set("Content-Type", ArtifactManifestMediaType)
	req.ContentLength = int64(len(data))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return v1.Hash{}, fmt.Errorf("registry: upload manifest: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated {
		return v1.Hash{}, fmt.Errorf("registry: upload manifest: %s", readStatus(resp))
	}

	if header := resp.Header.Get("Docker-Content-Digest"); header != "" {
		got, err := v1.NewHash(header)
		if err != nil {
			return v1.Hash{}, fmt.Errorf("registry: parse Docker-Content-Digest: %w", err)
		}
		if got != h {
			return v1.Hash{}, fmt.Errorf("registry: manifest digest mismatch: got %s, want %s", got, h)
		}
	}

	return h, nil
}

// ManifestLayer is one file layer described by a release manifest.
type ManifestLayer struct {
	Path   string
	Digest v1.Hash
}

// LayersFromManifest extracts layer paths and digests from release manifest JSON.
func LayersFromManifest(manifestJSON []byte) ([]ManifestLayer, error) {
	var m releaseManifest
	if err := json.Unmarshal(manifestJSON, &m); err != nil {
		return nil, fmt.Errorf("registry: parse manifest: %w", err)
	}
	out := make([]ManifestLayer, 0, len(m.Layers))
	for _, layer := range m.Layers {
		p := ""
		if layer.Annotations != nil {
			p = layer.Annotations[annotationVerityPath]
		}
		if strings.TrimSpace(p) == "" {
			return nil, fmt.Errorf("registry: layer missing %s annotation", annotationVerityPath)
		}
		out = append(out, ManifestLayer{Path: p, Digest: layer.Digest})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("registry: manifest has no layers")
	}
	return out, nil
}

// PullManifest downloads manifest bytes for the given digest or tag reference.
func (c *Client) PullManifest(ctx context.Context, repo, reference string) ([]byte, v1.Hash, error) {
	reference = strings.TrimSpace(reference)
	if reference == "" {
		return nil, v1.Hash{}, fmt.Errorf("registry: manifest reference is required")
	}

	manifestURL, err := c.manifestURL(repo, reference)
	if err != nil {
		return nil, v1.Hash{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, manifestURL, nil)
	if err != nil {
		return nil, v1.Hash{}, fmt.Errorf("registry: create manifest download request: %w", err)
	}
	req.Header.Set("Accept", ArtifactManifestMediaType)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, v1.Hash{}, fmt.Errorf("registry: download manifest: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return nil, v1.Hash{}, ErrNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return nil, v1.Hash{}, fmt.Errorf("registry: download manifest: %s", readStatus(resp))
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, MaxTotalReleaseSize+1))
	if err != nil {
		return nil, v1.Hash{}, fmt.Errorf("registry: read manifest: %w", err)
	}

	got, _, err := v1.SHA256(bytes.NewReader(data))
	if err != nil {
		return nil, v1.Hash{}, fmt.Errorf("registry: verify manifest digest: %w", err)
	}

	if strings.HasPrefix(reference, "sha256:") {
		want, err := v1.NewHash(reference)
		if err != nil {
			return nil, v1.Hash{}, fmt.Errorf("registry: parse manifest reference: %w", err)
		}
		if got != want {
			return nil, v1.Hash{}, fmt.Errorf("registry: manifest digest mismatch: got %s, want %s", got, want)
		}
	}

	return data, got, nil
}

// ManifestExists reports whether a manifest with the given digest is present.
func (c *Client) ManifestExists(ctx context.Context, repo string, h v1.Hash) (bool, error) {
	manifestURL, err := c.manifestURL(repo, h.String())
	if err != nil {
		return false, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodHead, manifestURL, nil)
	if err != nil {
		return false, fmt.Errorf("registry: create manifest head request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("registry: head manifest: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	switch resp.StatusCode {
	case http.StatusOK:
		return true, nil
	case http.StatusNotFound:
		return false, nil
	default:
		return false, fmt.Errorf("registry: head manifest: %s", readStatus(resp))
	}
}
