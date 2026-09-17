-- +migrate Down
ALTER TABLE users
DROP COLUMN IF EXISTS login_started_at;
