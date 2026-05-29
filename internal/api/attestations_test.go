package api

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/BrendenWalker/verity/internal/metadata"
	"github.com/BrendenWalker/verity/internal/provenance"
	"github.com/BrendenWalker/verity/internal/signing"
)

func TestVerifyAttestationEnvelope_validProvenance(t *testing.T) {
	t.Parallel()
	stmt, err := provenance.BuildSLSAStatement(provenance.BuildContext{
		ManifestDigest: "sha256:abc",
		RepositoryURI:  "https://github.com/acme/widget",
		CommitSHA:      "deadbeef",
	})
	if err != nil {
		t.Fatal(err)
	}
	statementJSON, err := provenance.MarshalStatement(stmt)
	if err != nil {
		t.Fatal(err)
	}
	bundle, _, err := signing.SignManifestForTest(statementJSON)
	if err != nil {
		t.Fatal(err)
	}
	envelope, err := json.Marshal(attestationEnvelope{Statement: statementJSON, Bundle: bundle})
	if err != nil {
		t.Fatal(err)
	}

	got, verified, err := verifyAttestationEnvelope(context.Background(), envelope)
	if err != nil {
		t.Fatal(err)
	}
	if !verified {
		t.Fatal("expected provenance verified")
	}
	if got.PredicateType != provenance.PredicateSLSAProvenanceV1 {
		t.Fatalf("predicate = %q", got.PredicateType)
	}
}

func TestVerifyAttestationEnvelope_tamperedStatement(t *testing.T) {
	t.Parallel()
	stmt, err := provenance.BuildSLSAStatement(provenance.BuildContext{ManifestDigest: "sha256:abc"})
	if err != nil {
		t.Fatal(err)
	}
	statementJSON, err := provenance.MarshalStatement(stmt)
	if err != nil {
		t.Fatal(err)
	}
	bundle, _, err := signing.SignManifestForTest(statementJSON)
	if err != nil {
		t.Fatal(err)
	}
	tampered := append([]byte(nil), statementJSON...)
	tampered[len(tampered)-2] ^= 0xff
	envelope, err := json.Marshal(attestationEnvelope{Statement: tampered, Bundle: bundle})
	if err != nil {
		t.Fatal(err)
	}

	_, verified, err := verifyAttestationEnvelope(context.Background(), envelope)
	if err != nil {
		t.Fatal(err)
	}
	if verified {
		t.Fatal("tampered statement must not verify (AC-PROV-003)")
	}
}

func TestVerifyAttestationEnvelope_tamperedBundle(t *testing.T) {
	t.Parallel()
	stmt, err := provenance.BuildSLSAStatement(provenance.BuildContext{ManifestDigest: "sha256:abc"})
	if err != nil {
		t.Fatal(err)
	}
	statementJSON, err := provenance.MarshalStatement(stmt)
	if err != nil {
		t.Fatal(err)
	}
	bundle, _, err := signing.SignManifestForTest(statementJSON)
	if err != nil {
		t.Fatal(err)
	}
	tampered := append([]byte(nil), bundle...)
	tampered[len(tampered)-2] ^= 0xff
	envelope, err := json.Marshal(attestationEnvelope{Statement: statementJSON, Bundle: tampered})
	if err != nil {
		t.Fatal(err)
	}

	_, verified, err := verifyAttestationEnvelope(context.Background(), envelope)
	if err != nil {
		t.Fatal(err)
	}
	if verified {
		t.Fatal("tampered bundle must not verify (AC-PROV-003)")
	}
}

func TestEvaluateAttestations_provenanceVerified(t *testing.T) {
	t.Parallel()
	stmt, err := provenance.BuildSLSAStatement(provenance.BuildContext{ManifestDigest: "sha256:abc"})
	if err != nil {
		t.Fatal(err)
	}
	statementJSON, err := provenance.MarshalStatement(stmt)
	if err != nil {
		t.Fatal(err)
	}
	bundle, _, err := signing.SignManifestForTest(statementJSON)
	if err != nil {
		t.Fatal(err)
	}
	envelope, err := json.Marshal(attestationEnvelope{Statement: statementJSON, Bundle: bundle})
	if err != nil {
		t.Fatal(err)
	}

	status := evaluateAttestations(context.Background(), []metadata.Attestation{{
		PredicateType: provenance.PredicateSLSAProvenanceV1,
		EnvelopeJSON:  envelope,
	}})
	if !status.Provenance || !status.ProvenanceVerified {
		t.Fatalf("got %+v", status)
	}
}
