package goingest_test

import (
	"path/filepath"
	"testing"

	"github.com/BrendenWalker/lineagis/internal/core/graph"
	"github.com/BrendenWalker/lineagis/internal/core/model"
	"github.com/BrendenWalker/lineagis/internal/analyze"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	return filepath.Join("..", "..", "..")
}

func TestAnalyzeMiniModule(t *testing.T) {
	g := graph.New()
	root := repoRoot(t)
	path := filepath.Join(root, "examples", "self-analysis")
	if err := analyze.Path(g, path); err != nil {
		t.Fatal(err)
	}
	app := model.PackageID("github.com/BrendenWalker/lineagis/examples/self-analysis/app")
	lib := model.PackageID("github.com/BrendenWalker/lineagis/examples/self-analysis/lib")
	assertNode(t, g, app)
	assertNode(t, g, lib)
	assertEdge(t, g, app, lib, model.EdgeImports)
}

func TestAnalyzeLineagisRepo(t *testing.T) {
	g := graph.New()
	if err := analyze.Path(g, repoRoot(t)); err != nil {
		t.Fatal(err)
	}
	cmd := model.PackageID("github.com/BrendenWalker/lineagis/cmd/lineagis")
	query := model.PackageID("github.com/BrendenWalker/lineagis/internal/core/query")
	graphPkg := model.PackageID("github.com/BrendenWalker/lineagis/internal/core/graph")

	assertNode(t, g, cmd)
	assertNode(t, g, graphPkg)
	assertEdge(t, g, cmd, query, model.EdgeImports)
	sym := model.SymbolID("github.com/BrendenWalker/lineagis/internal/core/graph", "New")
	assertNode(t, g, sym)

	snap := g.Export()
	if snap.SchemaVersion != model.SchemaGraphV2 {
		t.Fatalf("schema %q want %q", snap.SchemaVersion, model.SchemaGraphV2)
	}
}

func TestAnalyzeDedupe(t *testing.T) {
	g := graph.New()
	path := filepath.Join(repoRoot(t), "examples", "self-analysis")
	if err := analyze.Path(g, path); err != nil {
		t.Fatal(err)
	}
	n1, e1 := g.NodeCount(), g.EdgeCount()
	if err := analyze.Path(g, path); err != nil {
		t.Fatal(err)
	}
	if g.NodeCount() != n1 || g.EdgeCount() != e1 {
		t.Fatalf("dedupe failed nodes %d->%d edges %d->%d", n1, g.NodeCount(), e1, g.EdgeCount())
	}
}

func assertNode(t *testing.T, g *graph.Graph, id string) {
	t.Helper()
	if _, ok := g.GetNode(id); !ok {
		t.Fatalf("missing node %s", id)
	}
}

func assertEdge(t *testing.T, g *graph.Graph, from, to string, typ model.EdgeType) {
	t.Helper()
	if !hasEdge(g, from, to, typ) {
		t.Fatalf("missing edge %s -[%s]-> %s", from, typ, to)
	}
}

func hasEdge(g *graph.Graph, from, to string, typ model.EdgeType) bool {
	for _, e := range g.Edges() {
		if e.From == from && e.To == to && e.Type == typ {
			return true
		}
	}
	return false
}
