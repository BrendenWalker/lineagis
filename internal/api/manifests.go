package api

import (
	"context"
	"fmt"
	"strings"

	"github.com/BrendenWalker/verity/internal/registry"
)

// ManifestSource loads canonical manifest bytes for signature verification (FR-SIGN-006).
type ManifestSource interface {
	FetchManifest(ctx context.Context, namespace, artifact, digest string) ([]byte, error)
}

// RegistryManifests pulls release manifests from the OCI registry used at publish time.
type RegistryManifests struct {
	Client *registry.Client
}

// FetchManifest returns manifest JSON for the given digest reference.
func (r *RegistryManifests) FetchManifest(ctx context.Context, namespace, artifact, digest string) ([]byte, error) {
	if r == nil || r.Client == nil {
		return nil, fmt.Errorf("registry manifest source is not configured")
	}
	repo := registryRepo(namespace, artifact)
	data, pulled, err := r.Client.PullManifest(ctx, repo, digest)
	if err != nil {
		return nil, err
	}
	if pulled.String() != digest {
		return nil, fmt.Errorf("manifest digest mismatch: got %s, want %s", pulled, digest)
	}
	return data, nil
}

func registryRepo(namespace, artifact string) string {
	namespace = strings.Trim(namespace, "/")
	artifact = strings.Trim(artifact, "/")
	return namespace + "/" + artifact
}

// NewStaticManifestSource returns manifest bytes by digest for tests.
func NewStaticManifestSource(byDigest map[string][]byte) ManifestSource {
	return &staticManifestSource{manifests: byDigest}
}

// staticManifestSource is used in tests to avoid registry I/O.
type staticManifestSource struct {
	manifests map[string][]byte
}

func (s *staticManifestSource) FetchManifest(_ context.Context, _, _ string, digest string) ([]byte, error) {
	if s == nil {
		return nil, fmt.Errorf("manifest source is not configured")
	}
	data, ok := s.manifests[digest]
	if !ok {
		return nil, fmt.Errorf("manifest not found for %s", digest)
	}
	return data, nil
}
