package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/BrendenWalker/verity/internal/metadata"
)

type tagSummary struct {
	Name   string `json:"name"`
	Digest string `json:"digest"`
}

type getArtifactResponse struct {
	Name      string       `json:"name"`
	Namespace string       `json:"namespace"`
	Tags      []tagSummary `json:"tags"`
}

type listArtifactsResponse struct {
	Namespace string             `json:"namespace"`
	Artifacts []artifactResponse `json:"artifacts"`
	Limit     int                `json:"limit"`
	Offset    int                `json:"offset"`
}

type getTagResponse struct {
	Tag       string `json:"tag"`
	Digest    string `json:"digest"`
	Artifact  string `json:"artifact"`
	Namespace string `json:"namespace"`
}

func (h *Handler) listArtifacts(w http.ResponseWriter, r *http.Request, ns string) {
	if ns == "" {
		WriteError(w, http.StatusBadRequest, "VALIDATION_FAILED", "namespace is required", nil)
		return
	}
	ctx := r.Context()
	namespace, err := h.Store.GetNamespaceByName(ctx, ns)
	if err != nil {
		if mapStoreError(w, err) {
			return
		}
	}
	if err := authorizeRead(ctx, ns, namespace.Config); err != nil {
		writeAuthError(w, err)
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	arts, err := h.Store.ListArtifacts(ctx, namespace.ID, limit, offset)
	if err != nil {
		if mapStoreError(w, err) {
			return
		}
	}

	resp := listArtifactsResponse{
		Namespace: namespace.Name,
		Limit:     limitOrDefault(limit),
		Offset:    offset,
	}
	for _, a := range arts {
		resp.Artifacts = append(resp.Artifacts, artifactResponse{Name: a.Name, Namespace: namespace.Name})
	}
	if resp.Artifacts == nil {
		resp.Artifacts = []artifactResponse{}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *Handler) getArtifact(w http.ResponseWriter, r *http.Request, ns, artifact string) {
	if ns == "" || artifact == "" {
		WriteError(w, http.StatusBadRequest, "VALIDATION_FAILED", "namespace and artifact are required", nil)
		return
	}
	ctx := r.Context()
	namespace, err := h.Store.GetNamespaceByName(ctx, ns)
	if err != nil {
		if mapStoreError(w, err) {
			return
		}
	}
	if err := authorizeRead(ctx, ns, namespace.Config); err != nil {
		writeAuthError(w, err)
		return
	}
	art, err := h.Store.GetArtifact(ctx, namespace.ID, artifact)
	if err != nil {
		if mapStoreError(w, err) {
			return
		}
	}

	tags, err := h.Store.ListTagsForArtifact(ctx, art.ID)
	if err != nil {
		if mapStoreError(w, err) {
			return
		}
	}

	summaries := make([]tagSummary, 0, len(tags))
	for _, t := range tags {
		d, err := h.Store.GetDigestByID(ctx, t.DigestID)
		if err != nil {
			if mapStoreError(w, err) {
				return
			}
		}
		summaries = append(summaries, tagSummary{Name: t.Name, Digest: d.Digest})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(getArtifactResponse{
		Name:      art.Name,
		Namespace: namespace.Name,
		Tags:      summaries,
	})
}

func (h *Handler) getTag(w http.ResponseWriter, r *http.Request, ns, artifact, tag string) {
	if ns == "" || artifact == "" || tag == "" {
		WriteError(w, http.StatusBadRequest, "VALIDATION_FAILED", "namespace, artifact, and tag are required", nil)
		return
	}
	ctx := r.Context()
	namespace, err := h.Store.GetNamespaceByName(ctx, ns)
	if err != nil {
		if mapStoreError(w, err) {
			return
		}
	}
	if err := authorizeRead(ctx, ns, namespace.Config); err != nil {
		writeAuthError(w, err)
		return
	}
	art, err := h.Store.GetArtifact(ctx, namespace.ID, artifact)
	if err != nil {
		if mapStoreError(w, err) {
			return
		}
	}
	tagRow, err := h.Store.GetTag(ctx, art.ID, tag)
	if err != nil {
		if mapStoreError(w, err) {
			return
		}
	}
	d, err := h.Store.GetDigestByID(ctx, tagRow.DigestID)
	if err != nil {
		if mapStoreError(w, err) {
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(getTagResponse{
		Tag:       tagRow.Name,
		Digest:    d.Digest,
		Artifact:  art.Name,
		Namespace: namespace.Name,
	})
}

func (h *Handler) getTrustStatus(w http.ResponseWriter, r *http.Request, ns, artifact string) {
	if ns == "" || artifact == "" {
		WriteError(w, http.StatusBadRequest, "VALIDATION_FAILED", "namespace and artifact are required", nil)
		return
	}
	digestRef := strings.TrimSpace(r.URL.Query().Get("digest"))
	tagName := strings.TrimSpace(r.URL.Query().Get("tag"))
	if digestRef == "" && tagName == "" {
		WriteError(w, http.StatusBadRequest, "VALIDATION_FAILED", "digest or tag query parameter is required", nil)
		return
	}
	if digestRef != "" && tagName != "" {
		WriteError(w, http.StatusBadRequest, "VALIDATION_FAILED", "provide digest or tag, not both", nil)
		return
	}

	ctx := r.Context()
	namespace, err := h.Store.GetNamespaceByName(ctx, ns)
	if err != nil {
		if mapStoreError(w, err) {
			return
		}
	}
	if err := authorizeRead(ctx, ns, namespace.Config); err != nil {
		writeAuthError(w, err)
		return
	}
	art, err := h.Store.GetArtifact(ctx, namespace.ID, artifact)
	if err != nil {
		if mapStoreError(w, err) {
			return
		}
	}

	d, err := h.resolveTrustDigest(ctx, art.ID, tagName, digestRef)
	if err != nil {
		if mapStoreError(w, err) {
			return
		}
	}
	if d.ArtifactID != art.ID {
		WriteError(w, http.StatusNotFound, "NOT_FOUND", "digest not found for artifact", nil)
		return
	}

	resp, err := buildTrustStatus(ctx, h.Store, namespace.ID, ns, artifact, d)
	if err != nil {
		if mapStoreError(w, err) {
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *Handler) resolveTrustDigest(ctx context.Context, artifactID int64, tagName, digestRef string) (*metadata.Digest, error) {
	if tagName != "" {
		tagRow, err := h.Store.GetTag(ctx, artifactID, tagName)
		if err != nil {
			return nil, err
		}
		return h.Store.GetDigestByID(ctx, tagRow.DigestID)
	}
	return h.Store.GetDigestByString(ctx, digestRef)
}

func limitOrDefault(limit int) int {
	if limit <= 0 {
		return 50
	}
	if limit > 200 {
		return 200
	}
	return limit
}
