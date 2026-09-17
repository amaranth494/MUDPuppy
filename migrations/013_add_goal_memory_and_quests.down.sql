-- +migrate Down
-- Drop the quests table first (child of nothing, but reactivation logic
-- depends on it existing alone), then the two new columns, mirroring
-- migration 011's child-before-parent drop order.
DROP TABLE IF EXISTS quests;

ALTER TABLE game_sessions
DROP COLUMN IF EXISTS session_memory;

ALTER TABLE profiles
DROP COLUMN IF EXISTS session_goal;
