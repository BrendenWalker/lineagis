package sbom

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/BrendenWalker/lineagis/internal/core/model"
	"github.com/BrendenWalker/lineagis/internal/normalize/resolver"
)

type cycloneDXBOM struct {
	BOMFormat   string               `json:"bomFormat"`
	SpecVersion string               `json:"specVersion"`
	Metadata    cycloneDXMetadata    `json:"metadata"`
	Components  []cycloneDXComponent `json:"components"`
}

type cycloneDXMetadata struct {
	Component cycloneDXComponent `json:"component"`
}

type cycloneDXComponent struct {
	Type    string          `json:"type"`
	Name    string          `json:"name"`
	Version string          `json:"version"`
	PURL    string          `json:"purl"`
	Hashes  []cycloneDXHash `json:"hashes"`
	BOMRef  string          `json:"bom-ref"`
}

type cycloneDXHash struct {
	Alg     string `json:"alg"`
	Content string `json:"content"`
}

// ParseCycloneDX parses CycloneDX JSON into nodes and depends_on edges.
func ParseCycloneDX(data []byte) ([]model.Node, []model.Edge, error) {
	var bom cycloneDXBOM
	if err := json.Unmarshal(data, &bom); err != nil {
		return nil, nil, fmt.Errorf("cyclonedx: %w", err)
	}
	if bom.BOMFormat != "CycloneDX" {
		return nil, nil, fmt.Errorf("cyclonedx: missing bomFormat")
	}

	var root cycloneDXComponent
	if bom.Metadata.Component.Name != "" || len(bom.Metadata.Component.Hashes) > 0 {
		root = bom.Metadata.Component
	} else if len(bom.Components) > 0 {
		root = bom.Components[0]
	} else {
		return nil, nil, fmt.Errorf("cyclonedx: no root component")
	}

	rootID, rootNode, err := componentToArtifact(root)
	if err != nil {
		return nil, nil, err
	}
	var nodes []model.Node
	var edges []model.Edge
	nodes = append(nodes, rootNode)

	for _, comp := range bom.Components {
		if comp.BOMRef == root.BOMRef && comp.Name == root.Name {
			continue
		}
		depID, depNode, err := componentToDependency(comp)
		if err != nil {
			continue
		}
		nodes = append(nodes, depNode)
		edges = append(edges, model.Edge{From: rootID, To: depID, Type: model.EdgeDependsOn})
	}
	return nodes, edges, nil
}

func componentToArtifact(c cycloneDXComponent) (string, model.Node, error) {
	hex := sha256FromHashes(c.Hashes)
	if hex == "" {
		return "", model.Node{}, fmt.Errorf("cyclonedx: component %q missing sha256 hash", c.Name)
	}
	id := model.ArtifactID(hex)
	meta := map[string]string{
		"digest": id,
		"name":   c.Name,
	}
	if c.Version != "" {
		meta["version"] = c.Version
	}
	if c.PURL != "" {
		meta["purl"] = c.PURL
	}
	if c.Type != "" {
		meta["type"] = c.Type
	}
	return id, model.Node{ID: id, Type: model.NodeArtifact, Metadata: meta}, nil
}

func componentToDependency(c cycloneDXComponent) (string, model.Node, error) {
	eco, name, ver := parsePURL(c.PURL, c.Name, c.Version)
	id := model.DependencyID(eco, name, ver)
	meta := map[string]string{"ecosystem": eco, "name": name, "version": ver}
	if c.PURL != "" {
		meta["purl"] = c.PURL
	}
	return id, model.Node{ID: id, Type: model.NodeDependency, Metadata: meta}, nil
}

func sha256FromHashes(hashes []cycloneDXHash) string {
	for _, h := range hashes {
		if strings.EqualFold(h.Alg, "SHA-256") && h.Content != "" {
			return resolver.HexFromDigest(h.Content)
		}
	}
	return ""
}

func parsePURL(purl, name, version string) (eco, n, ver string) {
	ver = version
	n = name
	eco = "generic"
	if purl == "" {
		return eco, n, ver
	}
	// pkg:npm/lodash@4.17.21
	p := strings.TrimPrefix(purl, "pkg:")
	parts := strings.SplitN(p, "/", 3)
	if len(parts) >= 1 {
		eco = parts[0]
	}
	if len(parts) >= 2 {
		n = parts[1]
		if idx := strings.LastIndex(n, "@"); idx > 0 {
			ver = n[idx+1:]
			n = n[:idx]
		}
	}
	if len(parts) >= 3 && ver == "" {
		rest := parts[2]
		if idx := strings.LastIndex(rest, "@"); idx >= 0 {
			ver = rest[idx+1:]
		}
	}
	if ver == "" {
		ver = "unknown"
	}
	return eco, n, ver
}
