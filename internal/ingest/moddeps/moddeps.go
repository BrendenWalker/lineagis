package moddeps

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/mod/modfile"

	"github.com/BrendenWalker/lineagis/internal/core/graph"
	"github.com/BrendenWalker/lineagis/internal/core/model"
)

// Require is a direct go.mod dependency.
type Require struct {
	Path    string
	Version string
}

// Result holds nodes and edges from go.mod dependency ingest.
type Result struct {
	Nodes []model.Node
	Edges []model.Edge
}

// Ingest emits module nodes for direct external requires with import metadata (FR-SA-020, FR-SA-021).
func Ingest(g *graph.Graph, moduleRoot, modPath string) (Result, error) {
	requires, err := DirectRequires(moduleRoot)
	if err != nil {
		return Result{}, err
	}
	var res Result
	for _, req := range requires {
		if req.Path == modPath {
			continue
		}
		id := model.ModuleID(req.Path)
		importers := countImporters(g, modPath, req.Path)
		meta := map[string]string{
			"path":         req.Path,
			"version":      req.Version,
			"external":     "true",
			"import_count": strconv.Itoa(importers),
		}
		res.Nodes = append(res.Nodes, model.Node{ID: id, Type: model.NodeModule, Metadata: meta})

		for _, pkg := range g.ListByType(model.NodePackage) {
			pkgPath := strings.TrimPrefix(pkg.ID, "package:")
			if belongsToModule(pkgPath, req.Path) && !belongsToModule(pkgPath, modPath) {
				res.Edges = append(res.Edges, model.Edge{From: id, To: pkg.ID, Type: model.EdgeContains})
			}
		}
	}
	return res, nil
}

// DirectRequires returns non-indirect require directives from go.mod.
func DirectRequires(moduleRoot string) ([]Require, error) {
	data, err := os.ReadFile(filepath.Join(moduleRoot, "go.mod"))
	if err != nil {
		return nil, fmt.Errorf("read go.mod: %w", err)
	}
	f, err := modfile.Parse("go.mod", data, nil)
	if err != nil {
		return nil, fmt.Errorf("parse go.mod: %w", err)
	}
	var out []Require
	for _, r := range f.Require {
		if r.Indirect {
			continue
		}
		if r.Mod.Path == "" {
			continue
		}
		out = append(out, Require{Path: r.Mod.Path, Version: r.Mod.Version})
	}
	return out, nil
}

func countImporters(g *graph.Graph, modPath, externalModule string) int {
	importers := map[string]struct{}{}
	for _, e := range g.Edges() {
		if e.Type != model.EdgeImports {
			continue
		}
		fromPath := strings.TrimPrefix(e.From, "package:")
		toPath := strings.TrimPrefix(e.To, "package:")
		if !belongsToModule(fromPath, modPath) {
			continue
		}
		if belongsToModule(toPath, externalModule) {
			importers[e.From] = struct{}{}
		}
	}
	return len(importers)
}

func belongsToModule(importPath, modulePath string) bool {
	return importPath == modulePath || strings.HasPrefix(importPath, modulePath+"/")
}
