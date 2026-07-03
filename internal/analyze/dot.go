package analyze

import (
	"fmt"
	"strings"

	"github.com/BrendenWalker/lineagis/internal/core/graph"
	"github.com/BrendenWalker/lineagis/internal/core/model"
)

// PackageImportDOT emits Graphviz DOT for package-level import edges.
func PackageImportDOT(g *graph.Graph) string {
	var b strings.Builder
	b.WriteString("digraph imports {\n")
	for _, e := range g.Edges() {
		if e.Type != model.EdgeImports {
			continue
		}
		from, ok1 := g.GetNode(e.From)
		to, ok2 := g.GetNode(e.To)
		if !ok1 || !ok2 || from.Type != model.NodePackage || to.Type != model.NodePackage {
			continue
		}
		fmt.Fprintf(&b, "  %q -> %q;\n", dotLabel(from), dotLabel(to))
	}
	b.WriteString("}\n")
	return b.String()
}

func dotLabel(n model.Node) string {
	if name, ok := n.Metadata["name"]; ok && name != "" {
		return name
	}
	const prefix = "package:"
	if strings.HasPrefix(n.ID, prefix) {
		return n.ID[len(prefix):]
	}
	return n.ID
}
