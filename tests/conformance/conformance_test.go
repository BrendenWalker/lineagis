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

func TestSBOMCycloneDXIngest(t *testing.T) {
	root := repoRoot(t)
	g := graph.New()
	if err := lineage.IngestFiles(g, filepath.Join(root, "examples", "sbom-cyclonedx.json")); err != nil {
		t.Fatal(err)
	}
	if g.NodeCount() < 2 {
		t.Fatalf("expected artifact + dependency, got %d nodes", g.NodeCount())
	}
	id := model.ArtifactID("abc123")
	if _, ok := g.GetNode(id); !ok {
		t.Fatalf("missing artifact node %s", id)
	}
}

func TestSBOMSPDXIngest(t *testing.T) {
	root := repoRoot(t)
	g := graph.New()
	if err := lineage.IngestFiles(g, filepath.Join(root, "examples", "sbom-spdx.json")); err != nil {
		t.Fatal(err)
	}
	id := model.ArtifactID("abc123")
	if _, ok := g.GetNode(id); !ok {
		t.Fatalf("missing artifact %s", id)
	}
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
