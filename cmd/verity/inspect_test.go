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
)

func TestRunInspect_printsMustLines(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
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
	if got := strings.TrimSpace(out.String()); got != "✓ Signed by GitHub Actions" {
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
