-- +migrate Down
-- Drop connection_credentials table
DROP INDEX IF EXISTS idx_connection_credentials_connection_id;
DROP TABLE IF EXISTS connection_credentials;
