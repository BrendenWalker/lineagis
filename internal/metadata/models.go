package metadata

import (
	"encoding/json"
	"time"
)

// Namespace is a top-level isolation boundary.
type Namespace struct {
	ID        int64
	Name      string
	Config    json.RawMessage
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Artifact is a logical software unit within a namespace.
type Artifact struct {
	ID          int64
	NamespaceID int64
	Name        string
	CreatedAt   time.Time
}

// Digest is an immutable content address for a manifest.
type Digest struct {
	ID         int64
	Digest     string
	ArtifactID int64
	MediaType  *string
	SizeBytes  *int64
	CreatedAt  time.Time
}

// Tag maps a semver or label to a digest for an artifact.
type Tag struct {
	ID         int64
	ArtifactID int64
	Name       string
	DigestID   int64
	UpdatedAt  time.Time
}

// TagEvent records a tag move in audit history.
type TagEvent struct {
	ID           int64
	TagID        int64
	FromDigestID *int64
	ToDigestID   int64
	Actor        *string
	CreatedAt    time.Time
}

// Signature is a Sigstore bundle reference for a digest.
type Signature struct {
	ID         int64
	DigestID   int64
	BundleRef  *string
	BundleJSON json.RawMessage
	Issuer     *string
	Subject    *string
	CreatedAt  time.Time
}

// Attestation is an in-toto statement index row for a digest.
type Attestation struct {
	ID             int64
	DigestID       int64
	PredicateType  string
	EnvelopeRef    *string
	EnvelopeDigest *string
	CreatedAt      time.Time
}

// Policy is a versioned rule set for a namespace.
type Policy struct {
	ID          int64
	NamespaceID int64
	Version     int
	Document    json.RawMessage
	IsActive    bool
	CreatedAt   time.Time
}

// PolicyDecision is the outcome of evaluating a policy for a digest.
type PolicyDecision struct {
	ID          int64
	DigestID    int64
	PolicyID    int64
	Outcome     string
	Reasons     json.RawMessage
	EvaluatedAt time.Time
}

// AuditEvent is an append-only audit log entry.
type AuditEvent struct {
	ID           int64
	NamespaceID  *int64
	EventType    string
	Actor        *string
	ResourceType *string
	ResourceID   *string
	Payload      json.RawMessage
	CreatedAt    time.Time
}
