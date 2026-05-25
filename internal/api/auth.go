package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/BrendenWalker/verity/internal/auth"
)

// ActorFromContext returns the authenticated actor subject when present.
func ActorFromContext(ctx context.Context) string {
	a, ok := auth.ActorFromContext(ctx)
	if !ok {
		return ""
	}
	return a.Subject
}

// AuthMiddleware validates Authorization: Bearer using dev token and/or OIDC (FR-API-001).
func AuthMiddleware(a *auth.Authenticator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw, err := bearerToken(r.Header.Get("Authorization"))
			if err != nil {
				if errors.Is(err, errMissingBearer) {
					WriteError(w, http.StatusUnauthorized, "AUTH_REQUIRED", "missing bearer token", nil)
					return
				}
				WriteError(w, http.StatusUnauthorized, "AUTH_INVALID", err.Error(), nil)
				return
			}

			actor, err := a.Authenticate(r.Context(), raw)
			if err != nil {
				if errors.Is(err, auth.ErrAuthRequired) {
					WriteError(w, http.StatusUnauthorized, "AUTH_REQUIRED", err.Error(), nil)
					return
				}
				WriteError(w, http.StatusUnauthorized, "AUTH_INVALID", "invalid bearer token", nil)
				return
			}

			next.ServeHTTP(w, r.WithContext(auth.ContextWithActor(r.Context(), actor)))
		})
	}
}

// RequireBearer validates a fixed dev token (tests and legacy callers).
func RequireBearer(devToken string, next http.Handler) http.Handler {
	a, err := auth.New(context.Background(), auth.Config{DevToken: devToken})
	if err != nil {
		panic(err)
	}
	return AuthMiddleware(a)(next)
}

var errMissingBearer = errors.New("missing bearer token")

func bearerToken(header string) (string, error) {
	if header == "" {
		return "", errMissingBearer
	}
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return "", errors.New("invalid authorization header")
	}
	token := strings.TrimSpace(strings.TrimPrefix(header, prefix))
	if token == "" {
		return "", errors.New("empty bearer token")
	}
	return token, nil
}

func authorizeNamespace(ctx context.Context, ns string, config json.RawMessage) error {
	actor, ok := auth.ActorFromContext(ctx)
	if !ok {
		return auth.ErrAuthRequired
	}
	if err := auth.AuthorizeNamespace(actor, ns, config); err != nil {
		return err
	}
	return nil
}
