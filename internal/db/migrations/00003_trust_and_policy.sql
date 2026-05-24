-- +goose Up
CREATE TABLE signatures (
    id          BIGSERIAL PRIMARY KEY,
    digest_id   BIGINT NOT NULL REFERENCES digests (id) ON DELETE CASCADE,
    bundle_ref  TEXT,
    bundle_json JSONB,
    issuer      TEXT,
    subject     TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT signatures_bundle_present CHECK (
        bundle_ref IS NOT NULL OR bundle_json IS NOT NULL
    )
);

CREATE INDEX signatures_digest_id_idx ON signatures (digest_id);

CREATE TABLE attestations (
    id              BIGSERIAL PRIMARY KEY,
    digest_id       BIGINT NOT NULL REFERENCES digests (id) ON DELETE CASCADE,
    predicate_type  TEXT NOT NULL,
    envelope_ref    TEXT,
    envelope_digest TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT attestations_envelope_present CHECK (
        envelope_ref IS NOT NULL OR envelope_digest IS NOT NULL
    )
);

CREATE INDEX attestations_digest_id_idx ON attestations (digest_id);

CREATE TABLE policies (
    id           BIGSERIAL PRIMARY KEY,
    namespace_id BIGINT NOT NULL REFERENCES namespaces (id) ON DELETE CASCADE,
    version      INT NOT NULL,
    document     JSONB NOT NULL,
    is_active    BOOLEAN NOT NULL DEFAULT false,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (namespace_id, version)
);

CREATE INDEX policies_namespace_active_idx ON policies (namespace_id, is_active)
    WHERE is_active = true;

CREATE TABLE policy_decisions (
    id         BIGSERIAL PRIMARY KEY,
    digest_id  BIGINT NOT NULL REFERENCES digests (id) ON DELETE CASCADE,
    policy_id  BIGINT NOT NULL REFERENCES policies (id) ON DELETE CASCADE,
    outcome    TEXT NOT NULL CHECK (outcome IN ('pass', 'fail', 'warn')),
    reasons    JSONB NOT NULL DEFAULT '[]',
    evaluated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX policy_decisions_digest_id_idx ON policy_decisions (digest_id);

CREATE TABLE audit_events (
    id            BIGSERIAL PRIMARY KEY,
    namespace_id  BIGINT REFERENCES namespaces (id) ON DELETE SET NULL,
    event_type    TEXT NOT NULL,
    actor         TEXT,
    resource_type TEXT,
    resource_id   TEXT,
    payload       JSONB NOT NULL DEFAULT '{}',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX audit_events_namespace_created_idx ON audit_events (namespace_id, created_at DESC);

INSERT INTO verity_meta (key, value)
VALUES ('schema_trust', 'm03')
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value;

-- +goose Down
DROP TABLE IF EXISTS audit_events;
DROP TABLE IF EXISTS policy_decisions;
DROP TABLE IF EXISTS policies;
DROP TABLE IF EXISTS attestations;
DROP TABLE IF EXISTS signatures;

DELETE FROM verity_meta WHERE key = 'schema_trust';
