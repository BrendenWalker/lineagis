package memory_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/BrendenWalker/lineagis/internal/core/model"
	"github.com/BrendenWalker/lineagis/internal/storage/memory"
)

func TestSaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "graph.json")
	s := memory.NewStore()
	_ = s.Graph().AddNode(model.Node{ID: model.ArtifactID("abc"), Type: model.NodeArtifact})
	if err := s.Save(path); err != nil {
		t.Fatal(err)
	}
	s2 := memory.NewStore()
	if err := s2.Load(path); err != nil {
		t.Fatal(err)
	}
	if _, ok := s2.Graph().GetNode(model.ArtifactID("abc")); !ok {
		t.Fatal("reload missing node")
	}
}

func TestLoadMissingFile(t *testing.T) {
	s := memory.NewStore()
	if err := s.Load(filepath.Join(t.TempDir(), "missing.json")); err != nil {
		t.Fatal(err)
	}
	if s.Graph().NodeCount() != 0 {
		t.Fatal("expected empty graph")
	}
	_ = os.Getenv("LINEAGIS_GRAPH_FILE")
}
