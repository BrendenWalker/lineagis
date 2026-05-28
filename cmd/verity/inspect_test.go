package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/BrendenWalker/verity/internal/inspect"
)

func TestRunInspect_printsMustLines(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"namespace":  "ns",
			"artifact":   "app",
			"digest":     "sha256:abc",
			"signatures": map[string]string{"status": "valid"},
			"overall":    "pass",
		})
	}))
	defer srv.Close()

	t.Setenv("VERITY_TOKEN", "tok")
	t.Setenv("VERITY_API_URL", srv.URL)

	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	code := run([]string{"inspect", "sha256:abc", "--namespace", "ns", "--artifact", "app"})
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	os.Stdout = old

	var out bytes.Buffer
	if _, err := io.Copy(&out, r); err != nil {
		t.Fatal(err)
	}
	if err := r.Close(); err != nil {
		t.Fatal(err)
	}
	if code != 0 {
		t.Fatalf("exit = %d, want 0", code)
	}
	got := out.String()
	if !strings.Contains(got, inspect.TrustHeader) {
		t.Fatalf("stdout missing trust header: %q", got)
	}
	if !strings.Contains(got, "✓ Signed by GitHub Actions") {
		t.Fatalf("stdout = %q", got)
	}
}

func TestRunInspect_exitOnMustFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"signatures": map[string]string{"status": "missing"},
		})
	}))
	defer srv.Close()

	t.Setenv("VERITY_TOKEN", "tok")
	t.Setenv("VERITY_API_URL", srv.URL)

	if got := run([]string{"inspect", "sha256:abc", "--namespace", "ns", "--artifact", "app"}); got != 1 {
		t.Fatalf("exit = %d, want 1", got)
	}
}

func TestRunInspect_outputJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"namespace":  "ns",
			"artifact":   "app",
			"digest":     "sha256:abc",
			"signatures": map[string]string{"status": "valid"},
			"overall":    "pass",
		})
	}))
	defer srv.Close()

	t.Setenv("VERITY_TOKEN", "tok")
	t.Setenv("VERITY_API_URL", srv.URL)

	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	code := run([]string{"inspect", "sha256:abc", "--namespace", "ns", "--artifact", "app", "--output", "json"})
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	os.Stdout = old

	var out bytes.Buffer
	if _, err := io.Copy(&out, r); err != nil {
		t.Fatal(err)
	}
	if err := r.Close(); err != nil {
		t.Fatal(err)
	}
	if code != 0 {
		t.Fatalf("exit = %d, want 0", code)
	}

	var report struct {
		Version int    `json:"version"`
		Overall string `json:"overall"`
		Checks  []struct {
			Status        string `json:"status"`
			RequirementID string `json:"requirement_id"`
		} `json:"checks"`
	}
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatalf("json: %v body=%s", err, out.String())
	}
	if report.Version != 1 || report.Overall != "pass" || len(report.Checks) < 1 {
		t.Fatalf("report = %+v", report)
	}
	if report.Checks[0].Status != "pass" || report.Checks[0].RequirementID != "FR-SIGN-005" {
		t.Fatalf("check = %+v", report.Checks[0])
	}
}

func TestRunInspect_shouldWarningExitZero(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"namespace":  "ns",
			"artifact":   "app",
			"digest":     "sha256:abc",
			"signatures": map[string]string{"status": "valid"},
			"attestations": map[string]any{
				"sbom": false,
			},
		})
	}))
	defer srv.Close()

	t.Setenv("VERITY_TOKEN", "tok")
	t.Setenv("VERITY_API_URL", srv.URL)

	if got := run([]string{"inspect", "sha256:abc", "--namespace", "ns", "--artifact", "app"}); got != 0 {
		t.Fatalf("exit = %d, want 0 when only Should lines warn", got)
	}
}

func TestRunInspect_outputJSONMustFailureExit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"namespace":  "ns",
			"artifact":   "app",
			"digest":     "sha256:abc",
			"signatures": map[string]string{"status": "missing"},
			"policy": map[string]any{
				"status": "fail",
				"reasons": []map[string]string{
					{"rule": "require-signatures", "message": "digest has no signature"},
				},
			},
		})
	}))
	defer srv.Close()

	t.Setenv("VERITY_TOKEN", "tok")
	t.Setenv("VERITY_API_URL", srv.URL)

	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	code := run([]string{"inspect", "sha256:abc", "--namespace", "ns", "--artifact", "app", "--output", "json"})
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	os.Stdout = old

	var out bytes.Buffer
	_, _ = io.Copy(&out, r)
	_ = r.Close()

	if code != 1 {
		t.Fatalf("exit = %d, want 1", code)
	}
	var report struct {
		Overall string `json:"overall"`
		Checks  []struct {
			Status string `json:"status"`
			RuleID string `json:"rule_id"`
		} `json:"checks"`
	}
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if report.Overall != "fail" {
		t.Fatalf("overall = %q", report.Overall)
	}
	if len(report.Checks) < 1 || report.Checks[0].Status != "fail" || report.Checks[0].RuleID != "require-signatures" {
		t.Fatalf("checks = %+v", report.Checks)
	}
}
