package report

import (
	"fmt"
	"sort"
	"strings"

	"github.com/BrendenWalker/lineagis/internal/core/graph"
	"github.com/BrendenWalker/lineagis/internal/core/model"
)

// ExplainResult summarizes an external module dependency (FR-SA-032).
type ExplainResult struct {
	SchemaVersion string   `json:"schema_version"`
	Module        string   `json:"module"`
	Version       string   `json:"version,omitempty"`
	ImportCount   string   `json:"import_count,omitempty"`
	Importers     []string `json:"importers"`
	Message       string   `json:"message"`
}

const SchemaExplainV1 = "lineage-explain/v1"

// ExplainDependency summarizes purpose, importers, and removal impact.
func ExplainDependency(g *graph.Graph, modulePath string) (ExplainResult, error) {
	moduleID := model.ModuleID(modulePath)
	n, ok := g.GetNode(moduleID)
	if !ok {
		return ExplainResult{}, fmt.Errorf("module not found: %s (run lineagis analyze first)", modulePath)
	}
	importers := importersOfModule(g, moduleID)
	res := ExplainResult{
		SchemaVersion: SchemaExplainV1,
		Module:        modulePath,
		Version:       n.Metadata["version"],
		ImportCount:   n.Metadata["import_count"],
		Importers:     importers,
	}
	if len(importers) == 0 {
		res.Message = fmt.Sprintf("Module %s has no recorded in-repo importers.", modulePath)
	} else {
		res.Message = fmt.Sprintf("Module %s is imported by %d package(s). Removing it would affect those packages.", modulePath, len(importers))
	}
	return res, nil
}

func importersOfModule(g *graph.Graph, moduleID string) []string {
	packages := map[string]struct{}{}
	for _, e := range g.Edges() {
		if e.Type == model.EdgeContains && e.From == moduleID {
			packages[e.To] = struct{}{}
		}
	}
	seen := map[string]struct{}{}
	for pkgID := range packages {
		for _, n := range g.IncomingNeighbors(pkgID, []model.EdgeType{model.EdgeImports}) {
			seen[n.ID] = struct{}{}
		}
	}
	var out []string
	for id := range seen {
		out = append(out, strings.TrimPrefix(id, "package:"))
	}
	sort.Strings(out)
	return out
}

// ExplainMarkdown formats ExplainResult as text.
func ExplainMarkdown(res ExplainResult) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Module: %s\n", res.Module)
	if res.Version != "" {
		fmt.Fprintf(&b, "Version: %s\n", res.Version)
	}
	if res.ImportCount != "" {
		fmt.Fprintf(&b, "Direct require import count: %s\n", res.ImportCount)
	}
	b.WriteString("\n")
	b.WriteString(res.Message)
	b.WriteString("\n\nImporters:\n")
	if len(res.Importers) == 0 {
		b.WriteString("  (none)\n")
	} else {
		for _, p := range res.Importers {
			fmt.Fprintf(&b, "  - %s\n", p)
		}
	}
	return b.String()
}
