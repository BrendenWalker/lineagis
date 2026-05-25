package auth

import "context"

type contextKey string

const actorContextKey contextKey = "actor"

// Actor is the authenticated principal for an API request.
type Actor struct {
	Subject string
	Dev     bool
	GitHub  *GitHubClaims
}

// GitHubClaims holds OIDC claims used for namespace authorization (FR-API-003).
type GitHubClaims struct {
	Repository string
	Ref        string
}

// ContextWithActor attaches the authenticated actor to ctx.
func ContextWithActor(ctx context.Context, actor Actor) context.Context {
	return context.WithValue(ctx, actorContextKey, actor)
}

// ActorFromContext returns the authenticated actor when present.
func ActorFromContext(ctx context.Context) (Actor, bool) {
	v, ok := ctx.Value(actorContextKey).(Actor)
	return v, ok
}
