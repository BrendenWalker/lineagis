package arch

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/BrendenWalker/lineagis/internal/core/graph"
	"github.com/BrendenWalker/lineagis/internal/core/model"
)

func TestValidateImports_forbiddenCmdToStorage(t *testing.T) {
	modPath := "example.com/mod"
	rules := Rules{
		Layers: map[string]string{
			"cmd":     "cmd/",
			"storage": "internal/storage/",
		},
		Forbidden: []ForbiddenImport{{From: "cmd", To: "storage"}},
	}
	g := graph.New()
	cmdPkg := model.PackageID(modPath + "/cmd/app")
	storagePkg := model.PackageID(modPath + "/internal/storage/memory")
	_ = g.AddNode(model.Node{ID: cmdPkg, Type: model.NodePackage})
	_ = g.AddNode(model.Node{ID: storagePkg, Type: model.NodePackage})
	_ = g.AddEdge(cmdPkg, storagePkg, model.EdgeImports)

	violations := ValidateImports(g, modPath, rules)
	if len(violations) != 1 {
		t.Fatalf("violations: %+v", violations)
	}
	if violations[0].FromLayer != "cmd" || violations[0].ToLayer != "storage" {
		t.Fatalf("unexpected violation: %+v", violations[0])
	}
}

func TestLoadRules_fixture(t *testing.T) {
	path := filepath.Join("..", "..", "tests", "conformance", "fixtures", "lineagis-arch-cmd-storage.yaml")
	if _, err := os.Stat(path); err != nil {
		t.Skip(path)
	}
	rules, err := LoadRules(path)
	if err != nil {
		t.Fatal(err)
	}
	if rules.Layers["cmd"] != "cmd/" {
		t.Fatalf("layers: %+v", rules.Layers)
	}
}
