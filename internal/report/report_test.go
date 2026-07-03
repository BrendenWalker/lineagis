package report

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/BrendenWalker/lineagis/internal/analyze"
	"github.com/BrendenWalker/lineagis/internal/core/graph"
	"github.com/BrendenWalker/lineagis/internal/core/model"
)

func TestComputeMetrics_matchesGraph(t *testing.T) {
	root := repoRoot(t)
	g := graph.New()
	if err := analyze.Path(g, root); err != nil {
		t.Fatal(err)
	}
	m := ComputeMetrics(g)
	if m.PackageCount != len(g.ListByType(model.NodePackage)) {
		t.Fatalf("package count %d != graph %d", m.PackageCount, len(g.ListByType(model.NodePackage)))
	}
	if m.ImportEdges == 0 {
		t.Fatal("expected import edges")
	}
	md := ArchitectureMarkdown(g)
	if md == "" {
		t.Fatal("empty architecture markdown")
	}
}

func TestWriteTree_deterministic(t *testing.T) {
	root := repoRoot(t)
	g := graph.New()
	if err := analyze.Path(g, root); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := WriteTree(g, dir); err != nil {
		t.Fatal(err)
	}
	checks := []string{
		"architecture/overview.md",
		"reports/dependency-report.md",
		"reports/orphan-packages.md",
		"diagrams/imports.dot",
		"lineage/lineage.json",
	}
	for _, rel := range checks {
		if _, err := os.Stat(filepath.Join(dir, rel)); err != nil {
			t.Fatalf("missing %s: %v", rel, err)
		}
	}
	dir2 := t.TempDir()
	if err := WriteTree(g, dir2); err != nil {
		t.Fatal(err)
	}
	a, _ := os.ReadFile(filepath.Join(dir, "architecture/overview.md"))
	b, _ := os.ReadFile(filepath.Join(dir2, "architecture/overview.md"))
	if string(a) != string(b) {
		t.Fatal("architecture report not deterministic")
	}
}

func TestImpactPackage_graph(t *testing.T) {
	root := repoRoot(t)
	g := graph.New()
	if err := analyze.Path(g, root); err != nil {
		t.Fatal(err)
	}
	graphPkg := model.PackageID("github.com/BrendenWalker/lineagis/internal/core/graph")
	res, err := ImpactPackage(g, graphPkg)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Packages) == 0 {
		t.Fatal("expected downstream packages for internal/core/graph")
	}
	foundTest := false
	for _, f := range res.Tests {
		if f == model.FileID("internal/core/graph/graph_test.go") {
			foundTest = true
		}
	}
	if !foundTest {
		t.Fatalf("expected graph_test.go in impact tests, got %+v", res.Tests)
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found")
		}
		dir = parent
	}
}
