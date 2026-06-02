package dedupe

import (
	"github.com/BrendenWalker/lineagis/internal/core/graph"
	"github.com/BrendenWalker/lineagis/internal/core/model"
)

// Apply merges nodes and edges into g, deduplicating by canonical node ID.
func Apply(g *graph.Graph, nodes []model.Node, edges []model.Edge) error {
	for _, n := range nodes {
		if err := g.AddNode(n); err != nil {
			return err
		}
	}
	for _, e := range edges {
		if err := g.AddEdge(e.From, e.To, e.Type); err != nil {
			return err
		}
	}
	return nil
}
