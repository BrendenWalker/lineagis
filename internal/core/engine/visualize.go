package engine

import (
	"fmt"
	"strings"

	"github.com/BrendenWalker/lineagis/internal/core/graph"
)

// ToDOT emits Graphviz DOT for the upstream subgraph from rootID.
func ToDOT(g *graph.Graph, rootID string) string {
	nodes, edges := Upstream(g, rootID)
	var b strings.Builder
	b.WriteString("digraph lineage {\n  rankdir=LR;\n")
	seen := map[string]struct{}{}
	for _, n := range nodes {
		if _, ok := seen[n.ID]; ok {
			continue
		}
		seen[n.ID] = struct{}{}
		label := string(n.Type)
		if name := n.Metadata["name"]; name != "" {
			label = name
		}
		fmt.Fprintf(&b, "  %q [label=%q];\n", dotID(n.ID), label)
	}
	for _, e := range edges {
		fmt.Fprintf(&b, "  %q -> %q [label=%q];\n", dotID(e.From), dotID(e.To), e.Type)
	}
	b.WriteString("}\n")
	return b.String()
}

func dotID(id string) string {
	return strings.ReplaceAll(id, ":", "_")
}
