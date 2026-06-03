package sbom

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/BrendenWalker/lineagis/internal/core/model"
	"github.com/BrendenWalker/lineagis/internal/normalize/resolver"
)

type spdxDoc struct {
	SPDXVersion string        `json:"spdxVersion"`
	Packages    []spdxPackage `json:"packages"`
}

type spdxPackage struct {
	SPDXID       string       `json:"SPDXID"`
	Name         string       `json:"name"`
	VersionInfo  string       `json:"versionInfo"`
	ExternalRefs []spdxExtRef `json:"externalRefs"`
}

type spdxExtRef struct {
	ReferenceCategory string `json:"referenceCategory"`
	ReferenceType     string `json:"referenceType"`
	ReferenceLocator  string `json:"referenceLocator"`
}

// ParseSPDX parses SPDX JSON into nodes and depends_on edges (first package = root; rest = direct deps).
func ParseSPDX(data []byte) ([]model.Node, []model.Edge, error) {
	var doc spdxDoc
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, nil, fmt.Errorf("spdx: %w", err)
	}
	if err := checkSPDXVersion(doc.SPDXVersion); err != nil {
		return nil, nil, err
	}
	if len(doc.Packages) == 0 {
		return nil, nil, fmt.Errorf("spdx: no packages")
	}

	root := doc.Packages[0]
	rootID, rootNode, err := spdxPackageToArtifact(root)
	if err != nil {
		return nil, nil, err
	}
	var nodes []model.Node
	var edges []model.Edge
	nodes = append(nodes, rootNode)

	for i := 1; i < len(doc.Packages); i++ {
		p := doc.Packages[i]
		depID, depNode, err := spdxPackageToDependency(p)
		if err != nil {
			continue
		}
		nodes = append(nodes, depNode)
		edges = append(edges, model.Edge{From: rootID, To: depID, Type: model.EdgeDependsOn})
	}
	return nodes, edges, nil
}

func spdxPackageToArtifact(p spdxPackage) (string, model.Node, error) {
	hex := spdxSHA256(p)
	if hex == "" {
		return "", model.Node{}, fmt.Errorf("spdx: package %q missing sha256", p.Name)
	}
	id := model.ArtifactID(hex)
	meta := map[string]string{"digest": id, "name": p.Name}
	if p.VersionInfo != "" {
		meta["version"] = p.VersionInfo
	}
	if purl := spdxPURL(p); purl != "" {
		meta["purl"] = purl
	}
	return id, model.Node{ID: id, Type: model.NodeArtifact, Metadata: meta}, nil
}

func spdxPackageToDependency(p spdxPackage) (string, model.Node, error) {
	purl := spdxPURL(p)
	eco, name, ver := parsePURL(purl, p.Name, p.VersionInfo)
	id := model.DependencyID(eco, name, ver)
	meta := map[string]string{"ecosystem": eco, "name": name, "version": ver}
	if purl != "" {
		meta["purl"] = purl
	}
	return id, model.Node{ID: id, Type: model.NodeDependency, Metadata: meta}, nil
}

func spdxPURL(p spdxPackage) string {
	for _, ref := range p.ExternalRefs {
		if strings.EqualFold(ref.ReferenceType, "purl") {
			return ref.ReferenceLocator
		}
	}
	return ""
}

func spdxSHA256(p spdxPackage) string {
	for _, ref := range p.ExternalRefs {
		loc := strings.ToLower(ref.ReferenceLocator)
		if strings.Contains(loc, "sha256:") {
			return resolver.HexFromDigest(loc[strings.Index(loc, "sha256:"):])
		}
	}
	return ""
}
