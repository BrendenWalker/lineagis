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
	sigs, err := p.Store.ListSignatures(ctx, digestID)
	if err != nil {
		return fmt.Errorf("list signatures: %w", err)
	}
	if len(sigs) == 0 {
		return PolicyFailure{
			Rule: "require-signatures",
			Hint: "digest has no signature; attach a Sigstore bundle before tagging",
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
