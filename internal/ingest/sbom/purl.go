package sbom

import "strings"

// parsePURL extracts ecosystem, name, and version from a Package URL (or falls back to name/version).
func parsePURL(purl, name, version string) (eco, n, ver string) {
	ver = version
	n = name
	eco = "generic"
	if purl == "" {
		if ver == "" {
			ver = "unknown"
		}
		return eco, n, ver
	}
	// pkg:npm/lodash@4.17.21
	p := strings.TrimPrefix(purl, "pkg:")
	parts := strings.SplitN(p, "/", 3)
	if len(parts) >= 1 {
		eco = parts[0]
	}
	if len(parts) >= 2 {
		n = parts[1]
		if idx := strings.LastIndex(n, "@"); idx > 0 {
			ver = n[idx+1:]
			n = n[:idx]
		}
	}
	if len(parts) >= 3 && ver == "" {
		rest := parts[2]
		if idx := strings.LastIndex(rest, "@"); idx >= 0 {
			ver = rest[idx+1:]
		}
	}
	if ver == "" {
		ver = "unknown"
	}
	return eco, n, ver
}
