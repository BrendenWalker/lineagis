package resolver

import (
	"strings"
)

// NormalizeDigest returns lowercase sha256:hex form without artifact: prefix.
func NormalizeDigest(digest string) string {
	d := strings.TrimSpace(digest)
	d = strings.TrimPrefix(strings.ToLower(d), "artifact:")
	d = strings.TrimPrefix(d, "sha256:")
	return "sha256:" + d
}

// HexFromDigest strips the sha256: prefix.
func HexFromDigest(digest string) string {
	d := NormalizeDigest(digest)
	return strings.TrimPrefix(d, "sha256:")
}
