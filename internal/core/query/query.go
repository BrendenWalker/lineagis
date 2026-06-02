package query

import (
	"fmt"
	"strings"

	"github.com/BrendenWalker/lineagis/internal/core/engine"
	"github.com/BrendenWalker/lineagis/internal/core/graph"
	"github.com/BrendenWalker/lineagis/internal/core/model"
)

// Trace resolves ref and runs upstream traversal.
func Trace(g *graph.Graph, ref string) (engine.TraceResult, error) {
	id, err := model.ParseRef(ref)
	if err != nil {
		return engine.TraceResult{}, err
	}
	return engine.Trace(g, id)
}

// Why resolves ref and explains lineage.
func Why(g *graph.Graph, ref string) (engine.WhyResult, error) {
	id, err := model.ParseRef(ref)
	if err != nil {
		return engine.WhyResult{}, err
	}
	return engine.Why(g, id)
}

// SummaryTrace returns a human-readable trace summary.
func SummaryTrace(res engine.TraceResult) string {
	var commits []string
	for _, n := range res.Nodes {
		if n.Type == model.NodeCommit {
			commits = append(commits, n.ID)
		}
	}
	if len(commits) == 0 {
		return fmt.Sprintf("trace from %s: %d nodes, %d edges (no commits reached)", res.Root, len(res.Nodes), len(res.Edges))
	}
	return fmt.Sprintf("trace from %s → %s (%d nodes)", res.Root, strings.Join(commits, ", "), len(res.Nodes))
}

// SummaryWhy returns human-readable why output.
func SummaryWhy(res engine.WhyResult) string {
	if res.Complete {
		return res.Message
	}
	if res.Gap != "" {
		s := res.Gap
		if res.Remediation != "" {
			s += "\nRemediation: " + res.Remediation
		}
		return s
	}
	return res.Message
}
