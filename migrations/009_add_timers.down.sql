-- +migrate Down
-- Remove timers column from profiles table
ALTER TABLE profiles
DROP COLUMN IF EXISTS timers;
