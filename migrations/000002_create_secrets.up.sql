CREATE TABLE IF NOT EXISTS blobs (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    original_name TEXT NOT NULL,
    storage_name TEXT NOT NULL,
    storage_path TEXT NOT NULL,
    content_type TEXT NOT NULL,
    size BIGINT NOT NULL,
    checksum_sha256 TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    deleted_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS secrets (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    type TEXT NOT NULL CHECK (type IN ('credentials', 'card', 'text', 'binary')),
    name TEXT NOT NULL,
    metadata BYTEA NOT NULL,
    metadata_nonce BYTEA NOT NULL,
    payload BYTEA NOT NULL,
    payload_nonce BYTEA NOT NULL,
    blob_id UUID REFERENCES blobs (id),
    version INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    deleted_at TIMESTAMPTZ
);

ALTER TABLE secrets DROP CONSTRAINT IF EXISTS secrets_type_check;
ALTER TABLE secrets ADD CONSTRAINT secrets_type_check CHECK (type IN ('credentials', 'card', 'text', 'binary'));
ALTER TABLE secrets ADD COLUMN IF NOT EXISTS blob_id UUID REFERENCES blobs (id);

CREATE INDEX IF NOT EXISTS secrets_user_id_idx ON secrets (user_id);
CREATE INDEX IF NOT EXISTS secrets_updated_at_idx ON secrets (updated_at);
CREATE INDEX IF NOT EXISTS secrets_blob_id_idx ON secrets (blob_id);
CREATE INDEX IF NOT EXISTS blobs_user_id_idx ON blobs (user_id);
