package api

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/BrendenWalker/verity/internal/metadata"
	"github.com/BrendenWalker/verity/internal/signing"
)

type trustSignatures struct {
	Status string `json:"status"`
}

type trustPolicy struct {
	Status  string         `json:"status"`
	Reasons []PolicyReason `json:"reasons,omitempty"`
}

type trustAttestations struct {
	Provenance         bool   `json:"provenance"`
	SBOM               bool   `json:"sbom"`
	ProvenanceVerified bool   `json:"provenance_verified"`
	Repository         string `json:"repository,omitempty"`
	Commit             string `json:"commit,omitempty"`
	Workflow           string `json:"workflow,omitempty"`
	WorkflowRef        string `json:"workflow_ref,omitempty"`
	RunID              string `json:"run_id,omitempty"`
}

type trustStatusResponse struct {
	Namespace       string            `json:"namespace"`
	Artifact        string            `json:"artifact"`
	Digest          string            `json:"digest"`
	Overall         string            `json:"overall"`
	Signatures      trustSignatures   `json:"signatures"`
	Policy          trustPolicy       `json:"policy"`
	Attestations    trustAttestations `json:"attestations"`
	ConfiguredRules []string          `json:"configured_rules,omitempty"`
	Signer          *trustSigner      `json:"signer,omitempty"`
}

type trustSigner struct {
	Repository string `json:"repository,omitempty"`
	Workflow   string `json:"workflow,omitempty"`
	Ref        string `json:"ref,omitempty"`
	Issuer     string `json:"issuer,omitempty"`
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
	attStatus := evaluateAttestations(ctx, atts)

	provRec, _ := h.Store.GetProvenanceByDigest(ctx, d.ID)
	if provRec != nil {
		attStatus.Repository = provRec.RepositoryURI
		if provRec.CommitSHA != nil {
			attStatus.Commit = *provRec.CommitSHA
		}
		if provRec.WorkflowName != nil {
			attStatus.Workflow = *provRec.WorkflowName
		}
		if provRec.WorkflowRef != nil {
			attStatus.WorkflowRef = *provRec.WorkflowRef
		}
		if provRec.RunID != nil {
			attStatus.RunID = *provRec.RunID
		}
		attStatus.ProvenanceVerified = provRec.Verified
	}

	overall := "pass"
	if sigStatus != "valid" || policyStatus == "fail" {
		overall = "fail"
	} else if policyStatus == "warn" {
		overall = "warn"
	}

	configured := configuredRulesFromPolicy(ctx, h.Store, namespaceID)
	signer := signerFromSignatures(sigs)

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
		Attestations:    attStatus,
		ConfiguredRules: configured,
		Signer:          signer,
	}, nil
}

func configuredRulesFromPolicy(ctx context.Context, store *metadata.Store, namespaceID int64) []string {
	policy, err := store.GetActivePolicy(ctx, namespaceID)
	if err != nil {
		return nil
	}
	var doc policyDocument
	if err := json.Unmarshal(policy.Document, &doc); err != nil {
		return nil
	}
	out := make([]string, 0, len(doc.Rules))
	for _, rule := range doc.Rules {
		id := ruleIDFor(rule)
		if id != "" {
			out = append(out, id)
		}
	}
	return out
}

func signerFromSignatures(sigs []metadata.Signature) *trustSigner {
	for _, sig := range sigs {
		bundle := signatureBundleBytes(sig)
		if len(bundle) == 0 {
			continue
		}
		pub, ok := signing.GitHubPublisherFromBundle(bundle)
		if !ok {
			continue
		}
		ts := &trustSigner{
			Repository: pub.Repository,
			Workflow:   pub.Workflow,
			Ref:        pub.Ref,
		}
		if pem := signing.LegacyBundleCertPEM(bundle); len(pem) > 0 {
			if id, err := signing.IdentityFromCertificatePEM(pem); err == nil {
				ts.Issuer = id.Issuer
			}
		}
		return ts
	}
	return nil
}

func evaluateAttestations(ctx context.Context, atts []metadata.Attestation) trustAttestations {
	var out trustAttestations
	for _, att := range atts {
		pt := strings.ToLower(att.PredicateType)
		if isProvenancePredicate(pt) {
			out.Provenance = true
			if raw := attestationEnvelopeBytes(att); len(raw) > 0 {
				if _, verified, err := verifyAttestationEnvelope(ctx, raw); err == nil && verified {
					out.ProvenanceVerified = true
				}
			}
		}
		if isSBOMPredicate(pt) {
			out.SBOM = true
		}
	}
	return out
}
