package report

import (
	"fmt"
	"strings"

	"github.com/BrendenWalker/lineagis/internal/core/graph"
	"github.com/BrendenWalker/lineagis/internal/core/model"
)

// ArchitectureMarkdown renders a deterministic architecture summary (FR-SA-050).
func ArchitectureMarkdown(g *graph.Graph) string {
	m := ComputeMetrics(g)
	var b strings.Builder
	b.WriteString("# Architecture Report\n\n")
	b.WriteString("## Summary\n\n")
	fmt.Fprintf(&b, "- Packages: %d\n", m.PackageCount)
	fmt.Fprintf(&b, "- Symbols: %d\n", m.SymbolCount)
	fmt.Fprintf(&b, "- Modules: %d\n", m.ModuleCount)
	fmt.Fprintf(&b, "- Import edges: %d\n", m.ImportEdges)
	fmt.Fprintf(&b, "- Import cycles: %d\n\n", len(m.Cycles))

	b.WriteString("## Packages\n\n")
	for _, n := range g.ListByType(model.NodePackage) {
		name := n.Metadata["name"]
		if name == "" {
			name = strings.TrimPrefix(n.ID, "package:")
		}
		dir := n.Metadata["dir"]
		if dir != "" {
			fmt.Fprintf(&b, "- `%s` (`%s`)\n", name, dir)
		} else {
			fmt.Fprintf(&b, "- `%s`\n", name)
		}
	}

	if len(m.Cycles) > 0 {
		b.WriteString("\n## Import cycles\n\n")
		for _, c := range m.Cycles {
			b.WriteString("- ")
			for i, p := range c.Packages {
				if i > 0 {
					b.WriteString(" → ")
				}
				b.WriteString(strings.TrimPrefix(p, "package:"))
			}
			b.WriteString("\n")
		}
	}
	return b.String()
}
