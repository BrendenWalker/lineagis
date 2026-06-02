package artifact

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/BrendenWalker/lineagis/internal/core/model"
	"github.com/BrendenWalker/lineagis/internal/normalize/resolver"
)

// BuildSidecar links a build to commit and artifacts (FR-LIN-004).
type BuildSidecar struct {
	ID         string   `json:"id"`
	System     string   `json:"system"`
	Pipeline   string   `json:"pipeline"`
	CommitSHA  string   `json:"commit_sha"`
	Artifacts  []string `json:"artifacts"`
	Status     string   `json:"status,omitempty"`
	Timestamp  string   `json:"timestamp,omitempty"`
}

// IngestResult holds nodes and edges from build sidecar ingest.
type IngestResult struct {
	Nodes []model.Node
	Edges []model.Edge
}

// ParseBuildSidecar parses build metadata and emits produced_by / built_from edges.
func ParseBuildSidecar(data []byte) (IngestResult, error) {
	var sc BuildSidecar
	if err := json.Unmarshal(data, &sc); err != nil {
		return IngestResult{}, fmt.Errorf("build sidecar: %w", err)
	}
	if sc.ID == "" {
		return IngestResult{}, fmt.Errorf("build sidecar: missing id")
	}
	if sc.CommitSHA == "" {
		return IngestResult{}, fmt.Errorf("build sidecar: missing commit_sha")
	}
	buildID := model.BuildID(sc.ID)
	meta := map[string]string{
		"system":   sc.System,
		"pipeline": sc.Pipeline,
	}
	if sc.Status != "" {
		meta["status"] = sc.Status
	}
	if sc.Timestamp != "" {
		meta["timestamp"] = sc.Timestamp
	}
	var res IngestResult
	res.Nodes = append(res.Nodes, model.Node{ID: buildID, Type: model.NodeBuild, Metadata: meta})
	commitID := model.CommitID(sc.CommitSHA)
	res.Edges = append(res.Edges, model.Edge{From: buildID, To: commitID, Type: model.EdgeBuiltFrom})

	for _, art := range sc.Artifacts {
		hex := resolver.HexFromDigest(art)
		artID := model.ArtifactID(hex)
		res.Edges = append(res.Edges, model.Edge{From: artID, To: buildID, Type: model.EdgeProducedBy})
	}
	return res, nil
}

// ParseBuildSidecarFile reads a build sidecar from disk.
func ParseBuildSidecarFile(path string) (IngestResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return IngestResult{}, err
	}
	return ParseBuildSidecar(data)
}

// IsBuildSidecar returns true if JSON looks like a build sidecar.
func IsBuildSidecar(data []byte) bool {
	var sc BuildSidecar
	if json.Unmarshal(data, &sc) != nil {
		return false
	}
	return sc.ID != "" && sc.CommitSHA != ""
}

// EnsureArtifactNodes creates artifact nodes for digests referenced by build if missing.
func EnsureArtifactNodes(g interface {
	GetNode(string) (model.Node, bool)
	AddNode(model.Node) error
}, digests []string) error {
	type adder interface {
		GetNode(string) (model.Node, bool)
		AddNode(model.Node) error
	}
	for _, art := range digests {
		hex := resolver.HexFromDigest(art)
		id := model.ArtifactID(hex)
		if _, ok := g.GetNode(id); ok {
			continue
		}
		meta := map[string]string{"digest": id}
		if err := g.AddNode(model.Node{ID: id, Type: model.NodeArtifact, Metadata: meta}); err != nil {
			return err
		}
	}
	return nil
}

// EnsureCommitNode adds commit node if missing.
func EnsureCommitNode(g interface {
	GetNode(string) (model.Node, bool)
	AddNode(model.Node) error
}, sha string) error {
	id := model.CommitID(sha)
	if _, ok := g.GetNode(id); ok {
		return nil
	}
	meta := map[string]string{"sha": strings.ToLower(sha)}
	return g.AddNode(model.Node{ID: id, Type: model.NodeCommit, Metadata: meta})
}
