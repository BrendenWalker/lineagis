-- +goose Up
CREATE TABLE namespaces (
    id         BIGSERIAL PRIMARY KEY,
    name       TEXT NOT NULL UNIQUE,
    config     JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE artifacts (
    id           BIGSERIAL PRIMARY KEY,
    namespace_id BIGINT NOT NULL REFERENCES namespaces (id) ON DELETE CASCADE,
    name         TEXT NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (namespace_id, name)
);

CREATE INDEX artifacts_namespace_id_idx ON artifacts (namespace_id);

CREATE TABLE digests (
    id          BIGSERIAL PRIMARY KEY,
    digest      TEXT NOT NULL UNIQUE,
    artifact_id BIGINT NOT NULL REFERENCES artifacts (id) ON DELETE CASCADE,
    media_type  TEXT,
    size_bytes  BIGINT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX digests_artifact_id_idx ON digests (artifact_id);
CREATE INDEX digests_digest_idx ON digests (digest);

CREATE TABLE tags (
    id         BIGSERIAL PRIMARY KEY,
    artifact_id BIGINT NOT NULL REFERENCES artifacts (id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    digest_id  BIGINT NOT NULL REFERENCES digests (id),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (artifact_id, name)
);

CREATE INDEX tags_artifact_id_name_idx ON tags (artifact_id, name);
CREATE INDEX tags_digest_id_idx ON tags (digest_id);

CREATE TABLE tag_events (
    id              BIGSERIAL PRIMARY KEY,
    tag_id          BIGINT NOT NULL REFERENCES tags (id) ON DELETE CASCADE,
    from_digest_id  BIGINT REFERENCES digests (id),
    to_digest_id    BIGINT NOT NULL REFERENCES digests (id),
    actor           TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX tag_events_tag_id_idx ON tag_events (tag_id);
CREATE INDEX tag_events_created_at_idx ON tag_events (created_at);

INSERT INTO verity_meta (key, value)
VALUES ('schema_core', 'm03')
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value;

-- +goose Down
DROP TABLE IF EXISTS tag_events;
DROP TABLE IF EXISTS tags;
DROP TABLE IF EXISTS digests;
DROP TABLE IF EXISTS artifacts;
DROP TABLE IF EXISTS namespaces;

DELETE FROM verity_meta WHERE key = 'schema_core';
