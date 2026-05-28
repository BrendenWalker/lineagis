package inspect_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/BrendenWalker/verity/internal/apiclient"
	"github.com/BrendenWalker/verity/internal/inspect"
)

func TestRun_digestValid(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v1/namespaces/ns/artifacts/app/trust" {
			http.NotFound(w, r)
			return
		}
		if r.URL.Query().Get("digest") != "sha256:abc" {
			http.Error(w, "bad digest", http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"signatures": map[string]string{"status": "valid"},
			"overall":    "pass",
		})
	}))
	defer srv.Close()

	result, err := inspect.Run(context.Background(), apiclient.New(srv.URL, "tok"), inspect.Options{
		Namespace: "ns",
		Artifact:  "app",
		Ref:       "sha256:abc",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.MustLines) != 1 || result.MustLines[0].Text != "✓ Signed by GitHub Actions" || !result.MustLines[0].Pass {
		t.Fatalf("got %+v", result.MustLines)
	}
	if inspect.MustFailed(result.MustLines) {
		t.Fatal("expected pass")
	}
}

func TestRun_digestInvalid(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"signatures": map[string]string{"status": "invalid"},
		})
	}))
	defer srv.Close()

	result, err := inspect.Run(context.Background(), apiclient.New(srv.URL, "tok"), inspect.Options{
		Namespace: "ns",
		Artifact:  "app",
		Ref:       "sha256:dead",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.MustLines) != 1 || result.MustLines[0].Pass {
		t.Fatalf("got %+v", result.MustLines)
	}
	if result.MustLines[0].RequirementID != "FR-SIGN-005" {
		t.Fatalf("requirement_id = %q", result.MustLines[0].RequirementID)
	}
	if !strings.Contains(result.MustLines[0].Text, "FR-SIGN-005") {
		t.Fatalf("text = %q", result.MustLines[0].Text)
	}
	if !inspect.MustFailed(result.MustLines) {
		t.Fatal("expected Must failure")
	}
}

func TestRun_tagMissing(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("tag") != "1.0.0" {
			http.Error(w, "bad tag", http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"signatures": map[string]string{"status": "missing"},
		})
	}))
	defer srv.Close()

	result, err := inspect.Run(context.Background(), apiclient.New(srv.URL, "tok"), inspect.Options{
		Namespace: "ns",
		Artifact:  "app",
		Ref:       "1.0.0",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.MustLines) != 1 || result.MustLines[0].Pass {
		t.Fatalf("got %+v", result.MustLines)
	}
	if result.MustLines[0].RequirementID != "FR-SIGN-005" {
		t.Fatalf("requirement_id = %q", result.MustLines[0].RequirementID)
	}
	if !strings.Contains(result.MustLines[0].Text, "FR-SIGN-005") {
		t.Fatalf("text = %q", result.MustLines[0].Text)
	}
	if !inspect.MustFailed(result.MustLines) {
		t.Fatal("expected Must failure")
	}
}

func TestRun_localPath(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(dir+"/f.txt", []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	var gotDigest string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotDigest = r.URL.Query().Get("digest")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"signatures": map[string]string{"status": "valid"},
			"digest":     gotDigest,
		})
	}))
	defer srv.Close()

	result, err := inspect.Run(context.Background(), apiclient.New(srv.URL, "tok"), inspect.Options{
		Namespace: "ns",
		Artifact:  "app",
		Ref:       dir,
	})
	if err != nil {
		t.Fatal(err)
	}
	if gotDigest == "" || !strings.HasPrefix(gotDigest, "sha256:") {
		t.Fatalf("digest = %q", gotDigest)
	}
	if inspect.MustFailed(result.MustLines) {
		t.Fatalf("lines=%+v", result.MustLines)
	}
}

func TestRun_policyRuleOnSignatureFailure(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"namespace":  "ns",
			"artifact":   "app",
			"digest":     "sha256:unsigned",
			"signatures": map[string]string{"status": "missing"},
			"policy": map[string]any{
				"status": "fail",
				"reasons": []map[string]string{
					{"rule": "require-signatures", "message": "digest has no signature; attach a Sigstore bundle before verify"},
				},
			},
		})
	}))
	defer srv.Close()

	result, err := inspect.Run(context.Background(), apiclient.New(srv.URL, "tok"), inspect.Options{
		Namespace: "ns",
		Artifact:  "app",
		Ref:       "sha256:unsigned",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.MustLines) != 1 {
		t.Fatalf("lines = %+v", result.MustLines)
	}
	line := result.MustLines[0]
	if line.RuleID != "require-signatures" {
		t.Fatalf("rule_id = %q", line.RuleID)
	}
	if !strings.Contains(line.Text, "rule require-signatures") {
		t.Fatalf("text = %q", line.Text)
	}
}

func TestJSONReport_schemaFields(t *testing.T) {
	t.Parallel()
	result := &inspect.Result{
		Trust: &apiclient.TrustStatus{
			Namespace: "ns",
			Artifact:  "app",
			Digest:    "sha256:abc",
		},
		MustLines: []inspect.ChecklistLine{
			{Text: "✓ Signed by GitHub Actions", Must: true, Pass: true, RequirementID: "FR-SIGN-005"},
		},
	}
	report := inspect.JSONReport(result)
	if report.Version != 1 || report.Overall != "pass" || len(report.Checks) != 1 {
		t.Fatalf("got %+v", report)
	}
	if report.Checks[0].RequirementID != "FR-SIGN-005" || report.Checks[0].Status != "pass" {
		t.Fatalf("check = %+v", report.Checks[0])
	}
}

func TestHumanLines_includesTrustHeader(t *testing.T) {
	t.Parallel()
	result := &inspect.Result{
		MustLines: []inspect.ChecklistLine{
			{Text: "✓ Signed by GitHub Actions", Must: true, Pass: true},
		},
		ShouldLines: []inspect.ChecklistLine{
			{Text: "⚠ SBOM not attached", Must: false, Pass: false},
		},
	}
	lines := inspect.HumanLines(result)
	if len(lines) != 3 || lines[0] != inspect.TrustHeader {
		t.Fatalf("lines = %v", lines)
	}
}

func TestMustFailed_ignoresShouldOnly(t *testing.T) {
	t.Parallel()
	if inspect.MustFailed([]inspect.ChecklistLine{{Must: false, Pass: false}}) {
		t.Fatal("should-only lines must not count as Must failure")
	}
	if !inspect.MustFailed([]inspect.ChecklistLine{{Must: true, Pass: false}}) {
		t.Fatal("must failure expected")
	}
}

func TestMustChecklist_signatureStates(t *testing.T) {
	t.Parallel()
	cases := []struct {
		status string
		text   string
		pass   bool
	}{
		{"valid", "✓ Signed by GitHub Actions", true},
		{"missing", "Signature missing", false},
		{"invalid", "Signature invalid", false},
		{"weird", "Signature status unknown (weird)", false},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.status, func(t *testing.T) {
			t.Parallel()
			trust := &apiclient.TrustStatus{}
			trust.Signatures.Status = tc.status
			lines := inspect.MustChecklist(trust)
			if len(lines) != 1 || lines[0].Pass != tc.pass || !lines[0].Must {
				t.Fatalf("got %+v", lines)
			}
			if tc.pass {
				if lines[0].Text != tc.text {
					t.Fatalf("text = %q", lines[0].Text)
				}
			} else if !strings.Contains(lines[0].Text, tc.text) || !strings.Contains(lines[0].Text, "FR-SIGN-005") {
				t.Fatalf("text = %q", lines[0].Text)
			}
		})
	}
}
