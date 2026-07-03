package analyze

import (
	"fmt"

	"github.com/BrendenWalker/lineagis/internal/core/graph"
	"github.com/BrendenWalker/lineagis/internal/core/model"
	"github.com/BrendenWalker/lineagis/internal/ingest/docs"
	goingest "github.com/BrendenWalker/lineagis/internal/ingest/go"
	"github.com/BrendenWalker/lineagis/internal/ingest/workflow"
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

	moduleRoot, err := goingest.ModuleRoot(path)
	if err != nil {
		return err
	}
	modPath, err := goingest.ModulePath(moduleRoot)
	if err != nil {
		return err
	}

	docRes, err := docs.Ingest(moduleRoot, modPath, packageIDs(g))
	if err != nil {
		return fmt.Errorf("analyze docs: %w", err)
	}
	if err := dedupe.Apply(g, docRes.Nodes, docRes.Edges); err != nil {
		return fmt.Errorf("analyze docs merge: %w", err)
	}

	wfRes, err := workflow.Ingest(moduleRoot)
	if err != nil {
		return fmt.Errorf("analyze workflows: %w", err)
	}
	if err := dedupe.Apply(g, wfRes.Nodes, wfRes.Edges); err != nil {
		return fmt.Errorf("analyze workflows merge: %w", err)
	}

	_, bridgeEdges := Bridge(g, modPath)
	if err := dedupe.Apply(g, nil, bridgeEdges); err != nil {
		return fmt.Errorf("analyze bridge: %w", err)
	}
	return nil
}

func packageIDs(g *graph.Graph) []string {
	var ids []string
	for _, n := range g.ListByType(model.NodePackage) {
		ids = append(ids, n.ID)
	}
	return ids
}
