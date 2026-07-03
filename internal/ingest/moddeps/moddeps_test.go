package moddeps_test

import (
	"path/filepath"
	"testing"

	"github.com/BrendenWalker/lineagis/internal/analyze"
	"github.com/BrendenWalker/lineagis/internal/core/graph"
	"github.com/BrendenWalker/lineagis/internal/core/model"
	"github.com/BrendenWalker/lineagis/internal/ingest/moddeps"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	return filepath.Join("..", "..", "..")
}

func TestDirectRequires_lineagis(t *testing.T) {
	reqs, err := moddeps.DirectRequires(repoRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(reqs) < 2 {
		t.Fatalf("expected direct requires, got %+v", reqs)
	}
	foundTools := false
	for _, r := range reqs {
		if r.Path == "golang.org/x/tools" {
			foundTools = true
		}
	}
	if !foundTools {
		t.Fatalf("missing golang.org/x/tools in %+v", reqs)
	}
}

func TestIngest_externalModuleMetadata(t *testing.T) {
	root := repoRoot(t)
	g := graph.New()
	if err := analyze.Path(g, root); err != nil {
		t.Fatal(err)
	}
	toolsID := model.ModuleID("golang.org/x/tools")
	n, ok := g.GetNode(toolsID)
	if !ok {
		t.Fatalf("missing module node %s", toolsID)
	}
	if n.Metadata["external"] != "true" {
		t.Fatalf("expected external metadata: %+v", n.Metadata)
	}
	if n.Metadata["import_count"] == "" || n.Metadata["import_count"] == "0" {
		t.Fatalf("expected non-zero import_count, got %+v", n.Metadata)
	}
}
