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

	if ns, ok := parseNamespacePolicyPath(rest); ok {
		if r.Method == http.MethodGet {
			h.getPolicy(w, r, ns)
			return
		}
		if r.Method == http.MethodPut {
			h.putPolicy(w, r, ns)
			return
		}
		WriteError(w, http.StatusNotFound, "NOT_FOUND", "unknown route", nil)
		return
	}

	if ns, ok := parseNamespaceAuditPath(rest); ok {
		if r.Method == http.MethodGet {
			h.getAuditEvents(w, r, ns)
			return
		}
		WriteError(w, http.StatusNotFound, "NOT_FOUND", "unknown route", nil)
		return
	}

	if ns, ok := parseNamespaceArtifactsListPath(rest); ok {
		if r.Method == http.MethodGet {
			h.listArtifacts(w, r, ns)
			return
		}
		WriteError(w, http.StatusNotFound, "NOT_FOUND", "unknown route", nil)
		return
	}

	ns, artifact, suffix, ok := parseNamespacesPath(rest)
	if !ok {
		WriteError(w, http.StatusNotFound, "NOT_FOUND", "unknown route", nil)
		return
	}

	switch {
	case r.Method == http.MethodGet && suffix == "":
		h.getArtifact(w, r, ns, artifact)
	case r.Method == http.MethodGet && suffix == "trust":
		h.getTrustStatus(w, r, ns, artifact)
	case r.Method == http.MethodPost && suffix == "verify":
		h.postVerify(w, r, ns, artifact)
	case r.Method == http.MethodPost && suffix == "policy/evaluate":
		h.postEvaluatePolicy(w, r, ns, artifact)
	case r.Method == http.MethodGet && strings.HasPrefix(suffix, "tags/"):
		tag := strings.TrimPrefix(suffix, "tags/")
		if tag == "" || strings.Contains(tag, "/") {
			WriteError(w, http.StatusNotFound, "NOT_FOUND", "unknown route", nil)
			return
		}
		h.getTag(w, r, ns, artifact, tag)
	case r.Method == http.MethodPut && suffix == "":
		h.putArtifact(w, r, ns, artifact)
	case r.Method == http.MethodPost && suffix == "digests":
		h.postRegisterDigest(w, r, ns, artifact)
	case r.Method == http.MethodPost && suffix == "signatures":
		h.postAttachSignature(w, r, ns, artifact)
	case r.Method == http.MethodGet && suffix == "signatures":
		h.getListSignatures(w, r, ns, artifact)
	case r.Method == http.MethodPost && strings.HasPrefix(suffix, "digests/") && strings.HasSuffix(suffix, "/attestations"):
		rest := strings.TrimPrefix(suffix, "digests/")
		rest = strings.TrimSuffix(rest, "/attestations")
		if rest == "" || strings.Contains(rest, "/") {
			WriteError(w, http.StatusNotFound, "NOT_FOUND", "unknown route", nil)
			return
		}
		h.postAttachAttestation(w, r, ns, artifact, rest)
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

// parseNamespaceArtifactsListPath parses namespaces/{ns}/artifacts.
func parseNamespaceArtifactsListPath(path string) (namespace string, ok bool) {
	const prefix = "namespaces/"
	if !strings.HasPrefix(path, prefix) {
		return "", false
	}
	rest := strings.TrimPrefix(path, prefix)
	if rest == "" {
		return "", false
	}
	const suffix = "/artifacts"
	if !strings.HasSuffix(rest, suffix) {
		return "", false
	}
	ns := strings.TrimSuffix(rest, suffix)
	if ns == "" || strings.Contains(ns, "/artifacts") {
		return "", false
	}
	return ns, true
}

// parseNamespaceAuditPath parses namespaces/{ns}/audit.
func parseNamespaceAuditPath(path string) (namespace string, ok bool) {
	const prefix = "namespaces/"
	const suffix = "/audit"
	if !strings.HasPrefix(path, prefix) || !strings.HasSuffix(path, suffix) {
		return "", false
	}
	rest := strings.TrimPrefix(path, prefix)
	ns := strings.TrimSuffix(rest, suffix)
	if ns == "" || strings.Contains(ns, "/audit") {
		return "", false
	}
	return ns, true
}

// parseNamespacePolicyPath parses namespaces/{ns}/policy.
func parseNamespacePolicyPath(path string) (namespace string, ok bool) {
	const prefix = "namespaces/"
	const suffix = "/policy"
	if !strings.HasPrefix(path, prefix) || !strings.HasSuffix(path, suffix) {
		return "", false
	}
	rest := strings.TrimPrefix(path, prefix)
	ns := strings.TrimSuffix(rest, suffix)
	if ns == "" || strings.Contains(ns, "/policy") {
		return "", false
	}
	return ns, true
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
