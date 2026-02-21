-- +migrate Down
-- Drop saved_connections table
DROP INDEX IF EXISTS idx_saved_connections_user_id_name;
DROP INDEX IF EXISTS idx_saved_connections_user_id;
DROP TABLE IF EXISTS saved_connections;
