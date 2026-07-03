package conformance_test

import (
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/BrendenWalker/lineagis/internal/analyze"
	"github.com/BrendenWalker/lineagis/internal/core/engine"
	"github.com/BrendenWalker/lineagis/internal/core/graph"
	"github.com/BrendenWalker/lineagis/internal/core/model"
	"github.com/BrendenWalker/lineagis/internal/core/query"
	"github.com/BrendenWalker/lineagis/internal/lineage"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	return filepath.Join("..", "..")
}

// TestConformance_sbom_cyclonedx mirrors tests/conformance/sbom-cyclonedx.yaml (P2 exit).
func TestConformance_sbom_cyclonedx(t *testing.T) {
	assertSBOMIngest(t, filepath.Join(repoRoot(t), "examples", "sbom-cyclonedx.json"))
}

// TestConformance_sbom_spdx mirrors tests/conformance/sbom-spdx.yaml (P2 exit).
func TestConformance_sbom_spdx(t *testing.T) {
	assertSBOMIngest(t, filepath.Join(repoRoot(t), "examples", "sbom-spdx.json"))
}

func assertSBOMIngest(t *testing.T, path string) {
	t.Helper()
	g := graph.New()
	if err := lineage.IngestFiles(g, path); err != nil {
		t.Fatal(err)
	}
	if g.NodeCount() < 2 {
		t.Fatalf("expected artifact + dependency, got %d nodes", g.NodeCount())
	}
	artID := model.ArtifactID("abc123")
	depID := model.DependencyID("npm", "lodash", "4.17.21")
	if _, ok := g.GetNode(artID); !ok {
		t.Fatalf("missing artifact %s", artID)
	}
	if _, ok := g.GetNode(depID); !ok {
		t.Fatalf("missing dependency %s", depID)
	}
	edges := g.Edges()
	found := false
	for _, e := range edges {
		if e.From == artID && e.To == depID && e.Type == model.EdgeDependsOn {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("missing depends_on %s -> %s in %+v", artID, depID, edges)
	}
}

// TestSBOMEquivalentGraph (AC-LIN-002): CycloneDX and SPDX fixtures yield identical adjacency.
func TestSBOMEquivalentGraph(t *testing.T) {
	root := repoRoot(t)
	gCDX := graph.New()
	gSPDX := graph.New()
	if err := lineage.IngestFiles(gCDX, filepath.Join(root, "examples", "sbom-cyclonedx.json")); err != nil {
		t.Fatal(err)
	}
	if err := lineage.IngestFiles(gSPDX, filepath.Join(root, "examples", "sbom-spdx.json")); err != nil {
		t.Fatal(err)
	}
	if !graphsEqual(gCDX.Export(), gSPDX.Export()) {
		t.Fatalf("cyclonedx: %+v\nspdx: %+v", gCDX.Export(), gSPDX.Export())
	}
}

func TestSBOMDoubleIngestDedupe(t *testing.T) {
	root := repoRoot(t)
	path := filepath.Join(root, "examples", "sbom-cyclonedx.json")
	g := graph.New()
	if err := lineage.IngestFiles(g, path); err != nil {
		t.Fatal(err)
	}
	n1, e1 := g.NodeCount(), g.EdgeCount()
	if err := lineage.IngestFiles(g, path); err != nil {
		t.Fatal(err)
	}
	if g.NodeCount() != n1 || g.EdgeCount() != e1 {
		t.Fatalf("dedupe failed: nodes %d->%d edges %d->%d", n1, g.NodeCount(), e1, g.EdgeCount())
	}
}

func graphsEqual(a, b model.GraphSnapshot) bool {
	if len(a.Nodes) != len(b.Nodes) || len(a.Edges) != len(b.Edges) {
		return false
	}
	for i := range a.Nodes {
		if a.Nodes[i].ID != b.Nodes[i].ID || a.Nodes[i].Type != b.Nodes[i].Type {
			return false
		}
	}
	for i := range a.Edges {
		e1, e2 := a.Edges[i], b.Edges[i]
		if e1.From != e2.From || e1.To != e2.To || e1.Type != e2.Type {
			return false
		}
	}
	return true
}

// TestConformance_trace_full_chain mirrors tests/conformance/trace-full-chain.yaml (P3 exit).
func TestConformance_trace_full_chain(t *testing.T) {
	g := ingestFullChain(t)
	assertProvenanceEdges(t, g)
	res, err := query.Trace(g, "artifact@sha256:abc123")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{model.CommitID("def456"), model.BuildID("ci-789"), model.ArtifactID("abc123")}
	for _, w := range want {
		found := false
		for _, n := range res.Nodes {
			if n.ID == w {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("trace missing %s in %+v", w, res.Nodes)
		}
	}
	v := engine.VerifyGraph(g)
	if !v.Complete {
		t.Fatalf("expected complete lineage, findings: %v", v.Findings)
	}
}

func ingestFullChain(t *testing.T) *graph.Graph {
	t.Helper()
	root := repoRoot(t)
	g := graph.New()
	files := []string{
		filepath.Join(root, "examples", "sbom-cyclonedx.json"),
		filepath.Join(root, "examples", "build-sidecar.json"),
		filepath.Join(root, "examples", "commit-sidecar.json"),
	}
	if err := lineage.IngestFiles(g, files...); err != nil {
		t.Fatal(err)
	}
	return g
}

func assertProvenanceEdges(t *testing.T, g *graph.Graph) {
	t.Helper()
	artID := model.ArtifactID("abc123")
	buildID := model.BuildID("ci-789")
	commitID := model.CommitID("def456")
	var produced, built bool
	for _, e := range g.Edges() {
		if e.From == artID && e.To == buildID && e.Type == model.EdgeProducedBy {
			produced = true
		}
		if e.From == buildID && e.To == commitID && e.Type == model.EdgeBuiltFrom {
			built = true
		}
	}
	if !produced || !built {
		t.Fatalf("missing provenance edges produced=%v built=%v in %+v", produced, built, g.Edges())
	}
}

// TestFullChainDoubleIngestDedupe (FR-LIN-005): re-ingesting sidecars does not duplicate edges.
func TestFullChainDoubleIngestDedupe(t *testing.T) {
	root := repoRoot(t)
	files := []string{
		filepath.Join(root, "examples", "sbom-cyclonedx.json"),
		filepath.Join(root, "examples", "build-sidecar.json"),
		filepath.Join(root, "examples", "commit-sidecar.json"),
	}
	g := graph.New()
	if err := lineage.IngestFiles(g, files...); err != nil {
		t.Fatal(err)
	}
	n1, e1 := g.NodeCount(), g.EdgeCount()
	if err := lineage.IngestFiles(g, files...); err != nil {
		t.Fatal(err)
	}
	if g.NodeCount() != n1 || g.EdgeCount() != e1 {
		t.Fatalf("dedupe failed: nodes %d->%d edges %d->%d", n1, g.NodeCount(), e1, g.EdgeCount())
	}
}

func TestWhyMissingBuiltFrom(t *testing.T) {
	g := graph.New()
	artID := model.ArtifactID("deadbeef")
	buildID := model.BuildID("ci-42")
	_ = g.AddNode(model.Node{ID: artID, Type: model.NodeArtifact})
	_ = g.AddNode(model.Node{ID: buildID, Type: model.NodeBuild})
	_ = g.AddEdge(artID, buildID, model.EdgeProducedBy)

	res, err := query.Why(g, "artifact@sha256:deadbeef")
	if err != nil {
		t.Fatal(err)
	}
	if res.Complete {
		t.Fatal("expected incomplete chain")
	}
	if res.Gap == "" {
		t.Fatal("expected gap message")
	}
}

// TestConformance_ingest_order_determinism mirrors tests/conformance/ingest-order-determinism.yaml (AC-LIN-005).
func TestConformance_ingest_order_determinism(t *testing.T) {
	root := repoRoot(t)
	files := []string{
		filepath.Join(root, "examples", "commit-sidecar.json"),
		filepath.Join(root, "examples", "build-sidecar.json"),
		filepath.Join(root, "examples", "sbom-cyclonedx.json"),
	}
	reverse := []string{files[2], files[1], files[0]}

	g1 := graph.New()
	g2 := graph.New()
	if err := lineage.IngestFiles(g1, files...); err != nil {
		t.Fatal(err)
	}
	if err := lineage.IngestFiles(g2, reverse...); err != nil {
		t.Fatal(err)
	}
	s1 := g1.Export()
	s2 := g2.Export()
	if len(s1.Edges) != len(s2.Edges) {
		t.Fatalf("edge count mismatch %d vs %d", len(s1.Edges), len(s2.Edges))
	}
	if !graphsEqual(s1, s2) {
		t.Fatalf("graph export mismatch:\n%+v\n%+v", s1, s2)
	}
}

// TestConformance_git_repo_ingest (FR-LIN-003): local git repo path yields a commit node.
func TestConformance_git_repo_ingest(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not in PATH")
	}
	g := graph.New()
	if err := lineage.IngestFiles(g, repoRoot(t)); err != nil {
		t.Fatal(err)
	}
	commits := g.ListByType(model.NodeCommit)
	if len(commits) != 1 {
		t.Fatalf("expected 1 commit node, got %d: %+v", len(commits), commits)
	}
	if commits[0].Metadata["sha"] == "" {
		t.Fatalf("commit missing sha metadata: %+v", commits[0])
	}
}

// TestConformance_self_analysis_mini mirrors tests/conformance/self-analysis-mini.yaml (SA-P1).
func TestConformance_self_analysis_mini(t *testing.T) {
	g := graph.New()
	path := filepath.Join(repoRoot(t), "examples", "self-analysis")
	if err := analyze.Path(g, path); err != nil {
		t.Fatal(err)
	}
	app := model.PackageID("github.com/BrendenWalker/lineagis/examples/self-analysis/app")
	lib := model.PackageID("github.com/BrendenWalker/lineagis/examples/self-analysis/lib")
	assertGraphNode(t, g, app)
	assertGraphNode(t, g, lib)
	assertGraphEdge(t, g, app, lib, model.EdgeImports)
}

// TestConformance_self_analysis_lineagis mirrors tests/conformance/self-analysis-lineagis.yaml (SA-P1).
func TestConformance_self_analysis_lineagis(t *testing.T) {
	g := graph.New()
	if err := analyze.Path(g, repoRoot(t)); err != nil {
		t.Fatal(err)
	}
	cmd := model.PackageID("github.com/BrendenWalker/lineagis/cmd/lineagis")
	queryPkg := model.PackageID("github.com/BrendenWalker/lineagis/internal/core/query")
	graphPkg := model.PackageID("github.com/BrendenWalker/lineagis/internal/core/graph")
	assertGraphNode(t, g, graphPkg)
	assertGraphNode(t, g, cmd)
	assertGraphEdge(t, g, cmd, queryPkg, model.EdgeImports)
	sym := model.SymbolID("github.com/BrendenWalker/lineagis/internal/core/graph", "New")
	assertGraphNode(t, g, sym)
	snap := g.Export()
	if snap.SchemaVersion != model.SchemaGraphV2 {
		t.Fatalf("schema %q want %q", snap.SchemaVersion, model.SchemaGraphV2)
	}
}

// TestAnalyzeProvenanceMerge (AC-SA-005): analyze merges with provenance ingest without ID collision.
func TestAnalyzeProvenanceMerge(t *testing.T) {
	root := repoRoot(t)
	g := graph.New()
	if err := lineage.IngestFiles(g, filepath.Join(root, "examples", "sbom-cyclonedx.json")); err != nil {
		t.Fatal(err)
	}
	artBefore := model.ArtifactID("abc123")
	if err := analyze.Path(g, root); err != nil {
		t.Fatal(err)
	}
	if _, ok := g.GetNode(artBefore); !ok {
		t.Fatalf("provenance artifact %s lost after analyze", artBefore)
	}
	cmd := model.PackageID("github.com/BrendenWalker/lineagis/cmd/lineagis")
	if _, ok := g.GetNode(cmd); !ok {
		t.Fatalf("missing code node %s after merge", cmd)
	}
	snap := g.Export()
	if len(snap.Domains) != 2 {
		t.Fatalf("domains %+v want [provenance code]", snap.Domains)
	}
}

func assertGraphNode(t *testing.T, g *graph.Graph, id string) {
	t.Helper()
	if _, ok := g.GetNode(id); !ok {
		t.Fatalf("missing node %s", id)
	}
}

func assertGraphEdge(t *testing.T, g *graph.Graph, from, to string, typ model.EdgeType) {
	t.Helper()
	for _, e := range g.Edges() {
		if e.From == from && e.To == to && e.Type == typ {
			return
		}
	}
	t.Fatalf("missing edge %s -[%s]-> %s", from, typ, to)
}

// TestConformance_self_analysis_knowledge_graph (SA-P2): docs, tests, and workflow linkage.
func TestConformance_self_analysis_knowledge_graph(t *testing.T) {
	root := repoRoot(t)
	g := graph.New()
	if err := lineage.IngestFiles(g,
		filepath.Join(root, "examples", "sbom-cyclonedx.json"),
		filepath.Join(root, "examples", "commit-sidecar.json"),
	); err != nil {
		t.Fatal(err)
	}
	if err := analyze.Path(g, root); err != nil {
		t.Fatal(err)
	}
	modPath := "github.com/BrendenWalker/lineagis"
	graphPkg := model.PackageID(modPath + "/internal/core/graph")
	specDoc := model.DocID("docs/specs/self-analysis.md")
	assertGraphNode(t, g, specDoc)
	assertGraphEdge(t, g, specDoc, graphPkg, model.EdgeDocuments)

	testFile := model.FileID("internal/core/graph/graph_test.go")
	assertGraphEdge(t, g, testFile, graphPkg, model.EdgeTests)

	ciWF := model.WorkflowID("CI")
	assertGraphNode(t, g, ciWF)
	target := model.TargetID("test-lineage")
	assertGraphEdge(t, g, ciWF, target, model.EdgeRunsIn)

	commitID := model.CommitID("def456")
	assertGraphEdge(t, g, graphPkg, commitID, model.EdgeIntroducedBy)
	moduleID := model.ModuleID(modPath)
	artID := model.ArtifactID("abc123")
	assertGraphEdge(t, g, artID, moduleID, model.EdgeContains)
}
