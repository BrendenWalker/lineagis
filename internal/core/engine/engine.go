package engine

import (
	"fmt"
	"sort"

	"github.com/BrendenWalker/lineagis/internal/core/graph"
	"github.com/BrendenWalker/lineagis/internal/core/model"
)

// TraceResult is upstream lineage from a root node.
type TraceResult struct {
	SchemaVersion string       `json:"schema_version"`
	Root          string       `json:"root"`
	Nodes         []model.Node `json:"nodes"`
	Edges         []model.Edge `json:"edges"`
	Verification  Verification `json:"verification"`
}

const SchemaTraceV1 = "lineage-trace/v1"

// Verification summarizes lineage completeness for the traced subgraph.
type Verification struct {
	Complete bool     `json:"complete"`
	Findings []string `json:"findings"`
}

// WhyResult explains a path or missing links.
type WhyResult struct {
	SchemaVersion string       `json:"schema_version"`
	Root          string       `json:"root"`
	Complete      bool         `json:"complete"`
	Path          []string     `json:"path,omitempty"`
	Nodes         []model.Node `json:"nodes"`
	Edges         []model.Edge `json:"edges"`
	Message       string       `json:"message,omitempty"`
	Gap           string       `json:"gap,omitempty"`
	Remediation   string       `json:"remediation,omitempty"`
	Verification  Verification `json:"verification"`
}

// Upstream walks against edge direction toward commits (artifact→build→commit stored as forward edges).
func Upstream(g *graph.Graph, rootID string) (nodes []model.Node, edges []model.Edge) {
	visited := map[string]struct{}{}
	edgeSeen := map[string]struct{}{}
	var order []string
	var walk func(string)
	walk = func(id string) {
		if _, ok := visited[id]; ok {
			return
		}
		visited[id] = struct{}{}
		order = append(order, id)
		for _, e := range g.Edges() {
			var next string
			switch e.Type {
			case model.EdgeProducedBy:
				if e.From == id {
					next = e.To
				}
			case model.EdgeBuiltFrom:
				if e.From == id {
					next = e.To
				}
			case model.EdgeDependsOn:
				if e.To == id {
					next = e.From
				}
			}
			if next == "" {
				continue
			}
			key := e.From + "|" + e.To + "|" + string(e.Type)
			if _, dup := edgeSeen[key]; !dup {
				edgeSeen[key] = struct{}{}
				edges = append(edges, e)
			}
			walk(next)
		}
	}
	walk(rootID)
	sort.Strings(order)
	for _, id := range order {
		if n, ok := g.GetNode(id); ok {
			nodes = append(nodes, n)
		}
	}
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].From != edges[j].From {
			return edges[i].From < edges[j].From
		}
		if edges[i].To != edges[j].To {
			return edges[i].To < edges[j].To
		}
		return edges[i].Type < edges[j].Type
	})
	return nodes, edges
}

// Trace performs upstream traversal from rootID.
func Trace(g *graph.Graph, rootID string) (TraceResult, error) {
	if _, ok := g.GetNode(rootID); !ok {
		return TraceResult{}, fmt.Errorf("node %q not found", rootID)
	}
	nodes, edges := Upstream(g, rootID)
	v := VerifyGraph(g)
	return TraceResult{
		SchemaVersion: SchemaTraceV1,
		Root:          rootID,
		Nodes:         nodes,
		Edges:         edges,
		Verification:  v,
	}, nil
}

// Why returns shortest provenance path to a commit or reports a gap.
func Why(g *graph.Graph, rootID string) (WhyResult, error) {
	if _, ok := g.GetNode(rootID); !ok {
		return WhyResult{}, fmt.Errorf("node %q not found", rootID)
	}
	v := VerifyGraph(g)

	// BFS upstream on provenance edges only
	type state struct {
		id   string
		path []string
	}
	queue := []state{{id: rootID, path: []string{rootID}}}
	visited := map[string]struct{}{rootID: {}}

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		n, _ := g.GetNode(cur.id)
		if n.Type == model.NodeCommit {
			nodes, edges := collectPath(g, cur.path)
			return WhyResult{
				SchemaVersion: SchemaTraceV1,
				Root:          rootID,
				Complete:      true,
				Path:          cur.path,
				Nodes:         nodes,
				Edges:         edges,
				Message:       formatPath(cur.path),
				Verification:  v,
			}, nil
		}
		next := upstreamProvenanceStep(g, cur.id)
		if len(next) == 0 {
			gap, remediation := diagnoseGap(g, cur.id, n.Type)
			nodes, edges := collectPath(g, cur.path)
			return WhyResult{
				SchemaVersion: SchemaTraceV1,
				Root:          rootID,
				Complete:      false,
				Path:          cur.path,
				Nodes:         nodes,
				Edges:         edges,
				Message:       gap,
				Gap:           gap,
				Remediation:   remediation,
				Verification:  v,
			}, nil
		}
		for _, nid := range next {
			if _, seen := visited[nid]; seen {
				continue
			}
			visited[nid] = struct{}{}
			p := append(append([]string{}, cur.path...), nid)
			queue = append(queue, state{id: nid, path: p})
		}
	}

	gap, remediation := diagnoseGap(g, rootID, model.NodeArtifact)
	nodes, edges := collectPath(g, []string{rootID})
	return WhyResult{
		SchemaVersion: SchemaTraceV1,
		Root:          rootID,
		Complete:      false,
		Path:          []string{rootID},
		Nodes:         nodes,
		Edges:         edges,
		Gap:           gap,
		Remediation:   remediation,
		Verification:  v,
	}, nil
}

func upstreamProvenanceStep(g *graph.Graph, id string) []string {
	var next []string
	n, _ := g.GetNode(id)
	for _, e := range g.Edges() {
		switch e.Type {
		case model.EdgeProducedBy:
			if e.From == id {
				next = append(next, e.To)
			}
		case model.EdgeBuiltFrom:
			if e.From == id && n.Type == model.NodeBuild {
				next = append(next, e.To)
			}
		}
	}
	sort.Strings(next)
	return next
}

func diagnoseGap(g *graph.Graph, id string, t model.NodeType) (gap, remediation string) {
	switch t {
	case model.NodeArtifact:
		if !hasOutgoing(g, id, model.EdgeProducedBy) {
			return fmt.Sprintf("%s — missing produced_by → build", id),
				"ingest build metadata linking this artifact to a build"
		}
		builds := upstreamProvenanceStep(g, id)
		if len(builds) > 0 {
			return diagnoseGap(g, builds[0], model.NodeBuild)
		}
	case model.NodeBuild:
		if !hasOutgoing(g, id, model.EdgeBuiltFrom) {
			return fmt.Sprintf("%s — missing built_from → commit", id),
				"ingest build metadata or git commit sidecar linking " + id
		}
	}
	return fmt.Sprintf("%s — incomplete provenance chain", id),
		"ingest commit and build sidecars to complete lineage"
}

func hasOutgoing(g *graph.Graph, id string, et model.EdgeType) bool {
	for _, e := range g.Edges() {
		if e.From == id && e.Type == et {
			return true
		}
	}
	return false
}

func collectPath(g *graph.Graph, path []string) ([]model.Node, []model.Edge) {
	seen := map[string]struct{}{}
	var nodes []model.Node
	var edges []model.Edge
	for i, id := range path {
		if n, ok := g.GetNode(id); ok {
			nodes = append(nodes, n)
		}
		if i == 0 {
			continue
		}
		prev := path[i-1]
		for _, e := range g.Edges() {
			if e.From == prev && e.To == id {
				key := e.From + e.To + string(e.Type)
				if _, dup := seen[key]; !dup {
					seen[key] = struct{}{}
					edges = append(edges, e)
				}
			}
			if e.To == id && e.From == prev {
				key := e.From + e.To + string(e.Type)
				if _, dup := seen[key]; !dup {
					seen[key] = struct{}{}
					edges = append(edges, e)
				}
			}
		}
	}
	return nodes, edges
}

func formatPath(path []string) string {
	if len(path) == 0 {
		return ""
	}
	var b string
	for i, id := range path {
		if i > 0 {
			b += " → "
		}
		b += id
	}
	return b
}
