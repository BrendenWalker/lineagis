package sbom_test

import (
	"testing"

	"github.com/BrendenWalker/lineagis/internal/ingest/sbom"
)

func TestCycloneDXRejectsLowSpecVersion(t *testing.T) {
	data := []byte(`{"bomFormat":"CycloneDX","specVersion":"1.3","metadata":{"component":{"name":"x","hashes":[{"alg":"SHA-256","content":"abc"}]}}}`)
	_, _, err := sbom.ParseCycloneDX(data)
	if err == nil {
		t.Fatal("expected error for specVersion 1.3")
	}
}

func TestSPDXRejectsLowVersion(t *testing.T) {
	data := []byte(`{"spdxVersion":"SPDX-2.1","packages":[{"name":"x","externalRefs":[{"referenceLocator":"sha256:abc"}]}]}`)
	_, _, err := sbom.ParseSPDX(data)
	if err == nil {
		t.Fatal("expected error for spdxVersion 2.1")
	}
}
