-- +goose Up
ALTER TABLE attestations
    ADD COLUMN envelope_json JSONB;

ALTER TABLE attestations
    DROP CONSTRAINT attestations_envelope_present;

ALTER TABLE attestations
    ADD CONSTRAINT attestations_envelope_present CHECK (
        envelope_ref IS NOT NULL
        OR envelope_digest IS NOT NULL
        OR envelope_json IS NOT NULL
    );

CREATE TABLE provenance_records (
    id              BIGSERIAL PRIMARY KEY,
    attestation_id  BIGINT NOT NULL REFERENCES attestations (id) ON DELETE CASCADE,
    digest_id       BIGINT NOT NULL REFERENCES digests (id) ON DELETE CASCADE,
    repository_uri  TEXT NOT NULL,
    commit_sha      TEXT,
    workflow_name   TEXT,
    workflow_ref    TEXT,
    run_id          TEXT,
    verified        BOOLEAN NOT NULL DEFAULT false,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX provenance_records_digest_id_idx ON provenance_records (digest_id);
CREATE INDEX provenance_records_commit_sha_idx ON provenance_records (commit_sha)
    WHERE commit_sha IS NOT NULL;

INSERT INTO verity_meta (key, value)
VALUES ('schema_provenance', 'm04')
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value;

-- +goose Down
DROP TABLE IF EXISTS provenance_records;

ALTER TABLE attestations
    DROP CONSTRAINT attestations_envelope_present;

ALTER TABLE attestations
    ADD CONSTRAINT attestations_envelope_present CHECK (
        envelope_ref IS NOT NULL OR envelope_digest IS NOT NULL
    );

ALTER TABLE attestations
    DROP COLUMN IF EXISTS envelope_json;

DELETE FROM verity_meta WHERE key = 'schema_provenance';
