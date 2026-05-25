package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/BrendenWalker/verity/internal/metadata"
)

// ErrorBody is the JSON error envelope per api.md.
type ErrorBody struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

// WriteError writes a JSON error response.
func WriteError(w http.ResponseWriter, status int, code, message string, details map[string]any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(ErrorBody{
		Code:    code,
		Message: message,
		Details: details,
	})
}

// mapStoreError converts metadata store errors to HTTP status and API codes.
func mapStoreError(w http.ResponseWriter, err error) bool {
	if err == nil {
		return false
	}
	switch {
	case errors.Is(err, metadata.ErrNotFound):
		WriteError(w, http.StatusNotFound, "NOT_FOUND", err.Error(), nil)
	case errors.Is(err, metadata.ErrDigestWrongArtifact):
		WriteError(w, http.StatusConflict, "CONFLICT", err.Error(), nil)
	default:
		WriteError(w, http.StatusInternalServerError, "INTERNAL", "internal server error", nil)
	}
	return true
}
