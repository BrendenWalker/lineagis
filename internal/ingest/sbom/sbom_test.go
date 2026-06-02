package sbom_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/BrendenWalker/lineagis/internal/core/model"
	"github.com/BrendenWalker/lineagis/internal/ingest/sbom"
)

func TestParseCycloneDXFixture(t *testing.T) {
	root := filepath.Join("..", "..", "..", "examples")
	data, err := os.ReadFile(filepath.Join(root, "sbom-cyclonedx.json"))
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
	found := false
	for _, n := range nodes {
		if n.ID == model.ArtifactID("abc123") {
			found = true
		}
	}
	if !found {
		t.Fatal("missing root artifact")
	}
}
