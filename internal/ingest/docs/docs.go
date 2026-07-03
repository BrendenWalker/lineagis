package docs

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/BrendenWalker/lineagis/internal/core/model"
)

// Result holds nodes and edges from documentation ingest.
type Result struct {
	Nodes []model.Node
	Edges []model.Edge
}

// Ingest scans markdown under docs/ and links to packages when referenced.
func Ingest(moduleRoot, modPath string, packagePaths []string) (Result, error) {
	docsDir := filepath.Join(moduleRoot, "docs")
	var res Result
	if _, err := os.Stat(docsDir); err != nil {
		return res, nil
	}
	type pkgRef struct {
		id   string
		dir  string
		path string
	}
	var refs []pkgRef
	for _, p := range packagePaths {
		const prefix = "package:"
		if !strings.HasPrefix(p, prefix) {
			continue
		}
		importPath := strings.TrimPrefix(p, prefix)
		dir := strings.TrimPrefix(importPath, modPath+"/")
		if importPath == modPath {
			dir = ""
		}
		refs = append(refs, pkgRef{id: p, dir: dir, path: importPath})
	}

	err := filepath.WalkDir(docsDir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || !strings.HasSuffix(strings.ToLower(d.Name()), ".md") {
			return nil
		}
		rel, err := filepath.Rel(moduleRoot, path)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		content := string(data)
		docID := model.DocID(rel)
		res.Nodes = append(res.Nodes, model.Node{
			ID:   docID,
			Type: model.NodeDoc,
			Metadata: map[string]string{
				"format": "markdown",
			},
		})
		for _, ref := range refs {
			if ref.dir != "" && (strings.Contains(content, ref.dir) || strings.Contains(content, ref.path)) {
				res.Edges = append(res.Edges, model.Edge{From: docID, To: ref.id, Type: model.EdgeDocuments})
			}
		}
		return nil
	})
	return res, err
}
