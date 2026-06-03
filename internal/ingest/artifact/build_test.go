package artifact_test

import (
	"testing"

	"github.com/BrendenWalker/lineagis/internal/core/model"
	"github.com/BrendenWalker/lineagis/internal/ingest/artifact"
)

func TestParseBuildSidecar(t *testing.T) {
	data := []byte(`{
		"id": "ci-789",
		"system": "github-actions",
		"pipeline": "release.yml",
		"commit_sha": "def456",
		"artifacts": ["sha256:abc123"]
	}`)
	res, err := artifact.ParseBuildSidecar(data)
	if err != nil {
		t.Fatal(err)
	}
	buildID := model.BuildID("ci-789")
	commitID := model.CommitID("def456")
	artID := model.ArtifactID("abc123")

	if len(res.Nodes) != 1 || res.Nodes[0].ID != buildID {
		t.Fatalf("nodes: %+v", res.Nodes)
	}
	var produced, built bool
	for _, e := range res.Edges {
		switch {
		case e.From == artID && e.To == buildID && e.Type == model.EdgeProducedBy:
			produced = true
		case e.From == buildID && e.To == commitID && e.Type == model.EdgeBuiltFrom:
			built = true
		}
	}
	if !produced || !built {
		t.Fatalf("edges: %+v produced=%v built=%v", res.Edges, produced, built)
	}
}

func TestParseBuildSidecar_errors(t *testing.T) {
	_, err := artifact.ParseBuildSidecar([]byte(`{"commit_sha":"x"}`))
	if err == nil {
		t.Fatal("expected error for missing id")
	}
	_, err = artifact.ParseBuildSidecar([]byte(`{"id":"ci-1"}`))
	if err == nil {
		t.Fatal("expected error for missing commit_sha")
	}
}

func TestIsBuildSidecar(t *testing.T) {
	if !artifact.IsBuildSidecar([]byte(`{"id":"ci-1","commit_sha":"abc"}`)) {
		t.Fatal("expected true")
	}
	if artifact.IsBuildSidecar([]byte(`{"repo":"r","sha":"a"}`)) {
		t.Fatal("expected false for commit sidecar shape")
	}
}
