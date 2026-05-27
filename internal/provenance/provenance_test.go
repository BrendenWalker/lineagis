package provenance

import (
	"encoding/json"
	"testing"
)

func TestBuildSLSAStatement(t *testing.T) {
	t.Parallel()
	stmt, err := BuildSLSAStatement(BuildContext{
		ManifestDigest: "sha256:abc",
		RepositoryURI:  "https://github.com/acme/widget",
		CommitSHA:      "deadbeef",
		WorkflowName:   "release",
		WorkflowRef:    "refs/heads/main",
		RunID:          "99",
	})
	if err != nil {
		t.Fatal(err)
	}
	if stmt.PredicateType != PredicateSLSAProvenanceV1 {
		t.Fatalf("predicateType = %q", stmt.PredicateType)
	}
	fields := ParseFields(stmt)
	if fields.RepositoryURI != "https://github.com/acme/widget" {
		t.Fatalf("repository = %q", fields.RepositoryURI)
	}
	if fields.CommitSHA != "deadbeef" {
		t.Fatalf("commit = %q", fields.CommitSHA)
	}
	if fields.WorkflowName != "release" {
		t.Fatalf("workflow = %q", fields.WorkflowName)
	}
}

func TestSBOMPredicateType(t *testing.T) {
	t.Parallel()
	spdx := []byte(`{"spdxVersion":"SPDX-2.3"}`)
	pt, err := SBOMPredicateType(spdx)
	if err != nil || pt != PredicateSPDX {
		t.Fatalf("got %q err %v", pt, err)
	}
	cdx := []byte(`{"bomFormat":"CycloneDX","specVersion":"1.5"}`)
	pt, err = SBOMPredicateType(cdx)
	if err != nil || pt != PredicateCycloneDX {
		t.Fatalf("got %q err %v", pt, err)
	}
}

func TestParseStatementRoundTrip(t *testing.T) {
	t.Parallel()
	stmt, err := BuildSLSAStatement(BuildContext{ManifestDigest: "sha256:abc"})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := MarshalStatement(stmt)
	if err != nil {
		t.Fatal(err)
	}
	got, err := ParseStatement(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got.PredicateType != stmt.PredicateType {
		t.Fatalf("got %q", got.PredicateType)
	}
	if _, err := json.Marshal(got); err != nil {
		t.Fatal(err)
	}
}
