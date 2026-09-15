-- +migrate Down
-- Remove AI Player fields from profiles table
ALTER TABLE profiles
DROP COLUMN IF EXISTS conduct_rules,
DROP COLUMN IF EXISTS approach_guidance,
DROP COLUMN IF EXISTS ai_settings,
DROP COLUMN IF EXISTS policy_version_accepted,
DROP COLUMN IF EXISTS policy_accepted_at;
