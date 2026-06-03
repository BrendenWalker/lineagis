package conformance_test

import (
	"path/filepath"
	"testing"

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

func TestTraceFullChain(t *testing.T) {
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

func TestIngestOrderDeterminism(t *testing.T) {
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
	for i := range s1.Edges {
		e1, e2 := s1.Edges[i], s2.Edges[i]
		if e1.From != e2.From || e1.To != e2.To || e1.Type != e2.Type {
			t.Fatalf("edge mismatch at %d: %+v vs %+v", i, e1, e2)
		}
	}
}
