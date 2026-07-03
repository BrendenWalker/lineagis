package analyze

import (
	"strings"

	"github.com/BrendenWalker/lineagis/internal/core/graph"
	"github.com/BrendenWalker/lineagis/internal/core/model"
)

// Bridge links provenance and code subgraphs when both are present (FR-SA-014).
func Bridge(g *graph.Graph, modPath string) (nodes []model.Node, edges []model.Edge) {
	commits := g.ListByType(model.NodeCommit)
	if len(commits) == 1 {
		commitID := commits[0].ID
		for _, pkg := range g.ListByType(model.NodePackage) {
			importPath := strings.TrimPrefix(pkg.ID, "package:")
			if importPath == modPath || strings.HasPrefix(importPath, modPath+"/") {
				edges = append(edges, model.Edge{From: pkg.ID, To: commitID, Type: model.EdgeIntroducedBy})
			}
		}
	}
	modules := g.ListByType(model.NodeModule)
	moduleID := model.ModuleID(modPath)
	for _, m := range modules {
		if m.ID == moduleID {
			for _, art := range g.ListByType(model.NodeArtifact) {
				edges = append(edges, model.Edge{From: art.ID, To: moduleID, Type: model.EdgeContains})
			}
			break
		}
	}
	return nodes, edges
}
