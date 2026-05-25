package auth_test

import (
	"encoding/json"
	"testing"

	"github.com/BrendenWalker/verity/internal/auth"
)

func TestIsOperator(t *testing.T) {
	t.Parallel()
	cfg := json.RawMessage(`{"operators":["alice","bob"]}`)
	if !auth.IsOperator(auth.Actor{Subject: "alice"}, cfg) {
		t.Fatal("alice should be operator")
	}
	if auth.IsOperator(auth.Actor{Subject: "carol"}, cfg) {
		t.Fatal("carol should not be operator")
	}
	if !auth.IsOperator(auth.Actor{Dev: true}, cfg) {
		t.Fatal("dev actor should be operator")
	}
}

func TestAuthorizeRole_maintainer(t *testing.T) {
	t.Parallel()
	actor := auth.Actor{
		Subject: "repo:acme/widget:ref:refs/heads/main",
		GitHub:  &auth.GitHubClaims{Repository: "acme/widget", Ref: "refs/heads/main"},
	}
	if err := auth.AuthorizeRole(actor, "gh/acme/widget", nil, auth.RoleMaintainer); err != nil {
		t.Fatalf("github maintainer: %v", err)
	}
}

func TestAuthorizeRole_operatorRequired(t *testing.T) {
	t.Parallel()
	actor := auth.Actor{Subject: "not-listed"}
	cfg := json.RawMessage(`{"operators":["alice"]}`)
	if err := auth.AuthorizeRole(actor, "gh/acme/widget", cfg, auth.RoleOperator); err == nil {
		t.Fatal("expected operator rejection")
	}
}
