package sbom

import (
	"bytes"
	"fmt"
	"strings"
)

// Format identifies SBOM encoding.
type Format int

const (
	FormatUnknown Format = iota
	FormatCycloneDX
	FormatSPDX
)

// DetectFormat sniffs JSON SBOM type from content.
func DetectFormat(data []byte) (Format, error) {
	trim := strings.TrimSpace(string(data))
	if trim == "" {
		return FormatUnknown, fmt.Errorf("sbom: empty file")
	}
	if strings.Contains(trim, `"spdxVersion"`) {
		return FormatSPDX, nil
	}
	if strings.Contains(trim, `"bomFormat"`) && strings.Contains(trim, "CycloneDX") {
		return FormatCycloneDX, nil
	}
	return FormatUnknown, fmt.Errorf("sbom: unsupported format (want CycloneDX or SPDX JSON)")
}

// DetectFormatFromFile reads path and detects format.
func DetectFormatFromFile(data []byte) (Format, error) {
	if bytes.HasPrefix(data, []byte{0xEF, 0xBB, 0xBF}) {
		data = data[3:]
	}
	return DetectFormat(data)
}
