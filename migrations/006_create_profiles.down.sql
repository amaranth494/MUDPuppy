-- +migrate Down
-- Drop profiles table

-- Drop indexes first
DROP INDEX IF EXISTS idx_profiles_user_id;
DROP INDEX IF EXISTS idx_profiles_connection_id;

-- Drop table
DROP TABLE IF EXISTS profiles;
