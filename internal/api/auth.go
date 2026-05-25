package api

import (
	"context"
	"net/http"
	"strings"
)

type contextKey string

const actorContextKey contextKey = "actor"

// ActorFromContext returns the authenticated actor subject when present.
func ActorFromContext(ctx context.Context) string {
	v, _ := ctx.Value(actorContextKey).(string)
	return v
}

// RequireBearer validates Authorization: Bearer and attaches actor to context.
// When devToken is non-empty, it must match exactly (OQ-API-002 local dev stub).
func RequireBearer(devToken string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth == "" {
			WriteError(w, http.StatusUnauthorized, "AUTH_REQUIRED", "missing bearer token", nil)
			return
		}
		const prefix = "Bearer "
		if !strings.HasPrefix(auth, prefix) {
			WriteError(w, http.StatusUnauthorized, "AUTH_INVALID", "invalid authorization header", nil)
			return
		}
		token := strings.TrimSpace(strings.TrimPrefix(auth, prefix))
		if token == "" {
			WriteError(w, http.StatusUnauthorized, "AUTH_INVALID", "empty bearer token", nil)
			return
		}

		if devToken == "" {
			WriteError(w, http.StatusUnauthorized, "AUTH_REQUIRED", "api authentication not configured", nil)
			return
		}
		if token != devToken {
			WriteError(w, http.StatusUnauthorized, "AUTH_INVALID", "invalid bearer token", nil)
			return
		}

		ctx := context.WithValue(r.Context(), actorContextKey, "dev:"+token[:min(8, len(token))])
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
