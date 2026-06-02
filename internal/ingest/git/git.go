package git

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/BrendenWalker/lineagis/internal/core/model"
)

// CommitSidecar is JSON commit metadata (FR-LIN-003).
type CommitSidecar struct {
	Repo      string `json:"repo"`
	SHA       string `json:"sha"`
	Author    string `json:"author,omitempty"`
	Timestamp string `json:"timestamp,omitempty"`
}

// ParseSidecar parses commit sidecar JSON into a node.
func ParseSidecar(data []byte) (model.Node, error) {
	var sc CommitSidecar
	if err := json.Unmarshal(data, &sc); err != nil {
		return model.Node{}, fmt.Errorf("commit sidecar: %w", err)
	}
	if sc.SHA == "" {
		return model.Node{}, fmt.Errorf("commit sidecar: missing sha")
	}
	if sc.Repo == "" {
		return model.Node{}, fmt.Errorf("commit sidecar: missing repo")
	}
	meta := map[string]string{"repo": sc.Repo, "sha": strings.ToLower(sc.SHA)}
	if sc.Author != "" {
		meta["author"] = sc.Author
	}
	if sc.Timestamp != "" {
		meta["timestamp"] = sc.Timestamp
	}
	return model.Node{
		ID:       model.CommitID(sc.SHA),
		Type:     model.NodeCommit,
		Metadata: meta,
	}, nil
}

// ParseSidecarFile reads a commit sidecar file.
func ParseSidecarFile(path string) (model.Node, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return model.Node{}, err
	}
	return ParseSidecar(data)
}

// IsCommitSidecar returns true if JSON looks like a commit sidecar.
func IsCommitSidecar(data []byte) bool {
	var sc CommitSidecar
	if json.Unmarshal(data, &sc) != nil {
		return false
	}
	return sc.SHA != "" && sc.Repo != ""
}

// FromRepo reads HEAD commit from a local git repository (offline).
func FromRepo(repoPath string) (model.Node, error) {
	cmd := exec.Command("git", "-C", repoPath, "log", "-1", "--format=%H|%ae|%aI")
	out, err := cmd.Output()
	if err != nil {
		return model.Node{}, fmt.Errorf("git log: %w", err)
	}
	parts := strings.SplitN(strings.TrimSpace(string(out)), "|", 3)
	if len(parts) < 1 || parts[0] == "" {
		return model.Node{}, fmt.Errorf("git: empty log output")
	}
	remote, _ := exec.Command("git", "-C", repoPath, "remote", "get-url", "origin").Output()
	repo := strings.TrimSpace(string(remote))
	if repo == "" {
		repo = repoPath
	}
	meta := map[string]string{"repo": repo, "sha": strings.ToLower(parts[0])}
	if len(parts) > 1 {
		meta["author"] = parts[1]
	}
	if len(parts) > 2 {
		meta["timestamp"] = parts[2]
	}
	return model.Node{
		ID:       model.CommitID(parts[0]),
		Type:     model.NodeCommit,
		Metadata: meta,
	}, nil
}
