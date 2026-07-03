package report

import (
	"fmt"
	"sort"
	"strings"

	"github.com/BrendenWalker/lineagis/internal/core/graph"
	"github.com/BrendenWalker/lineagis/internal/core/model"
)

// PackageWhyResult explains upstream imports and provenance for a package (FR-SA-030).
type PackageWhyResult struct {
	SchemaVersion string       `json:"schema_version"`
	Root          string       `json:"root"`
	Upstream      []string     `json:"upstream"`
	Provenance    []model.Edge `json:"provenance,omitempty"`
	Message       string       `json:"message"`
}

const SchemaPackageWhyV1 = "lineage-package-why/v1"

// WhyPackage lists direct upstream import dependencies and provenance links.
func WhyPackage(g *graph.Graph, packageID string) (PackageWhyResult, error) {
	if _, ok := g.GetNode(packageID); !ok {
		return PackageWhyResult{}, fmt.Errorf("package not found: %s (run lineagis analyze first)", strings.TrimPrefix(packageID, "package:"))
	}
	var upstream []string
	for _, n := range g.Neighbors(packageID, []model.EdgeType{model.EdgeImports}) {
		upstream = append(upstream, n.ID)
	}
	sort.Strings(upstream)

	var prov []model.Edge
	for _, e := range g.Edges() {
		if e.From == packageID && e.Type == model.EdgeIntroducedBy {
			prov = append(prov, e)
		}
	}
	sort.Slice(prov, func(i, j int) bool {
		if prov[i].To != prov[j].To {
			return prov[i].To < prov[j].To
		}
		return prov[i].Type < prov[j].Type
	})

	msg := fmt.Sprintf("Package %s has %d direct upstream import(s).", strings.TrimPrefix(packageID, "package:"), len(upstream))
	if len(prov) > 0 {
		msg += fmt.Sprintf(" Linked to %d provenance commit(s).", len(prov))
	}
	return PackageWhyResult{
		SchemaVersion: SchemaPackageWhyV1,
		Root:          packageID,
		Upstream:      upstream,
		Provenance:    prov,
		Message:       msg,
	}, nil
}

// SummaryPackageWhy returns human-readable output.
func SummaryPackageWhy(res PackageWhyResult) string {
	var b strings.Builder
	b.WriteString(res.Message)
	b.WriteString("\n\nUpstream imports:\n")
	if len(res.Upstream) == 0 {
		b.WriteString("  (none)\n")
	} else {
		for _, id := range res.Upstream {
			fmt.Fprintf(&b, "  - %s\n", strings.TrimPrefix(id, "package:"))
		}
	}
	if len(res.Provenance) > 0 {
		b.WriteString("\nProvenance:\n")
		for _, e := range res.Provenance {
			fmt.Fprintf(&b, "  - introduced_by → %s\n", strings.TrimPrefix(e.To, "commit:"))
		}
	}
	return b.String()
}
