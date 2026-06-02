package graph_test

import (
	"testing"

	"github.com/BrendenWalker/lineagis/internal/core/graph"
	"github.com/BrendenWalker/lineagis/internal/core/model"
)

func TestAddNodeAndList(t *testing.T) {
	g := graph.New()
	if err := g.AddNode(model.Node{ID: model.CommitID("abc"), Type: model.NodeCommit, Metadata: map[string]string{"sha": "abc"}}); err != nil {
		t.Fatal(err)
	}
	n, ok := g.GetNode(model.CommitID("abc"))
	if !ok || n.Type != model.NodeCommit {
		t.Fatalf("get node: %+v ok=%v", n, ok)
	}
	if len(g.ListByType(model.NodeCommit)) != 1 {
		t.Fatal("list by type")
	}
}

func TestRejectProvenanceCycle(t *testing.T) {
	g := graph.New()
	_ = g.AddNode(model.Node{ID: "artifact:sha256:a", Type: model.NodeArtifact})
	_ = g.AddNode(model.Node{ID: "build:b1", Type: model.NodeBuild})
	_ = g.AddNode(model.Node{ID: "commit:c1", Type: model.NodeCommit})
	_ = g.AddEdge("artifact:sha256:a", "build:b1", model.EdgeProducedBy)
	_ = g.AddEdge("build:b1", "commit:c1", model.EdgeBuiltFrom)
	// Closing the loop: commit -> artifact would cycle through existing chain
	if err := g.AddEdge("commit:c1", "artifact:sha256:a", model.EdgeProducedBy); err == nil {
		t.Fatal("expected cycle rejection when linking commit back to artifact")
	}
}

func TestDependsOnAmongDependenciesAllowed(t *testing.T) {
	g := graph.New()
	_ = g.AddNode(model.Node{ID: "dependency:npm:a@1", Type: model.NodeDependency})
	_ = g.AddNode(model.Node{ID: "dependency:npm:b@2", Type: model.NodeDependency})
	if err := g.AddEdge("dependency:npm:a@1", "dependency:npm:b@2", model.EdgeDependsOn); err != nil {
		t.Fatal(err)
	}
	if err := g.AddEdge("dependency:npm:b@2", "dependency:npm:a@1", model.EdgeDependsOn); err != nil {
		t.Fatal("depends_on cycle among dependencies should be allowed")
	}
}

func TestExportDeterministic(t *testing.T) {
	g1 := graph.New()
	g2 := graph.New()
	for _, g := range []*graph.Graph{g1, g2} {
		_ = g.AddNode(model.Node{ID: "artifact:sha256:z", Type: model.NodeArtifact})
		_ = g.AddNode(model.Node{ID: "artifact:sha256:a", Type: model.NodeArtifact})
	}
	s1 := g1.Export()
	s2 := g2.Export()
	if len(s1.Nodes) != 2 || s1.Nodes[0].ID >= s1.Nodes[1].ID {
		t.Fatalf("nodes not sorted: %+v", s1.Nodes)
	}
	if s1.Nodes[0].ID != s2.Nodes[0].ID {
		t.Fatal("export order mismatch")
	}
}

func TestLoadSnapshot(t *testing.T) {
	g := graph.New()
	_ = g.AddNode(model.Node{ID: model.ArtifactID("abc123"), Type: model.NodeArtifact})
	snap := g.Export()
	g2 := graph.New()
	if err := g2.LoadSnapshot(snap); err != nil {
		t.Fatal(err)
	}
	if g2.NodeCount() != 1 {
		t.Fatal("reload")
	}
}
