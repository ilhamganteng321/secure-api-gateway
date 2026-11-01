CREATE TABLE api_clients (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(120) NOT NULL,
    description TEXT,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE api_keys (
    id BIGSERIAL PRIMARY KEY,
    client_id BIGINT NOT NULL REFERENCES api_clients(id)
        ON DELETE CASCADE,

    key_id VARCHAR(32) NOT NULL UNIQUE,
    key_hash BYTEA NOT NULL,

    scopes TEXT[] NOT NULL DEFAULT '{}',

    expires_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    last_used_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_api_keys_client_id
    ON api_keys(client_id);

CREATE INDEX idx_api_keys_key_id
    ON api_keys(key_id);
