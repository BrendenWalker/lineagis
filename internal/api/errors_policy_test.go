package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWritePolicyFailed_details(t *testing.T) {
	t.Parallel()
	rec := httptest.NewRecorder()
	writePolicyFailed(rec, PolicyFailure{
		Rule: "require-signatures",
		Hint: "attach a Sigstore bundle before tagging",
	})
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d", rec.Code)
	}
	var body ErrorBody
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Code != "POLICY_FAILED" {
		t.Fatalf("code = %q", body.Code)
	}
	if body.Details["rule"] != "require-signatures" {
		t.Fatalf("rule = %v", body.Details["rule"])
	}
	if body.Details["hint"] != "attach a Sigstore bundle before tagging" {
		t.Fatalf("hint = %v", body.Details["hint"])
	}
}
