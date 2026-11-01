
ALTER TABLE refresh_tokens
ADD COLUMN rotated_at TIMESTAMPTZ;

CREATE INDEX idx_refresh_tokens_active_family
ON refresh_tokens (family_id)
WHERE revoked_at IS NULL;

