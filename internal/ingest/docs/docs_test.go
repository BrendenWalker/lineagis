package docs_test

import (
	"path/filepath"
	"testing"

	"github.com/BrendenWalker/lineagis/internal/core/model"
	"github.com/BrendenWalker/lineagis/internal/ingest/docs"
	goingest "github.com/BrendenWalker/lineagis/internal/ingest/go"
)

func TestIngest_linksSpecToGraphPackage(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	modPath, err := goingest.ModulePath(root)
	if err != nil {
		t.Fatal(err)
	}
	graphPkg := model.PackageID(modPath + "/internal/core/graph")
	res, err := docs.Ingest(root, modPath, []string{graphPkg})
	if err != nil {
		t.Fatal(err)
	}
	specDoc := model.DocID("docs/specs/self-analysis.md")
	foundDoc := false
	foundEdge := false
	for _, n := range res.Nodes {
		if n.ID == specDoc {
			foundDoc = true
		}
	}
	for _, e := range res.Edges {
		if e.From == specDoc && e.To == graphPkg && e.Type == model.EdgeDocuments {
			foundEdge = true
		}
	}
	if !foundDoc {
		t.Fatalf("missing doc node %s in %+v", specDoc, res.Nodes)
	}
	if !foundEdge {
		t.Fatalf("missing documents edge from spec to graph package")
	}
}
