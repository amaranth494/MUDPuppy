-- +migrate Up
-- Migrate existing connections to have default profiles
-- This ensures all existing connections have associated profiles

-- Insert profile rows for all existing connections that don't have profiles
INSERT INTO profiles (id, user_id, connection_id, keybindings, settings, created_at, updated_at)
SELECT 
    gen_random_uuid(),
    user_id,
    id,
    '{}'::jsonb,
    '{"scrollback_limit": 1000, "echo_input": false, "timestamp_output": false, "word_wrap": true}'::jsonb,
    NOW(),
    NOW()
FROM saved_connections
WHERE id NOT IN (SELECT connection_id FROM profiles WHERE connection_id IS NOT NULL);
