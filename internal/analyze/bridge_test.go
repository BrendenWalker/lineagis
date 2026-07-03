package analyze_test

import (
	"testing"

	"github.com/BrendenWalker/lineagis/internal/analyze"
	"github.com/BrendenWalker/lineagis/internal/core/graph"
	"github.com/BrendenWalker/lineagis/internal/core/model"
)

func TestBridge_provenanceAndCode(t *testing.T) {
	g := graph.New()
	modPath := "example.com/mod"
	moduleID := model.ModuleID(modPath)
	pkgID := model.PackageID(modPath + "/pkg")
	commitID := model.CommitID("abc")
	artID := model.ArtifactID("deadbeef")
	_ = g.AddNode(model.Node{ID: moduleID, Type: model.NodeModule})
	_ = g.AddNode(model.Node{ID: pkgID, Type: model.NodePackage})
	_ = g.AddNode(model.Node{ID: commitID, Type: model.NodeCommit})
	_ = g.AddNode(model.Node{ID: artID, Type: model.NodeArtifact})

	_, edges := analyze.Bridge(g, modPath)
	var introduced, contains bool
	for _, e := range edges {
		if e.From == pkgID && e.To == commitID && e.Type == model.EdgeIntroducedBy {
			introduced = true
		}
		if e.From == artID && e.To == moduleID && e.Type == model.EdgeContains {
			contains = true
		}
	}
	if !introduced || !contains {
		t.Fatalf("bridge edges missing: %+v", edges)
	}
}
