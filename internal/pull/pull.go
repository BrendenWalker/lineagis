package pull

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BrendenWalker/lineagis/internal/apiclient"
	"github.com/BrendenWalker/lineagis/internal/inspect"
	"github.com/BrendenWalker/lineagis/internal/publish"
	"github.com/BrendenWalker/lineagis/internal/registry"
)

// Options configures a pull run (FR-DX-012).
type Options struct {
	Ref         string
	OutputDir   string
	Verify      bool
	APIURL      string
	RegistryURL string
	Token       string
}

// Pull resolves the reference, optionally verifies trust, and writes layer files to OutputDir.
func Pull(ctx context.Context, opts Options) (digest string, err error) {
	parsed, err := ParseRef(opts.Ref)
	if err != nil {
		return "", err
	}
	api := apiclient.New(opts.APIURL, opts.Token)
	reg, err := registry.New(opts.RegistryURL)
	if err != nil {
		return "", err
	}

	digest = parsed.Digest
	if digest == "" {
		tagRes, err := api.GetTag(ctx, parsed.Namespace, parsed.Artifact, parsed.Tag)
		if err != nil {
			return "", fmt.Errorf("resolve tag: %w", err)
		}
		digest = tagRes.Digest
	}

	if opts.Verify {
		result, err := inspect.Run(ctx, api, inspect.Options{
			Namespace:   parsed.Namespace,
			Artifact:    parsed.Artifact,
			Ref:         digest,
			LocalVerify: true,
			RegistryURL: opts.RegistryURL,
		})
		if err != nil {
			return "", fmt.Errorf("verify: %w", err)
		}
		if inspect.MustFailed(result.MustLines) {
			return "", fmt.Errorf("trust verification failed")
		}
	}

	repo := publish.RegistryRepo(parsed.Namespace, parsed.Artifact)
	manifestJSON, _, err := reg.PullManifest(ctx, repo, digest)
	if err != nil {
		return "", fmt.Errorf("pull manifest: %w", err)
	}
	layers, err := registry.LayersFromManifest(manifestJSON)
	if err != nil {
		return "", err
	}

	outDir := opts.OutputDir
	if outDir == "" {
		outDir = "."
	}
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return "", err
	}

	for _, layer := range layers {
		data, err := reg.PullBlob(ctx, repo, layer.Digest)
		if err != nil {
			return "", fmt.Errorf("pull layer %s: %w", layer.Path, err)
		}
		dest := filepath.Join(outDir, filepath.FromSlash(layer.Path))
		if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
			return "", err
		}
		if err := os.WriteFile(dest, data, 0644); err != nil {
			return "", fmt.Errorf("write %s: %w", layer.Path, err)
		}
	}
	return digest, nil
}
