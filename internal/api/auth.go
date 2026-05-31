package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/BrendenWalker/lineagis/internal/auth"
)

// ActorFromContext returns the authenticated actor subject when present.
func ActorFromContext(ctx context.Context) string {
	a, ok := auth.ActorFromContext(ctx)
	if !ok {
		return ""
	}
	return a.Subject
}

// AuthMiddleware validates Authorization: Bearer when present (FR-API-001).
// Missing bearer is allowed so GET handlers can enforce namespace read rules (FR-API-005).
// Write handlers must call requireActor.
func AuthMiddleware(a *auth.Authenticator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw, err := bearerToken(r.Header.Get("Authorization"))
			if err != nil {
				if errors.Is(err, errMissingBearer) {
					next.ServeHTTP(w, r)
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

func requireActor(ctx context.Context) error {
	if _, ok := auth.ActorFromContext(ctx); !ok {
		return auth.ErrAuthRequired
	}
	return nil
}

func authorizeRead(ctx context.Context, ns string, config json.RawMessage) error {
	if auth.AllowsAnonymousRead(config) {
		return nil
	}
	actor, ok := auth.ActorFromContext(ctx)
	if !ok {
		return auth.ErrAuthRequired
	}
	return auth.AuthorizeRole(actor, ns, config, auth.RoleReader)
}

func authorizeOperator(ctx context.Context, ns string, config json.RawMessage) error {
	if err := requireActor(ctx); err != nil {
		return err
	}
	actor, _ := auth.ActorFromContext(ctx)
	return auth.AuthorizeRole(actor, ns, config, auth.RoleOperator)
}

// RequireAuthMiddleware rejects requests without a valid bearer token.
func RequireAuthMiddleware(a *auth.Authenticator) func(http.Handler) http.Handler {
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
	return RequireAuthMiddleware(a)(next)
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
	if err := requireActor(ctx); err != nil {
		return err
	}
	actor, _ := auth.ActorFromContext(ctx)
	if err := auth.AuthorizeRole(actor, ns, config, auth.RoleMaintainer); err != nil {
		return err
	}
	return nil
}

func writeAuthError(w http.ResponseWriter, err error) {
	if errors.Is(err, auth.ErrAuthRequired) {
		WriteError(w, http.StatusUnauthorized, "AUTH_REQUIRED", err.Error(), nil)
		return
	}
	WriteError(w, http.StatusForbidden, "FORBIDDEN", err.Error(), nil)
}
