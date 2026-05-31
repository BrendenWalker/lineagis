-- +goose Up
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS lineagis_meta (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

INSERT INTO lineagis_meta (key, value)
VALUES ('schema_bootstrap', 'm02')
ON CONFLICT (key) DO NOTHING;

-- +goose Down
DROP TABLE IF EXISTS lineagis_meta;
