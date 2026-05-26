package inspect

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/BrendenWalker/verity/internal/apiclient"
	"github.com/BrendenWalker/verity/internal/publish"
	"github.com/BrendenWalker/verity/internal/registry"
)

// Options configures an inspect run (FR-SIGN-005, FR-DX-002).
type Options struct {
	Namespace string
	Artifact  string
	Ref       string
}

// Result is the trust checklist outcome for printing and exit codes.
type Result struct {
	Trust         *apiclient.TrustStatus
	SignatureLine string
	SignatureOK   bool
}

// Run resolves ref, fetches API trust status (server-side signature verify), and formats output.
func Run(ctx context.Context, api *apiclient.Client, opts Options) (*Result, error) {
	opts.Namespace = strings.TrimSpace(opts.Namespace)
	opts.Artifact = strings.TrimSpace(opts.Artifact)
	opts.Ref = strings.TrimSpace(opts.Ref)
	if opts.Namespace == "" || opts.Artifact == "" {
		return nil, fmt.Errorf("namespace and artifact are required")
	}
	if opts.Ref == "" {
		return nil, fmt.Errorf("ref is required")
	}

	digest, tag, err := resolveRef(opts.Ref)
	if err != nil {
		return nil, err
	}

	trust, err := api.GetTrustStatus(ctx, opts.Namespace, opts.Artifact, digest, tag)
	if err != nil {
		return nil, err
	}

	line, ok := signatureLine(trust.Signatures.Status)
	return &Result{
		Trust:         trust,
		SignatureLine: line,
		SignatureOK:   ok,
	}, nil
}

func resolveRef(ref string) (digest, tag string, err error) {
	if strings.HasPrefix(ref, "sha256:") {
		return ref, "", nil
	}
	if _, err := os.Stat(ref); err == nil {
		layers, publishRoot, err := publish.CollectFiles(ref)
		if err != nil {
			return "", "", err
		}
		_, h, err := registry.BuildArtifactManifest(layers, registry.ManifestOptions{PublishRoot: publishRoot})
		if err != nil {
			return "", "", err
		}
		return h.String(), "", nil
	}
	return "", ref, nil
}

func signatureLine(status string) (line string, ok bool) {
	switch status {
	case "valid":
		return "✓ Signed by GitHub Actions", true
	case "missing":
		return "✗ Signature missing", false
	case "invalid":
		return "✗ Signature invalid", false
	default:
		return fmt.Sprintf("✗ Signature status unknown (%s)", status), false
	}
}
