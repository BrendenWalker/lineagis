package git_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/BrendenWalker/lineagis/internal/core/model"
	"github.com/BrendenWalker/lineagis/internal/ingest/git"
)

func TestParseSidecar(t *testing.T) {
	data := []byte(`{"repo":"org/service","sha":"DEF456","author":"dev@example.com"}`)
	n, err := git.ParseSidecar(data)
	if err != nil {
		t.Fatal(err)
	}
	want := model.CommitID("def456")
	if n.ID != want {
		t.Fatalf("id %q want %q", n.ID, want)
	}
	if n.Type != model.NodeCommit {
		t.Fatalf("type %q", n.Type)
	}
	if n.Metadata["repo"] != "org/service" {
		t.Fatalf("repo metadata: %+v", n.Metadata)
	}
}

func TestParseSidecar_errors(t *testing.T) {
	_, err := git.ParseSidecar([]byte(`{"repo":"x"}`))
	if err == nil {
		t.Fatal("expected error for missing sha")
	}
	_, err = git.ParseSidecar([]byte(`{"sha":"abc"}`))
	if err == nil {
		t.Fatal("expected error for missing repo")
	}
}

func TestIsCommitSidecar(t *testing.T) {
	if !git.IsCommitSidecar([]byte(`{"repo":"r","sha":"a"}`)) {
		t.Fatal("expected true")
	}
	if git.IsCommitSidecar([]byte(`{"id":"ci-1","commit_sha":"x"}`)) {
		t.Fatal("expected false for build sidecar shape")
	}
}

func TestFromRepo(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not in PATH")
	}
	root, err := repoRoot()
	if err != nil {
		t.Skip(err)
	}
	n, err := git.FromRepo(root)
	if err != nil {
		t.Fatal(err)
	}
	if n.Type != model.NodeCommit {
		t.Fatalf("type %q", n.Type)
	}
	if n.Metadata["sha"] == "" {
		t.Fatal("missing sha metadata")
	}
}

func repoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", os.ErrNotExist
		}
		dir = parent
	}
}
