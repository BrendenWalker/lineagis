package api

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/BrendenWalker/verity/internal/metadata"
	"github.com/BrendenWalker/verity/internal/provenance"
	"github.com/BrendenWalker/verity/internal/signing"
)

type attestationEnvelope struct {
	Statement json.RawMessage `json:"statement"`
	Bundle    json.RawMessage `json:"bundle"`
}

func parseAttestationEnvelope(raw json.RawMessage) (attestationEnvelope, error) {
	var env attestationEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return attestationEnvelope{}, fmt.Errorf("invalid attestation envelope: %w", err)
	}
	if len(env.Statement) == 0 || len(env.Bundle) == 0 {
		return attestationEnvelope{}, fmt.Errorf("attestation envelope requires statement and bundle")
	}
	return env, nil
}

func verifyAttestationEnvelope(ctx context.Context, envelopeJSON json.RawMessage) (provenance.Statement, bool, error) {
	env, err := parseAttestationEnvelope(envelopeJSON)
	if err != nil {
		return provenance.Statement{}, false, err
	}
	stmt, err := provenance.ParseStatement(env.Statement)
	if err != nil {
		return provenance.Statement{}, false, err
	}
	cfg := signing.LoadConfig()
	if err := signing.VerifyManifestBundle(ctx, cfg, env.Statement, env.Bundle, signing.VerifyOptions{}); err != nil {
		return stmt, false, nil
	}
	return stmt, true, nil
}

func isProvenancePredicate(pt string) bool {
	pt = strings.ToLower(pt)
	return strings.Contains(pt, "provenance") || strings.Contains(pt, "slsaprovenance")
}

func isSBOMPredicate(pt string) bool {
	pt = strings.ToLower(pt)
	return strings.Contains(pt, "sbom") || strings.Contains(pt, "spdx") || strings.Contains(pt, "cyclonedx")
}

func attestationEnvelopeBytes(att metadata.Attestation) []byte {
	if len(att.EnvelopeJSON) > 0 {
		return att.EnvelopeJSON
	}
	return nil
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
