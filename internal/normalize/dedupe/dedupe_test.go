package dedupe_test

import (
	"testing"

	"github.com/BrendenWalker/lineagis/internal/core/graph"
	"github.com/BrendenWalker/lineagis/internal/core/model"
	"github.com/BrendenWalker/lineagis/internal/normalize/dedupe"
)

func TestApply_dedupesNodesAndEdges(t *testing.T) {
	g := graph.New()
	art := model.Node{ID: model.ArtifactID("abc"), Type: model.NodeArtifact}
	build := model.Node{ID: model.BuildID("ci-1"), Type: model.NodeBuild}
	edge := model.Edge{From: art.ID, To: build.ID, Type: model.EdgeProducedBy}

	if err := dedupe.Apply(g, []model.Node{art, build}, []model.Edge{edge}); err != nil {
		t.Fatal(err)
	}
	n1, e1 := g.NodeCount(), g.EdgeCount()

	if err := dedupe.Apply(g, []model.Node{art, build}, []model.Edge{edge}); err != nil {
		t.Fatal(err)
	}
	if g.NodeCount() != n1 || g.EdgeCount() != e1 {
		t.Fatalf("dedupe failed: nodes %d edges %d after re-apply", g.NodeCount(), g.EdgeCount())
	}
}
