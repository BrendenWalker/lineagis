package inspect_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/BrendenWalker/verity/internal/apiclient"
	"github.com/BrendenWalker/verity/internal/inspect"
	"github.com/BrendenWalker/verity/internal/registry"
	"github.com/BrendenWalker/verity/internal/signing"
)

func TestVerifyLocally_missingSignatures(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"signatures": []any{}})
	}))
	defer srv.Close()

	api := apiclient.New(srv.URL, "tok")
	reg, err := registry.New("http://127.0.0.1:1")
	if err != nil {
		t.Fatal(err)
	}

	out, err := inspect.VerifyLocally(context.Background(), reg, api, "ns", "app", "sha256:abc")
	if err != nil {
		t.Fatal(err)
	}
	if out.Status != "missing" {
		t.Fatalf("status = %q, want missing", out.Status)
	}
}

func TestMustChecklist_localVerifyValid(t *testing.T) {
	t.Parallel()
	local := &inspect.LocalVerifyResult{
		Status: "valid",
		Signer: signing.GitHubPublisher{Repository: "acme/widget", Workflow: "release.yml"},
	}
	trust := &apiclient.TrustStatus{Overall: "pass"}
	trust.Signatures.Status = "valid"
	lines := inspect.MustChecklist(trust, local)
	if len(lines) < 2 {
		t.Fatalf("expected local + API lines, got %d", len(lines))
	}
	if !lines[0].Pass || lines[0].RequirementID != "FR-SIGN-005" {
		t.Fatalf("local line: %+v", lines[0])
	}
	if !lines[1].Pass {
		t.Fatalf("API line: %+v", lines[1])
	}
}

func TestMustChecklist_localVerifyInvalid(t *testing.T) {
	t.Parallel()
	local := &inspect.LocalVerifyResult{Status: "invalid"}
	trust := &apiclient.TrustStatus{}
	trust.Signatures.Status = "valid"
	lines := inspect.MustChecklist(trust, local)
	if len(lines) == 0 || lines[0].Pass {
		t.Fatalf("expected failing local line, got %+v", lines)
	}
	if lines[0].RequirementID != "FR-SIGN-005" {
		t.Fatalf("requirement_id = %q", lines[0].RequirementID)
	}
}
