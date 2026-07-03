package analyze

import (
	"fmt"

	"github.com/BrendenWalker/lineagis/internal/core/graph"
	goingest "github.com/BrendenWalker/lineagis/internal/ingest/go"
	"github.com/BrendenWalker/lineagis/internal/normalize/dedupe"
)

// Path analyzes a Go module tree at path and merges results into g.
func Path(g *graph.Graph, path string) error {
	res, err := goingest.Analyze(path)
	if err != nil {
		return fmt.Errorf("analyze %s: %w", path, err)
	}
	if err := dedupe.Apply(g, res.Nodes, res.Edges); err != nil {
		return fmt.Errorf("analyze merge: %w", err)
	}
	return nil
}
