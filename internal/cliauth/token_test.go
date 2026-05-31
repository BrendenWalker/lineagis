package cliauth_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/BrendenWalker/lineagis/internal/cliauth"
)

func TestResolveToken_env(t *testing.T) {
	t.Setenv("LINEAGIS_TOKEN", "abc")
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	got, err := cliauth.ResolveToken(context.Background())
	if err != nil || got != "abc" {
		t.Fatalf("got %q err %v", got, err)
	}
}

func TestResolveToken_githubActions(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method", http.StatusMethodNotAllowed)
			return
		}
		_, _ = w.Write([]byte(`{"value":"jwt-from-gha"}`))
	}))
	defer srv.Close()

	t.Setenv("LINEAGIS_TOKEN", "")
	t.Setenv("LINEAGIS_DEV_TOKEN", "")
	t.Setenv("ACTIONS_ID_TOKEN_REQUEST_URL", srv.URL)
	t.Setenv("ACTIONS_ID_TOKEN_REQUEST_TOKEN", "req-tok")
	t.Setenv("LINEAGIS_OIDC_AUDIENCE", "lineagis-api")
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	got, err := cliauth.ResolveToken(context.Background())
	if err != nil || got != "jwt-from-gha" {
		t.Fatalf("got %q err %v", got, err)
	}
}

func TestSaveLoadFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	f := cliauth.File{APIURL: "http://api", Token: "secret"}
	if err := cliauth.SaveFile(f); err != nil {
		t.Fatal(err)
	}
	path, _ := cliauth.ConfigPath()
	if runtime.GOOS != "windows" {
		st, err := os.Stat(path)
		if err != nil {
			t.Fatalf("stat: %v", err)
		}
		if perm := st.Mode().Perm(); perm&0077 != 0 {
			t.Fatalf("config should not be group/world accessible, mode %o", perm)
		}
	}
	loaded, err := cliauth.LoadFile()
	if err != nil || loaded.Token != "secret" {
		t.Fatalf("loaded %+v err %v", loaded, err)
	}
	_ = filepath.Dir(path)
}
