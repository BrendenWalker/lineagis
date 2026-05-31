package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/BrendenWalker/lineagis/internal/auth"
	"github.com/BrendenWalker/lineagis/internal/metadata"
	"github.com/BrendenWalker/lineagis/internal/signing"
)

// EvalPhase selects push-time vs verify-time rule sets (FR-POL-004).
type EvalPhase string

const (
	EvalPhasePush   EvalPhase = "push"
	EvalPhaseVerify EvalPhase = "verify"
)

// EvaluateResult is the deterministic outcome of policy evaluation (NFR-POL-001).
type EvaluateResult struct {
	Outcome       string
	Reasons       []PolicyReason
	PolicyID      *int64
	PolicyVersion *int
}

func parseEvalPhase(raw string) (EvalPhase, error) {
	switch EvalPhase(raw) {
	case EvalPhasePush, EvalPhaseVerify:
		return EvalPhase(raw), nil
	default:
		return "", fmt.Errorf("phase must be push or verify")
	}
}

// evaluateActivePolicy evaluates the namespace active policy for a digest and phase.
func evaluateActivePolicy(ctx context.Context, store *metadata.Store, namespaceID, digestID int64, phase EvalPhase, verifyOpts VerifyEvalOpts) (*EvaluateResult, error) {
	policy, err := store.GetActivePolicy(ctx, namespaceID)
	if errors.Is(err, metadata.ErrNotFound) {
		return &EvaluateResult{Outcome: "none"}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load active policy: %w", err)
	}

	ns, err := store.GetNamespaceByID(ctx, namespaceID)
	if err != nil {
		return nil, err
	}

	evalCtx := policyEvalContext{verifyByTag: verifyOpts.ByTag, github: verifyOpts.GitHub}
	reasons, err := evaluatePolicyDocument(ctx, store, ns.Name, ns.Config, policy.Document, digestID, phase, evalCtx)
	if err != nil {
		return nil, err
	}

	outcome := "pass"
	if len(reasons) > 0 {
		outcome = "fail"
	}
	policyID := policy.ID
	version := policy.Version
	return &EvaluateResult{
		Outcome:       outcome,
		Reasons:       reasons,
		PolicyID:      &policyID,
		PolicyVersion: &version,
	}, nil
}

func evaluatePolicyDocument(ctx context.Context, store *metadata.Store, ns string, nsConfig json.RawMessage, document json.RawMessage, digestID int64, phase EvalPhase, evalCtx policyEvalContext) ([]PolicyReason, error) {
	var doc policyDocument
	if err := json.Unmarshal(document, &doc); err != nil {
		return nil, nil
	}

	var reasons []PolicyReason
	for _, rule := range doc.Rules {
		if !ruleAppliesInPhase(rule, phase) {
			continue
		}
		ruleID := ruleIDFor(rule)
		switch {
		case ruleRequiresSignatures(rule):
			action := "verify"
			if phase == EvalPhasePush {
				action = "tagging"
			}
			if err := checkRequireSignatures(ctx, store, digestID, action); err != nil {
				if pf, ok := asPolicyFailure(err); ok {
					reasons = append(reasons, PolicyReason{Rule: pf.Rule, Message: pf.Hint})
				} else {
					return nil, err
				}
			}
		case ruleTrustedPublishers(rule):
			if err := checkTrustedPublishers(ctx, store, digestID, rule); err != nil {
				if pf, ok := asPolicyFailure(err); ok {
					reasons = append(reasons, PolicyReason{Rule: ruleID, Message: pf.Hint})
				} else {
					return nil, err
				}
			}
		case ruleRepositoryOwnership(rule):
			if err := checkRepositoryOwnership(ctx, store, ns, nsConfig, digestID, rule, evalCtx.github); err != nil {
				if pf, ok := asPolicyFailure(err); ok {
					reasons = append(reasons, PolicyReason{Rule: ruleID, Message: pf.Hint})
				} else {
					return nil, err
				}
			}
		case ruleRequireProvenance(rule):
			if err := checkRequireProvenance(ctx, store, digestID); err != nil {
				if pf, ok := asPolicyFailure(err); ok {
					reasons = append(reasons, PolicyReason{Rule: ruleID, Message: pf.Hint})
				} else {
					return nil, err
				}
			}
		case ruleRequireDigestOnVerify(rule):
			if phase == EvalPhaseVerify && evalCtx.verifyByTag {
				reasons = append(reasons, PolicyReason{
					Rule:    ruleID,
					Message: "verify by semver tag is not allowed; use a sha256:… digest reference",
				})
			}
		}
	}
	return reasons, nil
}

func ruleIDFor(rule policyRule) string {
	if id := strings.TrimSpace(rule.ID); id != "" {
		return id
	}
	return strings.TrimSpace(rule.Type)
}

func asPolicyFailure(err error) (PolicyFailure, bool) {
	var pf PolicyFailure
	if errors.As(err, &pf) {
		return pf, true
	}
	return PolicyFailure{}, false
}

// ruleAppliesInPhase returns whether a rule runs for the given evaluation phase (FR-POL-012).
func ruleAppliesInPhase(rule policyRule, phase EvalPhase) bool {
	if ruleRequireDigestOnVerify(rule) {
		return phase == EvalPhaseVerify
	}
	switch phase {
	case EvalPhasePush, EvalPhaseVerify:
		return ruleRequiresSignatures(rule) ||
			ruleTrustedPublishers(rule) ||
			ruleRepositoryOwnership(rule) ||
			ruleRequireProvenance(rule)
	default:
		return false
	}
}

func checkTrustedPublishers(ctx context.Context, store *metadata.Store, digestID int64, rule policyRule) error {
	cfg, err := parseTrustedPublishersConfig(rule.Config)
	if err != nil {
		return PolicyFailure{Rule: ruleIDFor(rule), Hint: err.Error()}
	}
	if len(cfg.Publishers) == 0 {
		return PolicyFailure{Rule: ruleIDFor(rule), Hint: "trusted-publishers policy has no allowlisted publishers"}
	}

	sigs, err := store.ListSignatures(ctx, digestID)
	if err != nil {
		return err
	}
	for _, sig := range sigs {
		bundle := signatureBundleBytes(sig)
		if len(bundle) == 0 {
			continue
		}
		pub, ok := signing.GitHubPublisherFromBundle(bundle)
		if !ok {
			continue
		}
		var id signing.Identity
		if pem := signing.LegacyBundleCertPEM(bundle); len(pem) > 0 {
			id, _ = signing.IdentityFromCertificatePEM(pem)
		}
		for _, allowed := range cfg.Publishers {
			if matchPublisher(pub, allowed, id) {
				return nil
			}
		}
	}
	return PolicyFailure{
		Rule: ruleIDFor(rule),
		Hint: "signer workflow identity is not in the trusted publishers allowlist",
	}
}

func matchPublisher(actual signing.GitHubPublisher, allowed trustedPublisher, identity signing.Identity) bool {
	repo := strings.TrimSpace(allowed.Repository)
	wf := strings.TrimSpace(allowed.Workflow)
	ref := strings.TrimSpace(allowed.Ref)
	issuer := strings.TrimSpace(allowed.Issuer)
	if repo != "" && !strings.EqualFold(actual.Repository, repo) {
		return false
	}
	if wf != "" && !strings.EqualFold(actual.Workflow, wf) {
		return false
	}
	if ref != "" && !matchRefPattern(actual.Ref, ref) {
		return false
	}
	if issuer != "" && !strings.EqualFold(identity.Issuer, issuer) {
		return false
	}
	if env := strings.TrimSpace(allowed.Environment); env != "" && !strings.EqualFold(actual.Environment, env) {
		return false
	}
	return repo != "" || wf != "" || ref != "" || issuer != "" || strings.TrimSpace(allowed.Environment) != ""
}

func matchRefPattern(actual, pattern string) bool {
	actual = strings.TrimSpace(actual)
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return true
	}
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(actual, prefix)
	}
	return strings.EqualFold(actual, pattern)
}

func parseTrustedPublishersConfig(raw json.RawMessage) (trustedPublishersConfig, error) {
	if len(raw) == 0 {
		return trustedPublishersConfig{}, nil
	}
	var cfg trustedPublishersConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return trustedPublishersConfig{}, fmt.Errorf("trusted-publishers config: %w", err)
	}
	return cfg, nil
}

func checkRequireProvenance(ctx context.Context, store *metadata.Store, digestID int64) error {
	prov, err := store.GetProvenanceByDigest(ctx, digestID)
	if errors.Is(err, metadata.ErrNotFound) {
		return PolicyFailure{
			Rule: "require-provenance",
			Hint: "provenance attestation is required",
		}
	}
	if err != nil {
		return err
	}
	if !prov.Verified {
		return PolicyFailure{
			Rule: "require-provenance",
			Hint: "provenance signature verification failed or attestation is invalid",
		}
	}
	return nil
}

func parseRepositoryOwnershipConfig(raw json.RawMessage) repositoryOwnershipConfig {
	if len(raw) == 0 {
		return repositoryOwnershipConfig{}
	}
	var cfg repositoryOwnershipConfig
	_ = json.Unmarshal(raw, &cfg)
	return cfg
}

func checkRepositoryOwnership(ctx context.Context, store *metadata.Store, ns string, nsConfig json.RawMessage, digestID int64, rule policyRule, github GitHubRepoChecker) error {
	expectedRepo, ok := auth.ExpectedRepository(ns, nsConfig)
	if !ok {
		return PolicyFailure{
			Rule: "repository-ownership",
			Hint: "namespace is not linked to a GitHub repository",
		}
	}
	prov, err := store.GetProvenanceByDigest(ctx, digestID)
	if errors.Is(err, metadata.ErrNotFound) {
		return PolicyFailure{
			Rule: "repository-ownership",
			Hint: "provenance attestation is required to verify repository ownership",
		}
	}
	if err != nil {
		return err
	}
	claimRepo := repositoryFromURI(prov.RepositoryURI)
	if claimRepo == "" {
		return PolicyFailure{
			Rule: "repository-ownership",
			Hint: "provenance does not include a repository URI",
		}
	}
	if !strings.EqualFold(claimRepo, expectedRepo) {
		return PolicyFailure{
			Rule: "repository-ownership",
			Hint: fmt.Sprintf("provenance repository %q does not match namespace repository %q", claimRepo, expectedRepo),
		}
	}

	cfg := parseRepositoryOwnershipConfig(rule.Config)
	if !cfg.VerifyWithGitHubAPI {
		return nil
	}
	if github == nil {
		return PolicyFailure{
			Rule: "repository-ownership",
			Hint: "GitHub API verification is required but LINEAGIS_GITHUB_TOKEN is not configured on the server",
		}
	}
	exists, err := github.RepositoryExists(ctx, expectedRepo)
	if err != nil {
		return PolicyFailure{
			Rule: "repository-ownership",
			Hint: fmt.Sprintf("GitHub API verification failed: %v", err),
		}
	}
	if !exists {
		return PolicyFailure{
			Rule: "repository-ownership",
			Hint: fmt.Sprintf("GitHub repository %q was not found", expectedRepo),
		}
	}
	return nil
}

func repositoryFromURI(uri string) string {
	uri = strings.TrimSpace(uri)
	uri = strings.TrimSuffix(uri, "/")
	if strings.HasPrefix(uri, "https://github.com/") {
		return strings.TrimPrefix(uri, "https://github.com/")
	}
	if strings.HasPrefix(uri, "http://github.com/") {
		return strings.TrimPrefix(uri, "http://github.com/")
	}
	return uri
}
