package dedupe

import (
	"github.com/BrendenWalker/lineagis/internal/core/graph"
	"github.com/BrendenWalker/lineagis/internal/core/model"
)

// Apply merges nodes and edges into g, deduplicating by canonical node ID and edge triple.
func Apply(g *graph.Graph, nodes []model.Node, edges []model.Edge) error {
	for _, n := range nodes {
		if err := g.AddNode(n); err != nil {
			return err
		}
	}
	for _, e := range edges {
		if edgeExists(g, e.From, e.To, e.Type) {
			continue
		}
		if err := g.AddEdge(e.From, e.To, e.Type); err != nil {
			return err
		}
	}
	return nil
}

func edgeExists(g *graph.Graph, from, to string, t model.EdgeType) bool {
	for _, e := range g.Edges() {
		if e.From == from && e.To == to && e.Type == t {
			return true
		}
	}
	return false
}
