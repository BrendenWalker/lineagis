package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/BrendenWalker/verity/internal/metadata"
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
// Same digest, policy version, and phase always yield the same outcome and reasons.
func evaluateActivePolicy(ctx context.Context, store *metadata.Store, namespaceID, digestID int64, phase EvalPhase) (*EvaluateResult, error) {
	policy, err := store.GetActivePolicy(ctx, namespaceID)
	if errors.Is(err, metadata.ErrNotFound) {
		return &EvaluateResult{Outcome: "none"}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load active policy: %w", err)
	}

	reasons, err := evaluatePolicyDocument(ctx, store, policy.Document, digestID, phase)
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

func evaluatePolicyDocument(ctx context.Context, store *metadata.Store, document json.RawMessage, digestID int64, phase EvalPhase) ([]PolicyReason, error) {
	var doc policyDocument
	if err := json.Unmarshal(document, &doc); err != nil {
		return nil, nil
	}

	var reasons []PolicyReason
	for _, rule := range doc.Rules {
		if !ruleAppliesInPhase(rule, phase) {
			continue
		}
		if !ruleRequiresSignatures(rule) {
			continue
		}
		action := "verify"
		if phase == EvalPhasePush {
			action = "tagging"
		}
		if err := checkRequireSignatures(ctx, store, digestID, action); err != nil {
			var pf PolicyFailure
			if errors.As(err, &pf) {
				reasons = append(reasons, PolicyReason{Rule: pf.Rule, Message: pf.Hint})
				continue
			}
			return nil, err
		}
	}
	return reasons, nil
}

// ruleAppliesInPhase returns whether a rule runs for the given evaluation phase.
func ruleAppliesInPhase(rule policyRule, phase EvalPhase) bool {
	if phase == EvalPhaseVerify {
		return true
	}
	// Push-time (MVP): require-signatures and future push-scoped rules only.
	return ruleRequiresSignatures(rule)
}
