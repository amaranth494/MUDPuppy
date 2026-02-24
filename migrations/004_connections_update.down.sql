-- +migrate Down
-- Remove last_connected_at index
DROP INDEX IF EXISTS idx_saved_connections_last_connected_at;

-- Remove columns from saved_connections table
ALTER TABLE saved_connections DROP COLUMN IF EXISTS protocol;
ALTER TABLE saved_connections DROP COLUMN IF EXISTS last_connected_at;
