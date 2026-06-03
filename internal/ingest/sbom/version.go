package sbom

import (
	"fmt"
	"strconv"
	"strings"
)

func checkCycloneDXSpecVersion(spec string) error {
	if spec == "" {
		return fmt.Errorf("cyclonedx: missing specVersion")
	}
	major, minor, ok := parseSpecVersion(spec)
	if !ok {
		return fmt.Errorf("cyclonedx: invalid specVersion %q", spec)
	}
	if major < 1 || (major == 1 && minor < 4) {
		return fmt.Errorf("cyclonedx: specVersion %q below minimum 1.4", spec)
	}
	return nil
}

func checkSPDXVersion(version string) error {
	if version == "" {
		return fmt.Errorf("spdx: missing spdxVersion")
	}
	// SPDX-2.3, SPDX-2.2, etc.
	v := strings.TrimPrefix(strings.ToUpper(version), "SPDX-")
	major, minor, ok := parseSpecVersion(v)
	if !ok {
		return fmt.Errorf("spdx: invalid spdxVersion %q", version)
	}
	if major < 2 || (major == 2 && minor < 2) {
		return fmt.Errorf("spdx: spdxVersion %q below minimum 2.2", version)
	}
	return nil
}

func parseSpecVersion(s string) (major, minor int, ok bool) {
	parts := strings.SplitN(strings.TrimSpace(s), ".", 3)
	if len(parts) < 2 {
		return 0, 0, false
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, false
	}
	minor, err = strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, false
	}
	return major, minor, true
}
