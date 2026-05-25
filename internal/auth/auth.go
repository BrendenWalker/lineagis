package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
)

// Config configures API authentication (OQ-API-002 dev stub + OIDC).
type Config struct {
	DevToken string
	Issuer   string
	Audience string
}

// Authenticator validates bearer tokens for protected API routes.
type Authenticator struct {
	devToken string
	verifier *oidc.IDTokenVerifier
}

// New builds an Authenticator. When Issuer is set, Audience is required.
func New(ctx context.Context, cfg Config) (*Authenticator, error) {
	a := &Authenticator{devToken: strings.TrimSpace(cfg.DevToken)}
	issuer := strings.TrimSpace(cfg.Issuer)
	audience := strings.TrimSpace(cfg.Audience)
	if issuer == "" {
		return a, nil
	}
	if audience == "" {
		return nil, fmt.Errorf("VERITY_OIDC_AUDIENCE is required when VERITY_OIDC_ISSUER is set")
	}
	provider, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		return nil, fmt.Errorf("oidc provider: %w", err)
	}
	a.verifier = provider.Verifier(&oidc.Config{ClientID: audience})
	return a, nil
}

var (
	ErrAuthRequired = errors.New("missing bearer token")
	ErrAuthInvalid  = errors.New("invalid bearer token")
	ErrForbidden    = errors.New("forbidden")
)

// Authenticate validates a raw bearer token (without the "Bearer " prefix).
func (a *Authenticator) Authenticate(ctx context.Context, token string) (Actor, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return Actor{}, ErrAuthRequired
	}

	if a.devToken != "" && token == a.devToken {
		subject := "dev:" + token[:min(8, len(token))]
		return Actor{Subject: subject, Dev: true}, nil
	}

	if a.verifier == nil {
		if a.devToken == "" {
			return Actor{}, ErrAuthRequired
		}
		return Actor{}, ErrAuthInvalid
	}

	idToken, err := a.verifier.Verify(ctx, token)
	if err != nil {
		return Actor{}, fmt.Errorf("%w: %v", ErrAuthInvalid, err)
	}

	var claims struct {
		Subject    string `json:"sub"`
		Repository string `json:"repository"`
		Ref        string `json:"ref"`
	}
	if err := idToken.Claims(&claims); err != nil {
		return Actor{}, fmt.Errorf("%w: parse claims: %v", ErrAuthInvalid, err)
	}
	subject := strings.TrimSpace(claims.Subject)
	if subject == "" {
		subject = idToken.Subject
	}

	gh := &GitHubClaims{
		Repository: strings.TrimSpace(claims.Repository),
		Ref:        strings.TrimSpace(claims.Ref),
	}
	return Actor{Subject: subject, GitHub: gh}, nil
}
