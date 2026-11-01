
CREATE TABLE audit_events (
    id BIGSERIAL PRIMARY KEY,
    actor_user_id BIGINT NULL REFERENCES users(id) ON DELETE SET NULL,
    event_type VARCHAR(100) NOT NULL,
    outcome VARCHAR(20) NOT NULL
        CHECK (outcome IN ('success', 'failure', 'denied')),
    request_id VARCHAR(100),
    http_method VARCHAR(10),
    request_path VARCHAR(500),
    status_code SMALLINT,
    client_ip INET,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_events_created_at
    ON audit_events (created_at DESC);

CREATE INDEX idx_audit_events_actor_created
    ON audit_events (actor_user_id, created_at DESC);

CREATE INDEX idx_audit_events_type_created
    ON audit_events (event_type, created_at DESC);
