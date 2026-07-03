package report

import (
	"fmt"
	"sort"
	"strings"

	"github.com/BrendenWalker/lineagis/internal/core/graph"
	"github.com/BrendenWalker/lineagis/internal/core/model"
)

// ImpactResult lists downstream packages, tests, and docs (FR-SA-031).
type ImpactResult struct {
	SchemaVersion string   `json:"schema_version"`
	Root          string   `json:"root"`
	Packages      []string `json:"packages"`
	Tests         []string `json:"tests"`
	Docs          []string `json:"docs"`
}

const SchemaImpactV1 = "lineage-impact/v1"

// ImpactPackage walks reverse imports from packageID and collects related tests/docs.
func ImpactPackage(g *graph.Graph, packageID string) (ImpactResult, error) {
	if _, ok := g.GetNode(packageID); !ok {
		return ImpactResult{}, fmt.Errorf("package not found: %s (run lineagis analyze first)", strings.TrimPrefix(packageID, "package:"))
	}
	pkgs := downstreamPackages(g, packageID)
	allPkgs := append([]string{packageID}, pkgs...)
	tests := relatedByEdge(g, allPkgs, model.EdgeTests, true)
	docs := relatedByEdge(g, allPkgs, model.EdgeDocuments, false)
	return ImpactResult{
		SchemaVersion: SchemaImpactV1,
		Root:          packageID,
		Packages:      pkgs,
		Tests:         tests,
		Docs:          docs,
	}, nil
}

func downstreamPackages(g *graph.Graph, root string) []string {
	visited := map[string]struct{}{root: {}}
	queue := []string{root}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, n := range g.IncomingNeighbors(cur, []model.EdgeType{model.EdgeImports}) {
			if _, ok := visited[n.ID]; ok {
				continue
			}
			visited[n.ID] = struct{}{}
			queue = append(queue, n.ID)
		}
	}
	delete(visited, root)
	var out []string
	for id := range visited {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

func relatedByEdge(g *graph.Graph, packageIDs []string, edgeType model.EdgeType, fromFile bool) []string {
	pkgSet := make(map[string]struct{}, len(packageIDs))
	for _, id := range packageIDs {
		pkgSet[id] = struct{}{}
	}
	seen := map[string]struct{}{}
	var out []string
	for _, e := range g.Edges() {
		if e.Type != edgeType {
			continue
		}
		var related string
		if fromFile {
			if _, ok := pkgSet[e.To]; !ok {
				continue
			}
			related = e.From
		} else {
			if _, ok := pkgSet[e.To]; !ok {
				continue
			}
			related = e.From
		}
		if _, dup := seen[related]; dup {
			continue
		}
		seen[related] = struct{}{}
		out = append(out, related)
	}
	sort.Strings(out)
	return out
}

// ImpactMarkdown formats ImpactResult for reports.
func ImpactMarkdown(res ImpactResult) string {
	var b strings.Builder
	root := strings.TrimPrefix(res.Root, "package:")
	fmt.Fprintf(&b, "# Impact: %s\n\n", root)
	b.WriteString("## Downstream packages\n\n")
	if len(res.Packages) == 0 {
		b.WriteString("_None_\n")
	} else {
		for _, p := range res.Packages {
			fmt.Fprintf(&b, "- `%s`\n", strings.TrimPrefix(p, "package:"))
		}
	}
	b.WriteString("\n## Tests\n\n")
	if len(res.Tests) == 0 {
		b.WriteString("_None_\n")
	} else {
		for _, t := range res.Tests {
			fmt.Fprintf(&b, "- `%s`\n", strings.TrimPrefix(t, "file:"))
		}
	}
	b.WriteString("\n## Documentation\n\n")
	if len(res.Docs) == 0 {
		b.WriteString("_None_\n")
	} else {
		for _, d := range res.Docs {
			fmt.Fprintf(&b, "- `%s`\n", strings.TrimPrefix(d, "doc:"))
		}
	}
	return b.String()
}
