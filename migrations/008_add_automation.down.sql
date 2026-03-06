-- +migrate Down
-- Remove automation fields from profiles table
ALTER TABLE profiles
DROP COLUMN IF EXISTS aliases,
DROP COLUMN IF EXISTS triggers,
DROP COLUMN IF EXISTS variables;
