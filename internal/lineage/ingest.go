package lineage

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/BrendenWalker/lineagis/internal/core/graph"
	"github.com/BrendenWalker/lineagis/internal/core/model"
	"github.com/BrendenWalker/lineagis/internal/ingest/artifact"
	"github.com/BrendenWalker/lineagis/internal/ingest/git"
	"github.com/BrendenWalker/lineagis/internal/ingest/sbom"
	"github.com/BrendenWalker/lineagis/internal/normalize/dedupe"
)

// IngestFiles merges one or more lineage input files into g.
func IngestFiles(g *graph.Graph, paths ...string) error {
	for _, path := range paths {
		if err := ingestFile(g, path); err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
	}
	return nil
}

func ingestFile(g *graph.Graph, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		// Directory may be git repo
		if fi, statErr := os.Stat(path); statErr == nil && fi.IsDir() {
			return ingestGitDir(g, path)
		}
		return err
	}
	trim := strings.TrimSpace(string(data))

	if git.IsCommitSidecar(data) {
		n, err := git.ParseSidecar(data)
		if err != nil {
			return err
		}
		return dedupe.Apply(g, []model.Node{n}, nil)
	}
	if artifact.IsBuildSidecar(data) {
		res, err := artifact.ParseBuildSidecar(data)
		if err != nil {
			return err
		}
		var sc artifact.BuildSidecar
		_ = json.Unmarshal(data, &sc)
		if err := artifact.EnsureCommitNode(g, sc.CommitSHA); err != nil {
			return err
		}
		if err := artifact.EnsureArtifactNodes(g, sc.Artifacts); err != nil {
			return err
		}
		return dedupe.Apply(g, res.Nodes, res.Edges)
	}

	if strings.Contains(trim, `"bomFormat"`) || strings.Contains(trim, `"spdxVersion"`) {
		nodes, edges, err := sbom.ParseFile(path)
		if err != nil {
			return err
		}
		return dedupe.Apply(g, nodes, edges)
	}

	return fmt.Errorf("unrecognized input (want SBOM JSON, commit/build sidecar, or git repo path)")
}

func ingestGitDir(g *graph.Graph, path string) error {
	gitDir := strings.TrimSuffix(path, string(os.PathSeparator)) + string(os.PathSeparator) + ".git"
	if _, err := os.Stat(gitDir); err != nil {
		return fmt.Errorf("not a git repository: %s", path)
	}
	n, err := git.FromRepo(path)
	if err != nil {
		return err
	}
	return dedupe.Apply(g, []model.Node{n}, nil)
}
