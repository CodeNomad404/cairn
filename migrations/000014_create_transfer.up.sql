CREATE TABLE transfer_channels (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id),
    name TEXT NOT NULL CHECK (char_length(name) BETWEEN 1 AND 50 AND name = btrim(name)),
    deleted BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX transfer_channels_owner_name ON transfer_channels(user_id, name) WHERE NOT deleted;
CREATE INDEX transfer_channels_owner ON transfer_channels(user_id, created_at, id) WHERE NOT deleted;

CREATE TABLE transfer_items (
    id BIGSERIAL PRIMARY KEY,
    channel_id BIGINT NOT NULL REFERENCES transfer_channels(id),
    kind TEXT NOT NULL CHECK (kind IN ('text', 'file')),
    body TEXT NOT NULL DEFAULT '',
    storage_key TEXT NOT NULL DEFAULT '',
    filename TEXT NOT NULL DEFAULT '',
    size_bytes BIGINT NOT NULL CHECK (size_bytes >= 0),
    state TEXT NOT NULL CHECK (state IN ('pending', 'active', 'deleting')),
    cleanup_attempted_at TIMESTAMPTZ NOT NULL DEFAULT '-infinity',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK ((kind = 'text' AND storage_key = '' AND filename = '' AND size_bytes = octet_length(body))
        OR (kind = 'file' AND body = '' AND storage_key ~ '^[a-f0-9]{32}$' AND filename <> ''))
);
CREATE UNIQUE INDEX transfer_items_storage_key ON transfer_items(storage_key) WHERE kind = 'file';
CREATE INDEX transfer_items_channel ON transfer_items(channel_id, created_at DESC, id DESC) WHERE state = 'active';
CREATE INDEX transfer_items_cleanup ON transfer_items(state, created_at) WHERE state <> 'active';
CREATE INDEX transfer_items_cleanup_retry ON transfer_items(cleanup_attempted_at, id) WHERE state = 'deleting';
CREATE INDEX transfer_items_expiry ON transfer_items(created_at) WHERE state = 'active';
