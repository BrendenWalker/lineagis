package api

import "context"

// PushPolicy evaluates push-time rules before SetTag (FR-API-007).
// MVP stub: allow all until policy engine lands (#43).
type PushPolicy interface {
	AllowSetTag(ctx context.Context, namespaceID, artifactID, digestID int64) error
}

// AllowAllPolicy is the M06 stub that permits every tag move.
type AllowAllPolicy struct{}

func (AllowAllPolicy) AllowSetTag(context.Context, int64, int64, int64) error {
	return nil
}
