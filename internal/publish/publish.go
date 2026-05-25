package publish

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/go-containerregistry/pkg/v1"

	"github.com/BrendenWalker/verity/internal/apiclient"
	"github.com/BrendenWalker/verity/internal/registry"
	"github.com/BrendenWalker/verity/internal/signing"
)

// ManifestSigner signs manifest bytes and returns a Sigstore bundle (FR-SIGN-001–003).
type ManifestSigner interface {
	SignManifest(ctx context.Context, manifestJSON []byte) (bundle json.RawMessage, issuer, subject *string, err error)
}

// Options configures a publish run.
type Options struct {
	Namespace string
	Artifact  string
	Tag       string
	Path      string
	SkipSign  bool
	Signer    ManifestSigner
}

// RegistryRepo returns the OCI repository name {namespace}/{artifact}.
func RegistryRepo(namespace, artifact string) string {
	namespace = strings.Trim(namespace, "/")
	artifact = strings.Trim(artifact, "/")
	return namespace + "/" + artifact
}

// Publish uploads files to the registry and registers metadata via the API.
func Publish(ctx context.Context, reg *registry.Client, api *apiclient.Client, opts Options) (string, error) {
	opts.Namespace = strings.TrimSpace(opts.Namespace)
	opts.Artifact = strings.TrimSpace(opts.Artifact)
	opts.Tag = strings.TrimSpace(opts.Tag)
	if opts.Namespace == "" || opts.Artifact == "" {
		return "", fmt.Errorf("namespace and artifact are required")
	}

	layers, publishRoot, err := CollectFiles(opts.Path)
	if err != nil {
		return "", err
	}

	manifestJSON, manifestHash, err := registry.BuildArtifactManifest(layers, registry.ManifestOptions{
		PublishRoot: publishRoot,
	})
	if err != nil {
		return "", err
	}

	repo := RegistryRepo(opts.Namespace, opts.Artifact)
	for _, layer := range layers {
		if _, err := reg.PushBlob(ctx, repo, layer.Data); err != nil {
			return "", fmt.Errorf("push layer %s: %w", layer.Path, err)
		}
	}

	h, err := reg.PushManifest(ctx, repo, manifestJSON)
	if err != nil {
		return "", fmt.Errorf("push manifest: %w", err)
	}
	if h != manifestHash {
		return "", fmt.Errorf("manifest digest mismatch: got %s, want %s", h, manifestHash)
	}

	digest := h.String()
	mediaType := registry.ArtifactManifestMediaType
	size := int64(len(manifestJSON))

	if err := api.EnsureArtifact(ctx, opts.Namespace, opts.Artifact); err != nil {
		return "", fmt.Errorf("ensure artifact: %w", err)
	}
	if err := api.RegisterDigest(ctx, opts.Namespace, opts.Artifact, digest, &mediaType, &size); err != nil {
		return "", fmt.Errorf("register digest: %w", err)
	}

	if !opts.SkipSign {
		signer := opts.Signer
		if signer == nil {
			signer = defaultManifestSigner{}
		}
		bundle, issuer, subject, err := signer.SignManifest(ctx, manifestJSON)
		if err != nil {
			return "", fmt.Errorf("sign manifest: %w", err)
		}
		if err := api.AttachSignature(ctx, opts.Namespace, opts.Artifact, digest, bundle, issuer, subject); err != nil {
			return "", fmt.Errorf("attach signature: %w", err)
		}
	}

	if opts.Tag != "" {
		if err := api.SetTag(ctx, opts.Namespace, opts.Artifact, opts.Tag, digest); err != nil {
			return "", fmt.Errorf("set tag: %w", err)
		}
	}

	return digest, nil
}

type defaultManifestSigner struct{}

func (defaultManifestSigner) SignManifest(ctx context.Context, manifestJSON []byte) (json.RawMessage, *string, *string, error) {
	cfg := signing.LoadConfig()
	bundle, id, err := signing.SignManifest(ctx, cfg, manifestJSON)
	if err != nil {
		return nil, nil, nil, err
	}
	var issuer, subject *string
	if id.Issuer != "" {
		issuer = &id.Issuer
	}
	if id.Subject != "" {
		subject = &id.Subject
	}
	return bundle, issuer, subject, nil
}

// ManifestDigest computes manifest digest without pushing (for tests).
func ManifestDigest(layers []registry.FileLayer, publishRoot string) (v1.Hash, error) {
	_, h, err := registry.BuildArtifactManifest(layers, registry.ManifestOptions{PublishRoot: publishRoot})
	return h, err
}
