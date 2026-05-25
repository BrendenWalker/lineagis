package publish

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCollectFiles_directory(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(dir, "sub")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "b.txt"), []byte("b"), 0o644); err != nil {
		t.Fatal(err)
	}

	layers, _, err := CollectFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(layers) != 2 {
		t.Fatalf("layers = %d, want 2", len(layers))
	}
}

func TestCollectFiles_singleFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "only.txt")
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	layers, _, err := CollectFiles(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(layers) != 1 || layers[0].Path != "only.txt" {
		t.Fatalf("layers = %+v", layers)
	}
}

func TestCollectFiles_emptyDir(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	_, _, err := CollectFiles(dir)
	if err == nil {
		t.Fatal("expected error for empty directory")
	}
}
