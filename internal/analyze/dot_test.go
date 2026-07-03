package analyze_test

import (
	"strings"
	"testing"

	"github.com/BrendenWalker/lineagis/internal/analyze"
	"github.com/BrendenWalker/lineagis/internal/core/graph"
	"github.com/BrendenWalker/lineagis/internal/core/model"
)

func TestPackageImportDOT(t *testing.T) {
	g := graph.New()
	a := model.PackageID("example.com/a")
	b := model.PackageID("example.com/b")
	_ = g.AddNode(model.Node{ID: a, Type: model.NodePackage, Metadata: map[string]string{"name": "a"}})
	_ = g.AddNode(model.Node{ID: b, Type: model.NodePackage, Metadata: map[string]string{"name": "b"}})
	_ = g.AddEdge(a, b, model.EdgeImports)

	dot := analyze.PackageImportDOT(g)
	if !strings.Contains(dot, "digraph imports") {
		t.Fatalf("missing digraph header: %s", dot)
	}
	if !strings.Contains(dot, `"a" -> "b"`) {
		t.Fatalf("missing edge in dot: %s", dot)
	}
}
