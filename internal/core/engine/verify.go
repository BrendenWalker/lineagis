package engine

import (
	"fmt"
	"sort"

	"github.com/BrendenWalker/lineagis/internal/core/graph"
	"github.com/BrendenWalker/lineagis/internal/core/model"
)

// VerifyGraph scans the graph for incomplete lineage (FR-LIN-030, FR-LIN-031).
func VerifyGraph(g *graph.Graph) Verification {
	var findings []string
	for _, n := range g.ListByType(model.NodeArtifact) {
		if !hasEdgeFrom(g, n.ID, model.EdgeProducedBy) {
			findings = append(findings, fmt.Sprintf("artifact %s has no produced_by edge to build", n.ID))
		}
	}
	for _, n := range g.ListByType(model.NodeBuild) {
		if !hasEdgeFrom(g, n.ID, model.EdgeBuiltFrom) {
			findings = append(findings, fmt.Sprintf("build %s has no built_from edge to commit", n.ID))
		}
	}
	// Should: dependency orphans with no incoming depends_on from an artifact
	for _, n := range g.ListByType(model.NodeDependency) {
		if !hasEdgeTo(g, n.ID, model.EdgeDependsOn) {
			findings = append(findings, fmt.Sprintf("dependency %s is not linked from any artifact", n.ID))
		}
	}
	sort.Strings(findings)
	return Verification{Complete: len(findings) == 0, Findings: findings}
}

func hasEdgeFrom(g *graph.Graph, from string, et model.EdgeType) bool {
	for _, e := range g.Edges() {
		if e.From == from && e.Type == et {
			return true
		}
	}
	return false
}

func hasEdgeTo(g *graph.Graph, to string, et model.EdgeType) bool {
	for _, e := range g.Edges() {
		if e.To == to && e.Type == et {
			return true
		}
	}
	return false
}
