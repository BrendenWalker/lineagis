package sbom_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/BrendenWalker/lineagis/internal/core/model"
	"github.com/BrendenWalker/lineagis/internal/ingest/sbom"
)

func examplesDir(t *testing.T) string {
	t.Helper()
	return filepath.Join("..", "..", "..", "examples")
}

func TestParseCycloneDXFixture(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(examplesDir(t), "sbom-cyclonedx.json"))
	if err != nil {
		t.Fatal(err)
	}
	nodes, edges, err := sbom.ParseCycloneDX(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) < 2 || len(edges) < 1 {
		t.Fatalf("nodes=%d edges=%d", len(nodes), len(edges))
	}
	artID := model.ArtifactID("abc123")
	foundArt := false
	for _, n := range nodes {
		if n.ID == artID {
			foundArt = true
		}
	}
	if !foundArt {
		t.Fatal("missing root artifact")
	}
	depID := model.DependencyID("npm", "lodash", "4.17.21")
	if edges[0].From != artID || edges[0].To != depID || edges[0].Type != model.EdgeDependsOn {
		t.Fatalf("unexpected edge: %+v", edges[0])
	}
}

func TestParseSPDXFixture(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(examplesDir(t), "sbom-spdx.json"))
	if err != nil {
		t.Fatal(err)
	}
	nodes, edges, err := sbom.ParseSPDX(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) < 2 || len(edges) < 1 {
		t.Fatalf("nodes=%d edges=%d", len(nodes), len(edges))
	}
	artID := model.ArtifactID("abc123")
	depID := model.DependencyID("npm", "lodash", "4.17.21")
	if edges[0].From != artID || edges[0].To != depID || edges[0].Type != model.EdgeDependsOn {
		t.Fatalf("unexpected edge: %+v", edges[0])
	}
}

func TestCycloneDXAndSPDXEquivalentParse(t *testing.T) {
	root := examplesDir(t)
	cdx, err := os.ReadFile(filepath.Join(root, "sbom-cyclonedx.json"))
	if err != nil {
		t.Fatal(err)
	}
	spdx, err := os.ReadFile(filepath.Join(root, "sbom-spdx.json"))
	if err != nil {
		t.Fatal(err)
	}
	n1, e1, err := sbom.ParseCycloneDX(cdx)
	if err != nil {
		t.Fatal(err)
	}
	n2, e2, err := sbom.ParseSPDX(spdx)
	if err != nil {
		t.Fatal(err)
	}
	if !sameNodesAndEdges(n1, e1, n2, e2) {
		t.Fatalf("parse mismatch:\ncdx nodes=%+v edges=%+v\nspdx nodes=%+v edges=%+v", n1, e1, n2, e2)
	}
}

func sameNodesAndEdges(n1 []model.Node, e1 []model.Edge, n2 []model.Node, e2 []model.Edge) bool {
	if len(n1) != len(n2) || len(e1) != len(e2) {
		return false
	}
	ids1 := nodeIDs(n1)
	ids2 := nodeIDs(n2)
	for i := range ids1 {
		if ids1[i] != ids2[i] {
			return false
		}
	}
	for i := range e1 {
		if e1[i].From != e2[i].From || e1[i].To != e2[i].To || e1[i].Type != e2[i].Type {
			return false
		}
	}
	return true
}

func nodeIDs(nodes []model.Node) []string {
	ids := make([]string, len(nodes))
	for i, n := range nodes {
		ids[i] = n.ID
	}
	return ids
}
