DROP INDEX IF EXISTS idx_refresh_tokens_active_family;

ALTER TABLE refresh_tokens
DROP COLUMN IF EXISTS rotated_at;
