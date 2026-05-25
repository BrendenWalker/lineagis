package auth_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/BrendenWalker/verity/internal/auth"
	"github.com/go-jose/go-jose/v4"
	"github.com/golang-jwt/jwt/v5"
)

func TestAuthenticate_oidcGitHubClaims(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	kid := "test-key"
	jwk := jose.JSONWebKey{Key: &key.PublicKey, KeyID: kid, Algorithm: string(jose.RS256), Use: "sig"}
	jwksBody, err := json.Marshal(jose.JSONWebKeySet{Keys: []jose.JSONWebKey{jwk}})
	if err != nil {
		t.Fatal(err)
	}

	mux := http.NewServeMux()
	issuer := ""
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issuer":                                issuer,
			"jwks_uri":                              issuer + "/jwks",
			"authorization_endpoint":                issuer + "/auth",
			"token_endpoint":                        issuer + "/token",
			"response_types_supported":              []string{"id_token"},
			"subject_types_supported":               []string{"public"},
			"id_token_signing_alg_values_supported": []string{"RS256"},
		})
	})
	mux.HandleFunc("/jwks", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(jwksBody)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	issuer = srv.URL

	now := time.Now()
	claims := jwt.MapClaims{
		"iss":        issuer,
		"sub":        "repo:acme/widget:environment:prod",
		"aud":        "verity-api",
		"exp":        now.Add(time.Hour).Unix(),
		"iat":        now.Unix(),
		"repository": "acme/widget",
		"ref":        "refs/heads/main",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = kid
	raw, err := token.SignedString(key)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	a, err := auth.New(ctx, auth.Config{Issuer: issuer, Audience: "verity-api"})
	if err != nil {
		t.Fatal(err)
	}

	actor, err := a.Authenticate(ctx, raw)
	if err != nil {
		t.Fatalf("Authenticate: %v", err)
	}
	if actor.Dev {
		t.Fatal("expected OIDC actor")
	}
	if actor.GitHub == nil || actor.GitHub.Repository != "acme/widget" {
		t.Fatalf("github claims: %+v", actor.GitHub)
	}
	if err := auth.AuthorizeNamespace(actor, "gh/acme/widget", nil); err != nil {
		t.Fatalf("AuthorizeNamespace: %v", err)
	}
}

func TestAuthenticate_devToken(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	a, err := auth.New(ctx, auth.Config{DevToken: "local-dev"})
	if err != nil {
		t.Fatal(err)
	}
	actor, err := a.Authenticate(ctx, "local-dev")
	if err != nil {
		t.Fatal(err)
	}
	if !actor.Dev {
		t.Fatal("expected dev actor")
	}
}
