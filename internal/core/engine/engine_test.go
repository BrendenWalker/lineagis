package engine_test

import (
	"testing"

	"github.com/BrendenWalker/lineagis/internal/core/engine"
	"github.com/BrendenWalker/lineagis/internal/core/graph"
	"github.com/BrendenWalker/lineagis/internal/core/model"
)

func TestTraceFullChain(t *testing.T) {
	g := graph.New()
	art := model.ArtifactID("abc123")
	build := model.BuildID("ci-789")
	commit := model.CommitID("def456")
	_ = g.AddNode(model.Node{ID: art, Type: model.NodeArtifact})
	_ = g.AddNode(model.Node{ID: build, Type: model.NodeBuild})
	_ = g.AddNode(model.Node{ID: commit, Type: model.NodeCommit})
	_ = g.AddEdge(art, build, model.EdgeProducedBy)
	_ = g.AddEdge(build, commit, model.EdgeBuiltFrom)

	res, err := engine.Trace(g, art)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Nodes) < 3 {
		t.Fatalf("nodes: %+v", res.Nodes)
	}
}

func TestWhyBrokenChain(t *testing.T) {
	g := graph.New()
	art := model.ArtifactID("deadbeef")
	build := model.BuildID("ci-42")
	_ = g.AddNode(model.Node{ID: art, Type: model.NodeArtifact})
	_ = g.AddNode(model.Node{ID: build, Type: model.NodeBuild})
	_ = g.AddEdge(art, build, model.EdgeProducedBy)

	res, err := engine.Why(g, art)
	if err != nil {
		t.Fatal(err)
	}
	if res.Complete {
		t.Fatal("expected gap")
	}
}

func TestVerifyComplete(t *testing.T) {
	g := graph.New()
	art := model.ArtifactID("abc123")
	build := model.BuildID("ci-789")
	commit := model.CommitID("def456")
	_ = g.AddNode(model.Node{ID: art, Type: model.NodeArtifact})
	_ = g.AddNode(model.Node{ID: build, Type: model.NodeBuild})
	_ = g.AddNode(model.Node{ID: commit, Type: model.NodeCommit})
	_ = g.AddEdge(art, build, model.EdgeProducedBy)
	_ = g.AddEdge(build, commit, model.EdgeBuiltFrom)

	v := engine.VerifyGraph(g)
	if !v.Complete {
		t.Fatalf("expected complete: %+v", v.Findings)
	}
}
