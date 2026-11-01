ALTER TABLE api_clients
DROP CONSTRAINT IF EXISTS chk_api_clients_rate_limit;

ALTER TABLE api_clients
DROP COLUMN IF EXISTS rate_window_seconds,
DROP COLUMN IF EXISTS rate_limit;
