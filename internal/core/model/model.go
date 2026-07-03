package model

import (
	"fmt"
	"path/filepath"
	"strings"
)

// NodeType identifies a lineage graph node kind.
type NodeType string

const (
	NodeCommit     NodeType = "commit"
	NodeBuild      NodeType = "build"
	NodeArtifact   NodeType = "artifact"
	NodeDependency NodeType = "dependency"
	NodeModule     NodeType = "module"
	NodePackage    NodeType = "package"
	NodeFile       NodeType = "file"
	NodeSymbol     NodeType = "symbol"
)

// EdgeType identifies a directed lineage edge.
type EdgeType string

const (
	EdgeProducedBy EdgeType = "produced_by"
	EdgeBuiltFrom  EdgeType = "built_from"
	EdgeDependsOn  EdgeType = "depends_on"
	EdgeContains   EdgeType = "contains"
	EdgeImports    EdgeType = "imports"
)

// ProvenanceEdgeTypes are checked for cycles among commit/build/artifact nodes.
var ProvenanceEdgeTypes = []EdgeType{EdgeProducedBy, EdgeBuiltFrom}

// Node is a typed vertex in the lineage DAG.
type Node struct {
	ID       string            `json:"id"`
	Type     NodeType          `json:"type"`
	Metadata map[string]string `json:"metadata"`
}

// Edge is a directed link between two nodes.
type Edge struct {
	From     string            `json:"from"`
	To       string            `json:"to"`
	Type     EdgeType          `json:"type"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// GraphSnapshot is the on-disk graph envelope (lineage-graph/v1 or v2).
type GraphSnapshot struct {
	SchemaVersion string   `json:"schema_version"`
	Domains       []string `json:"domains,omitempty"`
	Nodes         []Node   `json:"nodes"`
	Edges         []Edge   `json:"edges"`
}

const (
	SchemaGraphV1 = "lineage-graph/v1"
	SchemaGraphV2 = "lineage-graph/v2"
	DomainProvenance = "provenance"
	DomainCode       = "code"
)

// CommitID returns canonical commit node ID.
func CommitID(sha string) string {
	return "commit:" + strings.ToLower(strings.TrimSpace(sha))
}

// BuildID returns canonical build node ID.
func BuildID(id string) string {
	return "build:" + strings.TrimSpace(id)
}

// ArtifactID returns canonical artifact node ID from a sha256 hex digest.
func ArtifactID(digest string) string {
	d := strings.TrimSpace(digest)
	d = strings.TrimPrefix(strings.ToLower(d), "sha256:")
	return "artifact:sha256:" + d
}

// DependencyID returns canonical dependency node ID.
func DependencyID(ecosystem, name, version string) string {
	return fmt.Sprintf("dependency:%s:%s@%s",
		strings.TrimSpace(ecosystem),
		strings.TrimSpace(name),
		strings.TrimSpace(version),
	)
}

// ModuleID returns canonical module node ID.
func ModuleID(path string) string {
	return "module:" + strings.TrimSpace(path)
}

// PackageID returns canonical package node ID.
func PackageID(importPath string) string {
	return "package:" + strings.TrimSpace(importPath)
}

// FileID returns canonical file node ID from a repo-relative path.
func FileID(repoRelPath string) string {
	p := filepath.ToSlash(strings.TrimSpace(repoRelPath))
	p = strings.TrimPrefix(p, "./")
	return "file:" + p
}

// SymbolID returns canonical symbol node ID.
func SymbolID(importPath, name string) string {
	return "symbol:" + strings.TrimSpace(importPath) + "#" + strings.TrimSpace(name)
}

// ParseRef resolves CLI refs like artifact@sha256:abc or artifact:artifact:sha256:abc.
func ParseRef(ref string) (string, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "", fmt.Errorf("empty ref")
	}
	const prefix = "artifact@"
	if strings.HasPrefix(ref, prefix) {
		digest := strings.TrimPrefix(ref, prefix)
		return ArtifactID(digest), nil
	}
	if strings.HasPrefix(ref, "artifact:") {
		return ref, nil
	}
	if strings.HasPrefix(ref, "dependency:") || strings.HasPrefix(ref, "build:") || strings.HasPrefix(ref, "commit:") {
		return ref, nil
	}
	return "", fmt.Errorf("unknown ref format %q (use artifact@sha256:… or artifact:<id>)", ref)
}

// IsProvenanceNode returns true for commit, build, or artifact types.
func IsProvenanceNode(t NodeType) bool {
	switch t {
	case NodeCommit, NodeBuild, NodeArtifact:
		return true
	default:
		return false
	}
}

// IsProvenanceEdge returns true for edges that participate in provenance cycle checks.
func IsProvenanceEdge(t EdgeType) bool {
	return t == EdgeProducedBy || t == EdgeBuiltFrom
}

// IsProvenanceNodeType returns true for v1.0 supply-chain node kinds.
func IsProvenanceNodeType(t NodeType) bool {
	switch t {
	case NodeCommit, NodeBuild, NodeArtifact, NodeDependency:
		return true
	default:
		return false
	}
}

// IsCodeNodeType returns true for repository self-analysis node kinds.
func IsCodeNodeType(t NodeType) bool {
	switch t {
	case NodeModule, NodePackage, NodeFile, NodeSymbol:
		return true
	default:
		return false
	}
}
