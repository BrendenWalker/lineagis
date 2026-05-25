package api

import (
	"net/http"
	"strings"
)

// RegisterRoutes mounts /v1 control-plane routes on mux.
// Namespace paths may contain slashes (e.g. gh/acme/widget); routing parses manually (OQ-API-001).
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	wrap := func(handler http.HandlerFunc) http.Handler {
		if h.Auth != nil {
			return h.Auth(handler)
		}
		return handler
	}
	mux.Handle("/v1/{path...}", wrap(h.routeV1))
}

func (h *Handler) routeV1(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.PathValue("path"), "/")
	if rest == "" {
		rest = r.PathValue("path")
	}
	ns, artifact, suffix, ok := parseNamespacesPath(rest)
	if !ok {
		WriteError(w, http.StatusNotFound, "NOT_FOUND", "unknown route", nil)
		return
	}

	switch {
	case r.Method == http.MethodPut && suffix == "":
		h.putArtifact(w, r, ns, artifact)
	case r.Method == http.MethodPost && suffix == "digests":
		h.postRegisterDigest(w, r, ns, artifact)
	case r.Method == http.MethodPut && strings.HasPrefix(suffix, "tags/"):
		tag := strings.TrimPrefix(suffix, "tags/")
		if tag == "" || strings.Contains(tag, "/") {
			WriteError(w, http.StatusNotFound, "NOT_FOUND", "unknown route", nil)
			return
		}
		h.putSetTag(w, r, ns, artifact, tag)
	default:
		WriteError(w, http.StatusNotFound, "NOT_FOUND", "unknown route", nil)
	}
}

// parseNamespacesPath parses paths like namespaces/gh/acme/widget/artifacts/widget/digests.
func parseNamespacesPath(path string) (namespace, artifact, suffix string, ok bool) {
	const prefix = "namespaces/"
	if !strings.HasPrefix(path, prefix) {
		return "", "", "", false
	}
	rest := strings.TrimPrefix(path, prefix)
	sep := "/artifacts/"
	idx := strings.Index(rest, sep)
	if idx < 0 {
		return "", "", "", false
	}
	namespace = rest[:idx]
	rest = rest[idx+len(sep):]
	if namespace == "" || rest == "" {
		return "", "", "", false
	}
	parts := strings.SplitN(rest, "/", 2)
	artifact = parts[0]
	if len(parts) == 2 {
		suffix = parts[1]
	}
	return namespace, artifact, suffix, true
}
