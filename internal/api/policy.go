package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	policy, err := p.Store.GetActivePolicy(ctx, namespaceID)
	if errors.Is(err, metadata.ErrNotFound) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("load active policy: %w", err)
	}
	if !policyRequiresSignatures(policy.Document) {
		return nil
	}
	return checkRequireSignatures(ctx, p.Store, digestID, "tagging")
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
	policy, err := p.Store.GetActivePolicy(ctx, namespaceID)
	if errors.Is(err, metadata.ErrNotFound) {
		return &VerifyResult{Outcome: "none"}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load active policy: %w", err)
	}

	var reasons []PolicyReason
	if policyRequiresSignatures(policy.Document) {
		if err := checkRequireSignatures(ctx, p.Store, digestID, "verify"); err != nil {
			var pf PolicyFailure
			if errors.As(err, &pf) {
				reasons = append(reasons, PolicyReason{Rule: pf.Rule, Message: pf.Hint})
			} else {
				return nil, err
			}
		}
	}

	outcome := "pass"
	if len(reasons) > 0 {
		outcome = "fail"
	}
	policyID := policy.ID
	return &VerifyResult{
		Outcome:  outcome,
		Reasons:  reasons,
		PolicyID: &policyID,
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
	ID   string `json:"id"`
	Type string `json:"type"`
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
	for _, v := range []string{r.ID, r.Type} {
		switch strings.ToLower(strings.TrimSpace(v)) {
		case "require-signatures", "require-signature":
			return true
		}
	}
	return false
}
