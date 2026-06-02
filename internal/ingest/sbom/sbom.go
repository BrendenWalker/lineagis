package sbom

import (
	"fmt"
	"os"

	"github.com/BrendenWalker/lineagis/internal/core/model"
)

// ParseFile reads and parses an SBOM file.
func ParseFile(path string) ([]model.Node, []model.Edge, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}
	format, err := DetectFormatFromFile(data)
	if err != nil {
		return nil, nil, err
	}
	switch format {
	case FormatCycloneDX:
		return ParseCycloneDX(data)
	case FormatSPDX:
		return ParseSPDX(data)
	default:
		return nil, nil, fmt.Errorf("sbom: unknown format")
	}
}
