package metadata

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Store provides PostgreSQL-backed metadata persistence.
type Store struct {
	pool *pgxpool.Pool
}

// NewStore returns a Store using the given connection pool.
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// CreateNamespace inserts a namespace or returns the existing row by name.
func (s *Store) CreateNamespace(ctx context.Context, name string, config json.RawMessage) (*Namespace, error) {
	if config == nil {
		config = json.RawMessage(`{}`)
	}
	var ns Namespace
	err := s.pool.QueryRow(ctx, `
		INSERT INTO namespaces (name, config)
		VALUES ($1, $2)
		ON CONFLICT (name) DO UPDATE SET updated_at = namespaces.updated_at
		RETURNING id, name, config, created_at, updated_at
	`, name, config).Scan(&ns.ID, &ns.Name, &ns.Config, &ns.CreatedAt, &ns.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create namespace: %w", err)
	}
	return &ns, nil
}

// GetNamespaceByName returns a namespace by name.
func (s *Store) GetNamespaceByName(ctx context.Context, name string) (*Namespace, error) {
	var ns Namespace
	err := s.pool.QueryRow(ctx, `
		SELECT id, name, config, created_at, updated_at
		FROM namespaces WHERE name = $1
	`, name).Scan(&ns.ID, &ns.Name, &ns.Config, &ns.CreatedAt, &ns.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get namespace: %w", err)
	}
	return &ns, nil
}

// RegisterArtifact creates an artifact under a namespace.
func (s *Store) RegisterArtifact(ctx context.Context, namespaceID int64, name string) (*Artifact, error) {
	var a Artifact
	err := s.pool.QueryRow(ctx, `
		INSERT INTO artifacts (namespace_id, name)
		VALUES ($1, $2)
		ON CONFLICT (namespace_id, name) DO UPDATE SET name = EXCLUDED.name
		RETURNING id, namespace_id, name, created_at
	`, namespaceID, name).Scan(&a.ID, &a.NamespaceID, &a.Name, &a.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("register artifact: %w", err)
	}
	return &a, nil
}

// GetArtifact returns an artifact by namespace and name.
func (s *Store) GetArtifact(ctx context.Context, namespaceID int64, name string) (*Artifact, error) {
	var a Artifact
	err := s.pool.QueryRow(ctx, `
		SELECT id, namespace_id, name, created_at
		FROM artifacts WHERE namespace_id = $1 AND name = $2
	`, namespaceID, name).Scan(&a.ID, &a.NamespaceID, &a.Name, &a.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get artifact: %w", err)
	}
	return &a, nil
}

// RegisterDigest records a manifest digest for an artifact (idempotent on digest string).
func (s *Store) RegisterDigest(ctx context.Context, artifactID int64, digest string, mediaType *string, sizeBytes *int64) (*Digest, error) {
	var d Digest
	err := s.pool.QueryRow(ctx, `
		INSERT INTO digests (digest, artifact_id, media_type, size_bytes)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (digest) DO NOTHING
		RETURNING id, digest, artifact_id, media_type, size_bytes, created_at
	`, digest, artifactID, mediaType, sizeBytes).Scan(
		&d.ID, &d.Digest, &d.ArtifactID, &d.MediaType, &d.SizeBytes, &d.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		existing, getErr := s.GetDigestByString(ctx, digest)
		if getErr != nil {
			return nil, getErr
		}
		if existing.ArtifactID != artifactID {
			return nil, ErrDigestWrongArtifact
		}
		return existing, nil
	}
	if err != nil {
		return nil, fmt.Errorf("register digest: %w", err)
	}
	return &d, nil
}

// GetDigestByString returns a digest by its immutable address.
func (s *Store) GetDigestByString(ctx context.Context, digest string) (*Digest, error) {
	var d Digest
	err := s.pool.QueryRow(ctx, `
		SELECT id, digest, artifact_id, media_type, size_bytes, created_at
		FROM digests WHERE digest = $1
	`, digest).Scan(&d.ID, &d.Digest, &d.ArtifactID, &d.MediaType, &d.SizeBytes, &d.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get digest: %w", err)
	}
	return &d, nil
}

// GetDigestByID returns a digest by primary key.
func (s *Store) GetDigestByID(ctx context.Context, id int64) (*Digest, error) {
	var d Digest
	err := s.pool.QueryRow(ctx, `
		SELECT id, digest, artifact_id, media_type, size_bytes, created_at
		FROM digests WHERE id = $1
	`, id).Scan(&d.ID, &d.Digest, &d.ArtifactID, &d.MediaType, &d.SizeBytes, &d.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get digest by id: %w", err)
	}
	return &d, nil
}

// SetTag maps a tag to a digest, recording audit history when the tag moves.
func (s *Store) SetTag(ctx context.Context, artifactID int64, tagName string, digestID int64, actor *string) (*Tag, error) {
	d, err := s.GetDigestByID(ctx, digestID)
	if err != nil {
		return nil, err
	}
	if d.ArtifactID != artifactID {
		return nil, ErrDigestWrongArtifact
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var tag Tag
	var fromDigestID *int64
	err = tx.QueryRow(ctx, `
		SELECT id, artifact_id, name, digest_id, updated_at
		FROM tags WHERE artifact_id = $1 AND name = $2
	`, artifactID, tagName).Scan(&tag.ID, &tag.ArtifactID, &tag.Name, &tag.DigestID, &tag.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		err = tx.QueryRow(ctx, `
			INSERT INTO tags (artifact_id, name, digest_id)
			VALUES ($1, $2, $3)
			RETURNING id, artifact_id, name, digest_id, updated_at
		`, artifactID, tagName, digestID).Scan(
			&tag.ID, &tag.ArtifactID, &tag.Name, &tag.DigestID, &tag.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("insert tag: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("lookup tag: %w", err)
	} else {
		if tag.DigestID == digestID {
			if err := tx.Commit(ctx); err != nil {
				return nil, fmt.Errorf("commit tx: %w", err)
			}
			return &tag, nil
		}
		fromDigestID = &tag.DigestID
		err = tx.QueryRow(ctx, `
			UPDATE tags SET digest_id = $1, updated_at = now()
			WHERE id = $2
			RETURNING id, artifact_id, name, digest_id, updated_at
		`, digestID, tag.ID).Scan(&tag.ID, &tag.ArtifactID, &tag.Name, &tag.DigestID, &tag.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("update tag: %w", err)
		}
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO tag_events (tag_id, from_digest_id, to_digest_id, actor)
		VALUES ($1, $2, $3, $4)
	`, tag.ID, fromDigestID, digestID, actor)
	if err != nil {
		return nil, fmt.Errorf("insert tag event: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}
	return &tag, nil
}

// GetTag returns the current tag mapping for an artifact.
func (s *Store) GetTag(ctx context.Context, artifactID int64, tagName string) (*Tag, error) {
	var tag Tag
	err := s.pool.QueryRow(ctx, `
		SELECT id, artifact_id, name, digest_id, updated_at
		FROM tags WHERE artifact_id = $1 AND name = $2
	`, artifactID, tagName).Scan(&tag.ID, &tag.ArtifactID, &tag.Name, &tag.DigestID, &tag.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get tag: %w", err)
	}
	return &tag, nil
}

// ListTagEvents returns tag move history for a tag.
func (s *Store) ListTagEvents(ctx context.Context, tagID int64) ([]TagEvent, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, tag_id, from_digest_id, to_digest_id, actor, created_at
		FROM tag_events WHERE tag_id = $1 ORDER BY created_at ASC
	`, tagID)
	if err != nil {
		return nil, fmt.Errorf("list tag events: %w", err)
	}
	defer rows.Close()

	var events []TagEvent
	for rows.Next() {
		var e TagEvent
		if err := rows.Scan(&e.ID, &e.TagID, &e.FromDigestID, &e.ToDigestID, &e.Actor, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan tag event: %w", err)
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

// AttachSignature stores a signature reference for a digest.
func (s *Store) AttachSignature(ctx context.Context, digestID int64, bundleRef *string, bundleJSON json.RawMessage, issuer, subject *string) (*Signature, error) {
	if _, err := s.GetDigestByID(ctx, digestID); err != nil {
		return nil, err
	}
	var sig Signature
	err := s.pool.QueryRow(ctx, `
		INSERT INTO signatures (digest_id, bundle_ref, bundle_json, issuer, subject)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, digest_id, bundle_ref, bundle_json, issuer, subject, created_at
	`, digestID, bundleRef, bundleJSON, issuer, subject).Scan(
		&sig.ID, &sig.DigestID, &sig.BundleRef, &sig.BundleJSON, &sig.Issuer, &sig.Subject, &sig.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("attach signature: %w", err)
	}
	return &sig, nil
}

// ListSignatures returns signatures for a digest.
func (s *Store) ListSignatures(ctx context.Context, digestID int64) ([]Signature, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, digest_id, bundle_ref, bundle_json, issuer, subject, created_at
		FROM signatures WHERE digest_id = $1 ORDER BY created_at ASC
	`, digestID)
	if err != nil {
		return nil, fmt.Errorf("list signatures: %w", err)
	}
	defer rows.Close()

	var sigs []Signature
	for rows.Next() {
		var sig Signature
		if err := rows.Scan(&sig.ID, &sig.DigestID, &sig.BundleRef, &sig.BundleJSON, &sig.Issuer, &sig.Subject, &sig.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan signature: %w", err)
		}
		sigs = append(sigs, sig)
	}
	return sigs, rows.Err()
}

// AttachAttestation stores an attestation index row for a digest.
func (s *Store) AttachAttestation(ctx context.Context, digestID int64, predicateType string, envelopeRef, envelopeDigest *string) (*Attestation, error) {
	if _, err := s.GetDigestByID(ctx, digestID); err != nil {
		return nil, err
	}
	var att Attestation
	err := s.pool.QueryRow(ctx, `
		INSERT INTO attestations (digest_id, predicate_type, envelope_ref, envelope_digest)
		VALUES ($1, $2, $3, $4)
		RETURNING id, digest_id, predicate_type, envelope_ref, envelope_digest, created_at
	`, digestID, predicateType, envelopeRef, envelopeDigest).Scan(
		&att.ID, &att.DigestID, &att.PredicateType, &att.EnvelopeRef, &att.EnvelopeDigest, &att.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("attach attestation: %w", err)
	}
	return &att, nil
}

// PutPolicy creates a new policy version, deactivates prior versions, and audit-logs the change.
func (s *Store) PutPolicy(ctx context.Context, namespaceID int64, document json.RawMessage, actor *string) (*Policy, error) {
	if document == nil {
		document = json.RawMessage(`{}`)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var nextVersion int
	err = tx.QueryRow(ctx, `
		SELECT COALESCE(MAX(version), 0) + 1 FROM policies WHERE namespace_id = $1
	`, namespaceID).Scan(&nextVersion)
	if err != nil {
		return nil, fmt.Errorf("next policy version: %w", err)
	}

	_, err = tx.Exec(ctx, `
		UPDATE policies SET is_active = false WHERE namespace_id = $1 AND is_active = true
	`, namespaceID)
	if err != nil {
		return nil, fmt.Errorf("deactivate policies: %w", err)
	}

	var p Policy
	err = tx.QueryRow(ctx, `
		INSERT INTO policies (namespace_id, version, document, is_active)
		VALUES ($1, $2, $3, true)
		RETURNING id, namespace_id, version, document, is_active, created_at
	`, namespaceID, nextVersion, document).Scan(
		&p.ID, &p.NamespaceID, &p.Version, &p.Document, &p.IsActive, &p.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert policy: %w", err)
	}

	payload, _ := json.Marshal(map[string]any{
		"policy_id": p.ID,
		"version":   p.Version,
	})
	_, err = tx.Exec(ctx, `
		INSERT INTO audit_events (namespace_id, event_type, actor, resource_type, resource_id, payload)
		VALUES ($1, 'policy.updated', $2, 'policy', $3, $4)
	`, namespaceID, actor, fmt.Sprintf("%d", p.ID), payload)
	if err != nil {
		return nil, fmt.Errorf("audit policy update: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}
	return &p, nil
}

// GetActivePolicy returns the active policy for a namespace.
func (s *Store) GetActivePolicy(ctx context.Context, namespaceID int64) (*Policy, error) {
	var p Policy
	err := s.pool.QueryRow(ctx, `
		SELECT id, namespace_id, version, document, is_active, created_at
		FROM policies WHERE namespace_id = $1 AND is_active = true
		ORDER BY version DESC LIMIT 1
	`, namespaceID).Scan(&p.ID, &p.NamespaceID, &p.Version, &p.Document, &p.IsActive, &p.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get active policy: %w", err)
	}
	return &p, nil
}

// RecordAuditEvent appends an audit log entry.
func (s *Store) RecordAuditEvent(ctx context.Context, namespaceID *int64, eventType string, actor *string, resourceType, resourceID *string, payload json.RawMessage) (*AuditEvent, error) {
	if payload == nil {
		payload = json.RawMessage(`{}`)
	}
	var e AuditEvent
	err := s.pool.QueryRow(ctx, `
		INSERT INTO audit_events (namespace_id, event_type, actor, resource_type, resource_id, payload)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, namespace_id, event_type, actor, resource_type, resource_id, payload, created_at
	`, namespaceID, eventType, actor, resourceType, resourceID, payload).Scan(
		&e.ID, &e.NamespaceID, &e.EventType, &e.Actor, &e.ResourceType, &e.ResourceID, &e.Payload, &e.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("record audit event: %w", err)
	}
	return &e, nil
}

// ListAuditEvents returns audit events for a namespace.
func (s *Store) ListAuditEvents(ctx context.Context, namespaceID int64, limit int) ([]AuditEvent, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, namespace_id, event_type, actor, resource_type, resource_id, payload, created_at
		FROM audit_events WHERE namespace_id = $1
		ORDER BY created_at DESC LIMIT $2
	`, namespaceID, limit)
	if err != nil {
		return nil, fmt.Errorf("list audit events: %w", err)
	}
	defer rows.Close()

	var events []AuditEvent
	for rows.Next() {
		var e AuditEvent
		if err := rows.Scan(&e.ID, &e.NamespaceID, &e.EventType, &e.Actor, &e.ResourceType, &e.ResourceID, &e.Payload, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan audit event: %w", err)
		}
		events = append(events, e)
	}
	return events, rows.Err()
}
