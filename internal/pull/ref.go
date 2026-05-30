package pull

import (
	"fmt"
	"strings"
)

// Ref is a parsed pull reference (namespace, artifact, tag or digest).
type Ref struct {
	Namespace string
	Artifact  string
	Tag       string
	Digest    string
}

// ParseRef parses `namespace/.../artifact@sha256:…` or `…/artifact:tag`.
func ParseRef(raw string) (Ref, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return Ref{}, fmt.Errorf("pull reference is required")
	}
	if i := strings.LastIndex(raw, "@"); i >= 0 {
		left, right := raw[:i], raw[i+1:]
		if !strings.HasPrefix(right, "sha256:") {
			return Ref{}, fmt.Errorf("digest reference must use @sha256:… form")
		}
		ns, art, err := splitNamespaceArtifact(left)
		if err != nil {
			return Ref{}, err
		}
		return Ref{Namespace: ns, Artifact: art, Digest: right}, nil
	}
	if i := strings.LastIndex(raw, ":"); i >= 0 {
		left, right := raw[:i], raw[i+1:]
		if strings.HasPrefix(right, "sha256:") {
			return Ref{}, fmt.Errorf("use @sha256 for digest references instead of a colon separator")
		}
		ns, art, err := splitNamespaceArtifact(left)
		if err != nil {
			return Ref{}, err
		}
		return Ref{Namespace: ns, Artifact: art, Tag: right}, nil
	}
	return Ref{}, fmt.Errorf("reference must include @sha256:… or :tag")
}

func splitNamespaceArtifact(left string) (namespace, artifact string, err error) {
	left = strings.Trim(left, "/")
	if left == "" {
		return "", "", fmt.Errorf("namespace and artifact are required in reference")
	}
	i := strings.LastIndex(left, "/")
	if i < 0 {
		return "", "", fmt.Errorf("reference must be namespace/.../artifact@digest or :tag")
	}
	namespace = left[:i]
	artifact = left[i+1:]
	if namespace == "" || artifact == "" {
		return "", "", fmt.Errorf("invalid namespace/artifact in reference")
	}
	return namespace, artifact, nil
}
