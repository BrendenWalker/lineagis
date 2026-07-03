package workflow

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/BrendenWalker/lineagis/internal/core/model"
)

// Result holds nodes and edges from GitHub Actions workflow ingest.
type Result struct {
	Nodes []model.Node
	Edges []model.Edge
}

var (
	nameRE = regexp.MustCompile(`(?m)^\s*name:\s*['"]?([^'"\n]+)`)
	makeRE = regexp.MustCompile(`run:\s*make\s+([a-zA-Z0-9_-]+)`)
)

// Ingest parses .github/workflows/*.yml and *.yaml under moduleRoot.
func Ingest(moduleRoot string) (Result, error) {
	wfDir := filepath.Join(moduleRoot, ".github", "workflows")
	var res Result
	if _, err := os.Stat(wfDir); err != nil {
		return res, nil
	}
	entries, err := os.ReadDir(wfDir)
	if err != nil {
		return Result{}, err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".yml") && !strings.HasSuffix(name, ".yaml") {
			continue
		}
		path := filepath.Join(wfDir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			return Result{}, err
		}
		text := string(data)
		wfName := strings.TrimSuffix(name, filepath.Ext(name))
		if m := nameRE.FindStringSubmatch(text); len(m) > 1 {
			wfName = strings.TrimSpace(m[1])
		}
		rel, _ := filepath.Rel(moduleRoot, path)
		wfID := model.WorkflowID(wfName)
		res.Nodes = append(res.Nodes, model.Node{
			ID:   wfID,
			Type: model.NodeWorkflow,
			Metadata: map[string]string{
				"path": filepath.ToSlash(rel),
			},
		})
		seenTarget := map[string]struct{}{}
		for _, m := range makeRE.FindAllStringSubmatch(text, -1) {
			if len(m) < 2 {
				continue
			}
			targetName := m[1]
			targetID := model.TargetID(targetName)
			if _, ok := seenTarget[targetID]; !ok {
				seenTarget[targetID] = struct{}{}
				res.Nodes = append(res.Nodes, model.Node{
					ID:   targetID,
					Type: model.NodeTarget,
					Metadata: map[string]string{
						"kind": "make",
					},
				})
			}
			res.Edges = append(res.Edges, model.Edge{From: wfID, To: targetID, Type: model.EdgeRunsIn})
		}
	}
	return res, nil
}
