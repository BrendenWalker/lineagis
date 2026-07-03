package report

import (
	"sort"
	"strings"

	"github.com/BrendenWalker/lineagis/internal/core/graph"
	"github.com/BrendenWalker/lineagis/internal/core/model"
)

// Metrics summarizes code subgraph statistics (FR-SA-050).
type Metrics struct {
	PackageCount int
	SymbolCount  int
	ImportEdges  int
	ModuleCount  int
	Cycles       []ImportCycle
}

// ImportCycle is a sorted list of package IDs forming an import cycle.
type ImportCycle struct {
	Packages []string
}

// ComputeMetrics derives architecture metrics from g.
func ComputeMetrics(g *graph.Graph) Metrics {
	m := Metrics{
		PackageCount: len(g.ListByType(model.NodePackage)),
		SymbolCount:  len(g.ListByType(model.NodeSymbol)),
		ModuleCount:  len(g.ListByType(model.NodeModule)),
	}
	for _, e := range g.Edges() {
		if e.Type == model.EdgeImports {
			m.ImportEdges++
		}
	}
	m.Cycles = detectImportCycles(g)
	return m
}

func detectImportCycles(g *graph.Graph) []ImportCycle {
	adj := map[string][]string{}
	for _, e := range g.Edges() {
		if e.Type != model.EdgeImports {
			continue
		}
		adj[e.From] = append(adj[e.From], e.To)
	}
	for id := range adj {
		sort.Strings(adj[id])
	}

	index := 0
	stack := []string{}
	indices := map[string]int{}
	lowlink := map[string]int{}
	onStack := map[string]bool{}
	var cycles []ImportCycle

	var strongConnect func(string)
	strongConnect = func(v string) {
		indices[v] = index
		lowlink[v] = index
		index++
		stack = append(stack, v)
		onStack[v] = true

		for _, w := range adj[v] {
			if _, ok := indices[w]; !ok {
				strongConnect(w)
				if lowlink[w] < lowlink[v] {
					lowlink[v] = lowlink[w]
				}
			} else if onStack[w] && indices[w] < lowlink[v] {
				lowlink[v] = indices[w]
			}
		}

		if lowlink[v] == indices[v] {
			var comp []string
			for {
				w := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				onStack[w] = false
				comp = append(comp, w)
				if w == v {
					break
				}
			}
			if len(comp) > 1 {
				sort.Strings(comp)
				cycles = append(cycles, ImportCycle{Packages: comp})
			}
		}
	}

	var nodes []string
	for id := range adj {
		nodes = append(nodes, id)
	}
	sort.Strings(nodes)
	for _, id := range nodes {
		if _, ok := indices[id]; !ok {
			strongConnect(id)
		}
	}
	sort.Slice(cycles, func(i, j int) bool {
		return strings.Join(cycles[i].Packages, ",") < strings.Join(cycles[j].Packages, ",")
	})
	return cycles
}
