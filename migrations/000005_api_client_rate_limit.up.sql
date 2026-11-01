ALTER TABLE api_clients
ADD COLUMN rate_limit INTEGER NOT NULL DEFAULT 60,
ADD COLUMN rate_window_seconds INTEGER NOT NULL DEFAULT 60;

ALTER TABLE api_clients
ADD CONSTRAINT chk_api_clients_rate_limit
CHECK (rate_limit > 0 AND rate_window_seconds > 0);
