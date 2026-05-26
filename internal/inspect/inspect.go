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

// ChecklistLine is one human-readable inspect row (AC-DX-002).
type ChecklistLine struct {
	Text string
	Must bool
	Pass bool
}

// Result is the trust checklist outcome for printing and exit codes.
type Result struct {
	Trust     *apiclient.TrustStatus
	MustLines []ChecklistLine
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

	return &Result{
		Trust:     trust,
		MustLines: MustChecklist(trust),
	}, nil
}

// MustFailed reports whether any Must checklist line failed (FR-DX-005).
func MustFailed(lines []ChecklistLine) bool {
	for _, l := range lines {
		if l.Must && !l.Pass {
			return true
		}
	}
	return false
}

// MustChecklist builds MVP Must output lines from API trust status (AC-DX-002).
func MustChecklist(trust *apiclient.TrustStatus) []ChecklistLine {
	text, pass := signatureLine(trust.Signatures.Status)
	return []ChecklistLine{{Text: text, Must: true, Pass: pass}}
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
