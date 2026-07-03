package report

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/BrendenWalker/lineagis/internal/analyze"
	"github.com/BrendenWalker/lineagis/internal/core/graph"
	"github.com/BrendenWalker/lineagis/internal/core/model"
)

// WriteTree writes generated artifacts under rootDir (FR-SA-051).
func WriteTree(g *graph.Graph, rootDir string) error {
	dirs := []string{
		filepath.Join(rootDir, "architecture"),
		filepath.Join(rootDir, "reports"),
		filepath.Join(rootDir, "diagrams"),
		filepath.Join(rootDir, "lineage"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return fmt.Errorf("mkdir %s: %w", d, err)
		}
	}

	archPath := filepath.Join(rootDir, "architecture", "overview.md")
	if err := os.WriteFile(archPath, []byte(ArchitectureMarkdown(g)), 0o644); err != nil {
		return err
	}

	depPath := filepath.Join(rootDir, "reports", "dependency-report.md")
	if err := os.WriteFile(depPath, []byte(DependencyMarkdown(g)), 0o644); err != nil {
		return err
	}

	orphanPath := filepath.Join(rootDir, "reports", "orphan-packages.md")
	if err := os.WriteFile(orphanPath, []byte(OrphanPackagesMarkdown(g)), 0o644); err != nil {
		return err
	}

	dotPath := filepath.Join(rootDir, "diagrams", "imports.dot")
	if err := os.WriteFile(dotPath, []byte(analyze.PackageImportDOT(g)), 0o644); err != nil {
		return err
	}

	snap := g.Export()
	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	lineagePath := filepath.Join(rootDir, "lineage", "lineage.json")
	return os.WriteFile(lineagePath, data, 0o644)
}

// DependencyMarkdown summarizes external module dependencies.
func DependencyMarkdown(g *graph.Graph) string {
	var b strings.Builder
	b.WriteString("# Dependency Report\n\n")
	var external []model.Node
	for _, n := range g.ListByType(model.NodeModule) {
		if n.Metadata["external"] == "true" {
			external = append(external, n)
		}
	}
	sort.Slice(external, func(i, j int) bool { return external[i].ID < external[j].ID })
	if len(external) == 0 {
		b.WriteString("_No external modules recorded._\n")
		return b.String()
	}
	for _, n := range external {
		path := strings.TrimPrefix(n.ID, "module:")
		ver := n.Metadata["version"]
		count := n.Metadata["import_count"]
		fmt.Fprintf(&b, "- `%s`", path)
		if ver != "" {
			fmt.Fprintf(&b, " @ %s", ver)
		}
		if count != "" {
			fmt.Fprintf(&b, " (imported by %s packages)", count)
		}
		b.WriteString("\n")
	}
	return b.String()
}

// OrphanPackagesMarkdown lists packages with no importers and no tests.
func OrphanPackagesMarkdown(g *graph.Graph) string {
	imported := map[string]struct{}{}
	for _, e := range g.Edges() {
		if e.Type == model.EdgeImports {
			imported[e.To] = struct{}{}
		}
	}
	tested := map[string]struct{}{}
	for _, e := range g.Edges() {
		if e.Type == model.EdgeTests {
			tested[e.To] = struct{}{}
		}
	}
	var orphans []string
	for _, n := range g.ListByType(model.NodePackage) {
		if strings.Contains(n.Metadata["name"], "_test") {
			continue
		}
		if _, ok := imported[n.ID]; ok {
			continue
		}
		if _, ok := tested[n.ID]; ok {
			continue
		}
		orphans = append(orphans, strings.TrimPrefix(n.ID, "package:"))
	}
	sort.Strings(orphans)
	var b strings.Builder
	b.WriteString("# Orphan Packages\n\n")
	b.WriteString("Packages with no in-repo importers and no linked tests.\n\n")
	if len(orphans) == 0 {
		b.WriteString("_None_\n")
		return b.String()
	}
	for _, p := range orphans {
		fmt.Fprintf(&b, "- `%s`\n", p)
	}
	return b.String()
}

// WriteImpactReport writes blast-radius markdown for a package.
func WriteImpactReport(g *graph.Graph, packageID, path string) error {
	res, err := ImpactPackage(g, packageID)
	if err != nil {
		return err
	}
	return os.WriteFile(path, []byte(ImpactMarkdown(res)), 0o644)
}
