package workflow_test

import (
	"path/filepath"
	"testing"

	"github.com/BrendenWalker/lineagis/internal/core/model"
	"github.com/BrendenWalker/lineagis/internal/ingest/workflow"
)

func TestIngest_ciWorkflow(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	res, err := workflow.Ingest(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Nodes) < 1 {
		t.Fatal("expected workflow nodes")
	}
	var ci bool
	for _, n := range res.Nodes {
		if n.Type == model.NodeWorkflow && n.Metadata["path"] == ".github/workflows/ci.yml" {
			ci = true
		}
	}
	if !ci {
		t.Fatalf("missing ci workflow in %+v", res.Nodes)
	}
	var testLineageTarget bool
	targetID := model.TargetID("test-lineage")
	for _, n := range res.Nodes {
		if n.ID == targetID {
			testLineageTarget = true
		}
	}
	if !testLineageTarget {
		t.Fatalf("missing test-lineage target in %+v", res.Nodes)
	}
}
