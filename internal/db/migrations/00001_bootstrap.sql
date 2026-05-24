-- +goose Up
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS verity_meta (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

INSERT INTO verity_meta (key, value)
VALUES ('schema_bootstrap', 'm02')
ON CONFLICT (key) DO NOTHING;

-- +goose Down
DROP TABLE IF EXISTS verity_meta;
