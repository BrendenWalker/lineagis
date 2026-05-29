package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/BrendenWalker/verity/internal/metadata"
)

// PushPolicy evaluates push-time rules before SetTag (FR-API-007).
type PushPolicy interface {
	AllowSetTag(ctx context.Context, namespaceID, artifactID, digestID int64) error
}

// AllowAllPolicy permits every tag move (tests and explicit opt-out).
type AllowAllPolicy struct{}

func (AllowAllPolicy) AllowSetTag(context.Context, int64, int64, int64) error {
	return nil
}

// StorePushPolicy evaluates the namespace active policy against digest trust metadata.
type StorePushPolicy struct {
	Store *metadata.Store
}

// NewStorePushPolicy returns push-time policy evaluation backed by the metadata store.
func NewStorePushPolicy(store *metadata.Store) StorePushPolicy {
	return StorePushPolicy{Store: store}
}

func (p StorePushPolicy) AllowSetTag(ctx context.Context, namespaceID, _, digestID int64) error {
	result, err := evaluateActivePolicy(ctx, p.Store, namespaceID, digestID, EvalPhasePush)
	if err != nil {
		return err
	}
	if result.Outcome != "fail" || len(result.Reasons) == 0 {
		return nil
	}
	r := result.Reasons[0]
	return PolicyFailure{Rule: r.Rule, Hint: r.Message}
}

// VerifyPolicy evaluates verify-time rules for a digest (FR-POL-004).
type VerifyPolicy interface {
	Evaluate(ctx context.Context, namespaceID, digestID int64) (*VerifyResult, error)
}

// VerifyResult is the outcome of verify-time policy evaluation.
type VerifyResult struct {
	Outcome  string
	Reasons  []PolicyReason
	PolicyID *int64
}

// PolicyReason describes a single rule outcome for clients and audit.
type PolicyReason struct {
	Rule    string `json:"rule"`
	Message string `json:"message"`
}

// StoreVerifyPolicy evaluates the namespace active policy at verify time.
type StoreVerifyPolicy struct {
	Store *metadata.Store
}

// NewStoreVerifyPolicy returns verify-time policy evaluation backed by the metadata store.
func NewStoreVerifyPolicy(store *metadata.Store) StoreVerifyPolicy {
	return StoreVerifyPolicy{Store: store}
}

func (p StoreVerifyPolicy) Evaluate(ctx context.Context, namespaceID, digestID int64) (*VerifyResult, error) {
	result, err := evaluateActivePolicy(ctx, p.Store, namespaceID, digestID, EvalPhaseVerify)
	if err != nil {
		return nil, err
	}
	return &VerifyResult{
		Outcome:  result.Outcome,
		Reasons:  result.Reasons,
		PolicyID: result.PolicyID,
	}, nil
}

func checkRequireSignatures(ctx context.Context, store *metadata.Store, digestID int64, action string) error {
	sigs, err := store.ListSignatures(ctx, digestID)
	if err != nil {
		return fmt.Errorf("list signatures: %w", err)
	}
	if len(sigs) == 0 {
		return PolicyFailure{
			Rule: "require-signatures",
			Hint: fmt.Sprintf("digest has no signature; attach a Sigstore bundle before %s", action),
		}
	}
	return nil
}

// PolicyFailure is returned when a push-time rule rejects SetTag (FR-POL-009).
type PolicyFailure struct {
	Rule string
	Hint string
}

func (e PolicyFailure) Error() string {
	return e.Rule + ": " + e.Hint
}

type policyDocument struct {
	Rules []policyRule `json:"rules"`
}

type policyRule struct {
	ID     string          `json:"id"`
	Type   string          `json:"type"`
	Config json.RawMessage `json:"config,omitempty"`
}

type trustedPublishersConfig struct {
	Publishers []trustedPublisher `json:"publishers"`
}

type trustedPublisher struct {
	Repository  string `json:"repository"`
	Workflow    string `json:"workflow"`
	Ref         string `json:"ref,omitempty"`
	Environment string `json:"environment,omitempty"`
	Issuer      string `json:"issuer,omitempty"`
}

func validatePolicyDocument(document json.RawMessage) error {
	decoder := json.NewDecoder(bytes.NewReader(document))
	decoder.DisallowUnknownFields()

	var doc policyDocument
	if err := decoder.Decode(&doc); err != nil {
		return fmt.Errorf("policy document must be an object with optional rules array: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("policy document must contain a single JSON value")
	}

	for i, rule := range doc.Rules {
		id := strings.TrimSpace(rule.ID)
		typ := strings.TrimSpace(rule.Type)
		if id == "" && typ == "" {
			return fmt.Errorf("rules[%d] must include id or type", i)
		}
	}
	return nil
}

func policyRequiresSignatures(document json.RawMessage) bool {
	var doc policyDocument
	if err := json.Unmarshal(document, &doc); err != nil {
		return false
	}
	for _, r := range doc.Rules {
		if ruleRequiresSignatures(r) {
			return true
		}
	}
	return false
}

func ruleRequiresSignatures(r policyRule) bool {
	return ruleMatches(r, "require-signatures", "require-signature")
}

func ruleTrustedPublishers(r policyRule) bool {
	return ruleMatches(r, "trusted-publishers", "trusted-publisher")
}

func ruleRepositoryOwnership(r policyRule) bool {
	return ruleMatches(r, "repository-ownership", "repo-ownership")
}

func ruleRequireProvenance(r policyRule) bool {
	return ruleMatches(r, "require-provenance", "require-provenances")
}

func ruleMatches(r policyRule, names ...string) bool {
	for _, v := range []string{r.ID, r.Type} {
		v = strings.ToLower(strings.TrimSpace(v))
		for _, name := range names {
			if v == name {
				return true
			}
		}
	}
	return false
}
