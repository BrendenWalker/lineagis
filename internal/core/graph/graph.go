package graph

import (
	"fmt"
	"sort"

	"github.com/BrendenWalker/lineagis/internal/core/model"
)

// Graph is an in-memory directed graph of lineage nodes and edges.
type Graph struct {
	nodes map[string]model.Node
	edges []model.Edge
}

// New returns an empty graph.
func New() *Graph {
	return &Graph{
		nodes: make(map[string]model.Node),
	}
}

// AddNode inserts or replaces a node by canonical ID.
func (g *Graph) AddNode(n model.Node) error {
	if n.ID == "" {
		return fmt.Errorf("add node: empty id")
	}
	if n.Metadata == nil {
		n.Metadata = map[string]string{}
	}
	g.nodes[n.ID] = n
	return nil
}

// GetNode returns a node by ID.
func (g *Graph) GetNode(id string) (model.Node, bool) {
	n, ok := g.nodes[id]
	return n, ok
}

// ListByType returns nodes of the given type sorted by ID.
func (g *Graph) ListByType(t model.NodeType) []model.Node {
	var out []model.Node
	for _, n := range g.nodes {
		if n.Type == t {
			out = append(out, n)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// Edges returns a copy of all edges sorted deterministically.
func (g *Graph) Edges() []model.Edge {
	out := append([]model.Edge(nil), g.edges...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].From != out[j].From {
			return out[i].From < out[j].From
		}
		if out[i].To != out[j].To {
			return out[i].To < out[j].To
		}
		return out[i].Type < out[j].Type
	})
	return out
}

// Neighbors returns nodes reachable from id via outgoing edges of the given types.
func (g *Graph) Neighbors(from string, edgeTypes []model.EdgeType) []model.Node {
	allowed := make(map[model.EdgeType]struct{}, len(edgeTypes))
	for _, et := range edgeTypes {
		allowed[et] = struct{}{}
	}
	seen := map[string]struct{}{}
	var ids []string
	for _, e := range g.edges {
		if e.From != from {
			continue
		}
		if _, ok := allowed[e.Type]; !ok && len(edgeTypes) > 0 {
			continue
		}
		if _, dup := seen[e.To]; dup {
			continue
		}
		seen[e.To] = struct{}{}
		ids = append(ids, e.To)
	}
	sort.Strings(ids)
	var out []model.Node
	for _, id := range ids {
		if n, ok := g.nodes[id]; ok {
			out = append(out, n)
		}
	}
	return out
}

// IncomingNeighbors returns nodes with edges pointing to id (reverse direction).
func (g *Graph) IncomingNeighbors(to string, edgeTypes []model.EdgeType) []model.Node {
	allowed := make(map[model.EdgeType]struct{}, len(edgeTypes))
	for _, et := range edgeTypes {
		allowed[et] = struct{}{}
	}
	seen := map[string]struct{}{}
	var ids []string
	for _, e := range g.edges {
		if e.To != to {
			continue
		}
		if _, ok := allowed[e.Type]; !ok && len(edgeTypes) > 0 {
			continue
		}
		if _, dup := seen[e.From]; dup {
			continue
		}
		seen[e.From] = struct{}{}
		ids = append(ids, e.From)
	}
	sort.Strings(ids)
	var out []model.Node
	for _, id := range ids {
		if n, ok := g.nodes[id]; ok {
			out = append(out, n)
		}
	}
	return out
}

// OutgoingEdges returns edges from id, optionally filtered by type.
func (g *Graph) OutgoingEdges(from string, edgeTypes []model.EdgeType) []model.Edge {
	allowed := make(map[model.EdgeType]struct{}, len(edgeTypes))
	for _, et := range edgeTypes {
		allowed[et] = struct{}{}
	}
	var out []model.Edge
	for _, e := range g.edges {
		if e.From != from {
			continue
		}
		if len(edgeTypes) > 0 {
			if _, ok := allowed[e.Type]; !ok {
				continue
			}
		}
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].To != out[j].To {
			return out[i].To < out[j].To
		}
		return out[i].Type < out[j].Type
	})
	return out
}

// AddEdge appends a directed edge; rejects provenance cycles.
func (g *Graph) AddEdge(from, to string, edgeType model.EdgeType) error {
	if _, ok := g.nodes[from]; !ok {
		return fmt.Errorf("add edge %s -> %s: unknown from node %q", from, to, from)
	}
	if _, ok := g.nodes[to]; !ok {
		return fmt.Errorf("add edge %s -> %s: unknown to node %q", from, to, to)
	}
	if model.IsProvenanceEdge(edgeType) && g.wouldCreateProvenanceCycle(from, to) {
		return fmt.Errorf("add edge %s -> %s: would create cycle", from, to)
	}
	g.edges = append(g.edges, model.Edge{From: from, To: to, Type: edgeType})
	return nil
}

func (g *Graph) wouldCreateProvenanceCycle(from, to string) bool {
	// Adding from->to creates a cycle if there is a provenance path to->...->from.
	visited := map[string]struct{}{}
	var dfs func(string) bool
	dfs = func(cur string) bool {
		if cur == from {
			return true
		}
		if _, seen := visited[cur]; seen {
			return false
		}
		visited[cur] = struct{}{}
		for _, e := range g.edges {
			if !model.IsProvenanceEdge(e.Type) {
				continue
			}
			if e.From != cur {
				continue
			}
			if dfs(e.To) {
				return true
			}
		}
		return false
	}
	return dfs(to)
}

// Export returns a deterministic snapshot of the graph.
func (g *Graph) Export() model.GraphSnapshot {
	nodes := make([]model.Node, 0, len(g.nodes))
	var hasProvenance, hasCode bool
	for _, n := range g.nodes {
		nodes = append(nodes, n)
		if model.IsProvenanceNodeType(n.Type) {
			hasProvenance = true
		}
		if model.IsCodeNodeType(n.Type) {
			hasCode = true
		}
	}
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].ID < nodes[j].ID })

	snap := model.GraphSnapshot{
		Nodes: nodes,
		Edges: g.Edges(),
	}
	if hasCode {
		snap.SchemaVersion = model.SchemaGraphV2
		if hasProvenance {
			snap.Domains = []string{model.DomainProvenance, model.DomainCode}
		} else {
			snap.Domains = []string{model.DomainCode}
		}
	} else {
		snap.SchemaVersion = model.SchemaGraphV1
	}
	return snap
}

// LoadSnapshot replaces graph contents from a snapshot.
func (g *Graph) LoadSnapshot(snap model.GraphSnapshot) error {
	g.nodes = make(map[string]model.Node)
	g.edges = nil
	for _, n := range snap.Nodes {
		if err := g.AddNode(n); err != nil {
			return err
		}
	}
	for _, e := range snap.Edges {
		if err := g.AddEdge(e.From, e.To, e.Type); err != nil {
			return err
		}
	}
	return nil
}

// NodeCount returns the number of nodes.
func (g *Graph) NodeCount() int { return len(g.nodes) }

// EdgeCount returns the number of edges.
func (g *Graph) EdgeCount() int { return len(g.edges) }
