package api

import (
	"context"
	"encoding/json"
	"testing"
)

func TestEvaluatePolicyDocument_requireDigestOnVerify_byTag(t *testing.T) {
	t.Parallel()
	doc := []byte(`{"rules":[{"id":"require-digest-on-verify"}]}`)
	reasons, err := evaluatePolicyDocument(
		context.Background(),
		nil,
		"gh/acme/widget",
		nil,
		doc,
		0,
		EvalPhaseVerify,
		policyEvalContext{verifyByTag: true},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(reasons) != 1 {
		t.Fatalf("reasons = %+v", reasons)
	}
	if reasons[0].Rule != "require-digest-on-verify" {
		t.Fatalf("rule = %q", reasons[0].Rule)
	}
	if reasons[0].Message == "" {
		t.Fatal("expected message")
	}
}

func TestEvaluatePolicyDocument_requireDigestOnVerify_digestOK(t *testing.T) {
	t.Parallel()
	doc := []byte(`{"rules":[{"id":"require-digest-on-verify"}]}`)
	reasons, err := evaluatePolicyDocument(
		context.Background(),
		nil,
		"gh/acme/widget",
		nil,
		doc,
		0,
		EvalPhaseVerify,
		policyEvalContext{verifyByTag: false},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(reasons) != 0 {
		t.Fatalf("reasons = %+v", reasons)
	}
}

func TestParseRepositoryOwnershipConfig(t *testing.T) {
	t.Parallel()
	raw := json.RawMessage(`{"verify_with_github_api":true}`)
	cfg := parseRepositoryOwnershipConfig(raw)
	if !cfg.VerifyWithGitHubAPI {
		t.Fatalf("cfg = %+v", cfg)
	}
}
