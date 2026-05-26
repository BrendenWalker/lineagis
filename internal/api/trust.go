package api

import (
	"context"
	"strings"

	"github.com/BrendenWalker/verity/internal/metadata"
)

type trustSignatures struct {
	Status string `json:"status"`
}

type trustPolicy struct {
	Status  string         `json:"status"`
	Reasons []PolicyReason `json:"reasons,omitempty"`
}

type trustAttestations struct {
	Provenance bool `json:"provenance"`
	SBOM       bool `json:"sbom"`
}

type trustStatusResponse struct {
	Namespace    string            `json:"namespace"`
	Artifact     string            `json:"artifact"`
	Digest       string            `json:"digest"`
	Overall      string            `json:"overall"`
	Signatures   trustSignatures   `json:"signatures"`
	Policy       trustPolicy       `json:"policy"`
	Attestations trustAttestations `json:"attestations"`
}

func (h *Handler) buildTrustStatus(ctx context.Context, namespaceID int64, ns, artifact string, d *metadata.Digest) (*trustStatusResponse, error) {
	sigs, err := h.Store.ListSignatures(ctx, d.ID)
	if err != nil {
		return nil, err
	}
	sigStatus, err := h.evaluateSignatures(ctx, ns, artifact, d, sigs)
	if err != nil {
		return nil, err
	}

	evaluator := h.verifyPolicy()
	policyEval, err := evaluator.Evaluate(ctx, namespaceID, d.ID)
	if err != nil {
		return nil, err
	}
	policyStatus := policyEval.Outcome

	atts, err := h.Store.ListAttestations(ctx, d.ID)
	if err != nil {
		return nil, err
	}
	var prov, sbom bool
	for _, att := range atts {
		pt := strings.ToLower(att.PredicateType)
		if strings.Contains(pt, "provenance") || strings.Contains(pt, "slsaprovenance") {
			prov = true
		}
		if strings.Contains(pt, "sbom") || strings.Contains(pt, "spdx") || strings.Contains(pt, "cyclonedx") {
			sbom = true
		}
	}

	overall := "pass"
	if sigStatus != "valid" || policyStatus == "fail" {
		overall = "fail"
	} else if policyStatus == "warn" {
		overall = "warn"
	}

	return &trustStatusResponse{
		Namespace: ns,
		Artifact:  artifact,
		Digest:    d.Digest,
		Overall:   overall,
		Signatures: trustSignatures{
			Status: sigStatus,
		},
		Policy: trustPolicy{
			Status:  policyStatus,
			Reasons: policyEval.Reasons,
		},
		Attestations: trustAttestations{
			Provenance: prov,
			SBOM:       sbom,
		},
	}, nil
}
