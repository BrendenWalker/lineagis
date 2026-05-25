package api

import (
	"context"
	"errors"
	"strings"

	"github.com/BrendenWalker/verity/internal/metadata"
)

type trustSignatures struct {
	Status string `json:"status"`
}

type trustPolicy struct {
	Status string `json:"status"`
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

func buildTrustStatus(ctx context.Context, store *metadata.Store, ns, artifact string, d *metadata.Digest) (*trustStatusResponse, error) {
	sigs, err := store.ListSignatures(ctx, d.ID)
	if err != nil {
		return nil, err
	}
	sigStatus := "missing"
	if len(sigs) > 0 {
		sigStatus = "valid"
	}

	policyStatus := "none"
	decision, err := store.LatestPolicyDecision(ctx, d.ID)
	if err != nil && !errors.Is(err, metadata.ErrNotFound) {
		return nil, err
	}
	if decision != nil {
		policyStatus = decision.Outcome
	}

	atts, err := store.ListAttestations(ctx, d.ID)
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
			Status: policyStatus,
		},
		Attestations: trustAttestations{
			Provenance: prov,
			SBOM:       sbom,
		},
	}, nil
}
